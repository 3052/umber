package main

import (
   "bytes"
   "context"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "os"
   "path/filepath"
   "sort"
   "strings"
   "sync"
   "time"
)

// visitorExpiredReason is the specific reason text that indicates the
// visitor ID has expired rather than the video being unavailable.
const visitorExpiredReason = "This content isn't available, try again later."

// errETASkipped is returned when the ETA exceeds the maximum allowed.
var errETASkipped = fmt.Errorf("skipped due to ETA")

// errVisitorExpired is returned when the visitor ID has expired and needs refresh.
var errVisitorExpired = fmt.Errorf("visitor ID expired")

// downloadFile fetches the audio stream using multiple parallel connections
// to bypass YouTube's 1x speed throttle, logging progress once per second.
// If the ETA exceeds maxETA, the download is cancelled.
func downloadFile(url, filename string, threads int, maxETA time.Duration) error {
   // Probe with a 1-byte range request to get total size.
   probeReq, err := http.NewRequest("GET", url, nil)
   if err != nil {
      return fmt.Errorf("create probe request: %w", err)
   }
   probeReq.Header.Set("Range", "bytes=0-0")

   probeResp, err := http.DefaultClient.Do(probeReq)
   if err != nil {
      return fmt.Errorf("probe request: %w", err)
   }

   contentRange := probeResp.Header.Get("Content-Range")
   io.Copy(io.Discard, probeResp.Body)
   probeResp.Body.Close()

   // If server doesn't support Range, fall back to single-threaded download.
   if contentRange == "" {
      return downloadFileSingle(url, filename, maxETA)
   }

   // Parse total size from "bytes 0-0/1234567"
   parts := strings.Split(contentRange, "/")
   if len(parts) != 2 {
      return downloadFileSingle(url, filename, maxETA)
   }
   var total int64
   if _, err := fmt.Sscanf(parts[1], "%d", &total); err != nil {
      return downloadFileSingle(url, filename, maxETA)
   }

   chunkSize := (total + int64(threads) - 1) / int64(threads) // ceil division

   type result struct {
      data []byte
      err  error
   }
   results := make([]result, threads)
   var wg sync.WaitGroup

   ctx, cancel := context.WithCancel(context.Background())
   defer cancel()

   start := time.Now()
   lastLog := time.Now()
   var downloaded int64
   var mu sync.Mutex
   var skipped bool

   logProgress := func() {
      now := time.Now()
      if now.Sub(lastLog) < time.Second {
         return
      }
      elapsed := now.Sub(start).Round(time.Millisecond)
      var etaDuration time.Duration
      if downloaded > 0 {
         speed := float64(downloaded) / elapsed.Seconds()
         if speed > 0 {
            remaining := float64(total - downloaded)
            if remaining < 0 {
               remaining = 0
            }
            etaSec := remaining / speed
            etaDuration = time.Duration(etaSec * float64(time.Second))
         }
      }
      etaStr := "unknown"
      if etaDuration > 0 {
         etaStr = etaDuration.Round(time.Millisecond).String()
      }
      log.Printf("%s  %s / %s  elapsed %s  eta %s",
         filepath.Base(filename),
         formatBytes(downloaded),
         formatBytes(total),
         elapsed.String(),
         etaStr,
      )
      lastLog = now

      // Check ETA — only after 2 seconds to avoid false positives at start
      if etaDuration > maxETA && now.Sub(start) > 2*time.Second {
         skipped = true
         cancel()
      }
   }

   for i := 0; i < threads; i++ {
      wg.Add(1)
      go func(idx int) {
         defer wg.Done()
         startByte := int64(idx) * chunkSize
         endByte := startByte + chunkSize - 1
         if endByte > total-1 {
            endByte = total - 1
         }
         if startByte > endByte {
            return // no work for this thread
         }

         chunkReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
         if err != nil {
            results[idx].err = err
            return
         }
         chunkReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))

         chunkResp, err := http.DefaultClient.Do(chunkReq)
         if err != nil {
            results[idx].err = err
            return
         }
         defer chunkResp.Body.Close()

         if chunkResp.StatusCode != http.StatusOK && chunkResp.StatusCode != http.StatusPartialContent {
            results[idx].err = fmt.Errorf("chunk %d returned status %d", idx, chunkResp.StatusCode)
            return
         }

         buf := make([]byte, 32*1024)
         for {
            n, rerr := chunkResp.Body.Read(buf)
            if n > 0 {
               results[idx].data = append(results[idx].data, buf[:n]...)
               mu.Lock()
               downloaded += int64(n)
               logProgress()
               mu.Unlock()
            }
            if rerr == io.EOF {
               break
            }
            if rerr != nil {
               results[idx].err = rerr
               return
            }
         }
      }(i)
   }

   wg.Wait()

   if skipped {
      return fmt.Errorf("%w: ETA exceeds max %s", errETASkipped, maxETA)
   }

   // Check for errors
   for i := range results {
      if results[i].err != nil {
         return fmt.Errorf("thread %d: %w", i, results[i].err)
      }
   }

   // Reassemble in order
   out, err := os.Create(filename)
   if err != nil {
      return fmt.Errorf("create file: %w", err)
   }
   defer out.Close()

   for i := range results {
      if len(results[i].data) > 0 {
         if _, err := out.Write(results[i].data); err != nil {
            return fmt.Errorf("write file: %w", err)
         }
      }
   }

   log.Printf("%s  done  %s in %s", strings.TrimSuffix(filepath.Base(filename), ".tmp"), formatBytes(total), time.Since(start).Round(time.Millisecond).String())
   return nil
}

