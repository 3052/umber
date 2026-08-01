package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "os"
   "path/filepath"
   "sort"
   "strings"
   "time"
)

// downloadFile fetches the audio stream and writes it to filename,
// logging progress once per second with elapsed time and ETA.
func downloadFile(url, filename string) error {
   resp, err := http.Get(url)
   if err != nil {
      return fmt.Errorf("download request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("download returned status %d", resp.StatusCode)
   }

   total := resp.ContentLength
   out, err := os.Create(filename)
   if err != nil {
      return fmt.Errorf("create file: %w", err)
   }
   defer out.Close()

   start := time.Now()
   lastLog := time.Now()
   buf := make([]byte, 32*1024)
   var downloaded int64

   for {
      n, err := resp.Body.Read(buf)
      if n > 0 {
         if _, werr := out.Write(buf[:n]); werr != nil {
            return fmt.Errorf("write file: %w", werr)
         }
         downloaded += int64(n)

         now := time.Now()
         if now.Sub(lastLog) >= time.Second {
            elapsed := now.Sub(start)
            eta := "unknown"
            if total > 0 && downloaded > 0 {
               speed := float64(downloaded) / elapsed.Seconds()
               if speed > 0 {
                  remaining := float64(total - downloaded)
                  if remaining < 0 {
                     remaining = 0
                  }
                  etaSec := remaining / speed
                  eta = formatDuration(time.Duration(etaSec * float64(time.Second)))
               }
            }
            log.Printf("%s  %s / %s  elapsed %s  eta %s",
               filepath.Base(filename),
               formatBytes(downloaded),
               formatBytes(total),
               formatDuration(elapsed),
               eta,
            )
            lastLog = now
         }
      }
      if err == io.EOF {
         break
      }
      if err != nil {
         return fmt.Errorf("read body: %w", err)
      }
   }

   log.Printf("%s  done  %s in %s", filepath.Base(filename), formatBytes(downloaded), formatDuration(time.Since(start)))
   return nil
}

// downloadVideo calls the YouTube Inner Player API, picks the audio stream,
// and saves it to the output directory named by video ID.
func downloadVideo(videoID, visitorID, outputDir string) error {
   payload := PlayerRequest{
      VideoId: videoID,
      Context: PlayerContext{
         Client: PlayerClient{
            ClientName:    "ANDROID_VR",
            ClientVersion: "1.65.10",
         },
      },
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return fmt.Errorf("marshal payload: %w", err)
   }

   req, err := http.NewRequest("POST", "https://www.youtube.com/youtubei/v1/player", bytes.NewReader(body))
   if err != nil {
      return fmt.Errorf("create request: %w", err)
   }

   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("X-Goog-Visitor-Id", visitorID)

   client := &http.Client{}
   resp, err := client.Do(req)
   if err != nil {
      return fmt.Errorf("api request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("api returned status %d", resp.StatusCode)
   }

   var player PlayerResponse
   if err := json.NewDecoder(resp.Body).Decode(&player); err != nil {
      return fmt.Errorf("decode player response: %w", err)
   }

   if player.PlayabilityStatus.Status != "OK" {
      return fmt.Errorf("playability: %s — %s", player.PlayabilityStatus.Status, player.PlayabilityStatus.Reason)
   }

   // Sort adaptiveFormats by bitrate descending (replicate JS logic).
   formats := player.StreamingData.AdaptiveFormats
   sort.Slice(formats, func(i, j int) bool {
      return formats[i].Bitrate > formats[j].Bitrate
   })

   var audioURL, mimeType string
   for _, f := range formats {
      if f.AudioQuality == "AUDIO_QUALITY_MEDIUM" {
         audioURL = f.URL
         mimeType = f.MimeType
         break
      }
   }

   if audioURL == "" {
      return fmt.Errorf("no AUDIO_QUALITY_MEDIUM format found")
   }

   ext := getExtension(mimeType)
   filename := filepath.Join(outputDir, videoID+ext)
   return downloadFile(audioURL, filename)
}

func formatBytes(b int64) string {
   if b < 0 {
      return "?"
   }
   if b < 1024 {
      return fmt.Sprintf("%d B", b)
   }
   const unit = 1024
   div, exp := int64(unit), 0
   for n := b / unit; n >= unit; n /= unit {
      div *= unit
      exp++
   }
   return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
   if d < 0 {
      return "?"
   }
   s := int(d.Seconds())
   if s < 60 {
      return fmt.Sprintf("%ds", s)
   }
   m := s / 60
   s %= 60
   if m < 60 {
      return fmt.Sprintf("%dm%02ds", m, s)
   }
   h := m / 60
   m %= 60
   return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
}

// getExtension derives a file extension from the MIME type.
func getExtension(mimeType string) string {
   parts := strings.Split(mimeType, ";")
   main := strings.TrimSpace(parts[0])
   switch main {
   case "audio/webm":
      return ".webm"
   case "audio/mp4":
      return ".m4a"
   case "audio/ogg":
      return ".ogg"
   default:
      return ".bin"
   }
}

type AdaptiveFormat struct {
   Bitrate      int    `json:"bitrate"`
   AudioQuality string `json:"audioQuality"`
   URL          string `json:"url"`
   MimeType     string `json:"mimeType"`
}

type PlayerClient struct {
   ClientName    string `json:"clientName"`
   ClientVersion string `json:"clientVersion"`
}

type PlayerContext struct {
   Client PlayerClient `json:"client"`
}

// PlayerRequest is the payload sent to the YouTube Inner Player API.
type PlayerRequest struct {
   VideoId string        `json:"videoId"`
   Context PlayerContext `json:"context"`
}

// PlayerResponse is the relevant subset of the API response.
type PlayerResponse struct {
   VideoDetails struct {
      Author string `json:"author"`
      Title  string `json:"title"`
   } `json:"videoDetails"`
   PlayabilityStatus struct {
      Status string `json:"status"`
      Reason string `json:"reason"`
   } `json:"playabilityStatus"`
   StreamingData struct {
      AdaptiveFormats []AdaptiveFormat `json:"adaptiveFormats"`
   } `json:"streamingData"`
}

// download.go