// downloadFileSingle is the fallback when Range requests aren't supported.
func downloadFileSingle(url, filename string, maxETA time.Duration) error {
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
            elapsed := now.Sub(start).Round(time.Millisecond)
            var etaDuration time.Duration
            if total > 0 && downloaded > 0 {
               speed := float64(downloaded) / elapsed.Seconds()
               if speed > 0 {
                  remaining := float64(total - downloaded)
                  if remaining < 0 {
                     remaining = 0
                  }
                  etaSec := remaining / speed
                  etaDuration = time.Duration(etaSec * float64(time.Second))
               }
            }
            etaStr := "unknown"
            if etaDuration > 0 {
               etaStr = etaDuration.Round(time.Millisecond).String()
            }
            log.Printf("%s  %s / %s  elapsed %s  eta %s",
               strings.TrimSuffix(filepath.Base(filename), ".tmp"),
               formatBytes(downloaded),
               formatBytes(total),
               elapsed.String(),
               etaStr,
            )
            lastLog = now

            // Check ETA — only after 2 seconds to avoid false positives at start
            if etaDuration > maxETA && now.Sub(start) > 2*time.Second {
               return fmt.Errorf("%w: ETA %s exceeds max %s", errETASkipped, etaDuration.Round(time.Millisecond), maxETA)
            }
         }
      }
      if err == io.EOF {
         break
      }
      if err != nil {
         return fmt.Errorf("read body: %w", err)
      }
   }

   log.Printf("%s  done  %s in %s", strings.TrimSuffix(filepath.Base(filename), ".tmp"), formatBytes(downloaded), time.Since(start).Round(time.Millisecond).String())
   return nil
}

// downloadVideo calls the YouTube Inner Player API, picks the audio stream,
// and saves it to the output directory named by the title.
func downloadVideo(videoID, title, visitorID, outputDir string, threads int, maxETA time.Duration) error {
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
      if player.PlayabilityStatus.Status == "UNPLAYABLE" && player.PlayabilityStatus.Reason == visitorExpiredReason {
         return fmt.Errorf("%w: %s — %s", errVisitorExpired, player.PlayabilityStatus.Status, player.PlayabilityStatus.Reason)
      }
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
   finalPath := filepath.Join(outputDir, sanitizeFilename(title)+ext)
   tmpPath := finalPath + ".tmp"
   if err := downloadFile(audioURL, tmpPath, threads, maxETA); err != nil {
      os.Remove(tmpPath)
      return err
   }
   return os.Rename(tmpPath, finalPath)
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

// sanitizeFilename replaces characters invalid in filenames with underscores.
func sanitizeFilename(s string) string {
   invalid := `\/:*?"<>|`
   var b strings.Builder
   for _, c := range s {
      if strings.ContainsRune(invalid, c) {
         b.WriteByte('_')
      } else {
         b.WriteRune(c)
      }
   }
   return strings.TrimRight(b.String(), ". ")
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
