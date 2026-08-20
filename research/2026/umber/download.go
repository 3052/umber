package main

import (
   "bytes"
   "cmp"
   "context"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "os"
   "os/exec"
   "path/filepath"
   "slices"
   "strings"
   "sync"
   "time"
)

// VISIONOS client constants, matching what current yt-dlp sends (verified
// against a mitmproxy capture of yt-dlp 2026.08). The user agent is sent both
// in the client context and as the HTTP User-Agent header.
const (
   visionOSClientName    = "VISIONOS"
   visionOSClientVersion = "1.02"
   visionOSClientID      = "101"
   visionOSUserAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15"
)

func downloadFile(url, filename string, threads int, maxETA time.Duration) error {
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
   if _, err := io.Copy(io.Discard, probeResp.Body); err != nil {
      probeResp.Body.Close()
      return fmt.Errorf("drain probe body: %w", err)
   }
   if err := probeResp.Body.Close(); err != nil {
      return fmt.Errorf("close probe body: %w", err)
   }

   if contentRange == "" {
      return downloadFileSingle(url, filename, maxETA)
   }

   parts := strings.Split(contentRange, "/")
   if len(parts) != 2 {
      return downloadFileSingle(url, filename, maxETA)
   }
   var total int64
   if _, err := fmt.Sscanf(parts[1], "%d", &total); err != nil {
      return downloadFileSingle(url, filename, maxETA)
   }

   chunkSize := (total + int64(threads) - 1) / int64(threads)
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
            etaDuration = time.Duration(remaining / speed * float64(time.Second))
         }
      }
      etaStr := "unknown"
      if etaDuration > 0 {
         etaStr = etaDuration.Round(time.Millisecond).String()
      }
      log.Printf("%s  %s / %s  elapsed %s  eta %s",
         filepath.Base(filename), formatBytes(downloaded), formatBytes(total), elapsed.String(), etaStr)
      lastLog = now
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
            return
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
   for i := range results {
      if results[i].err != nil {
         return fmt.Errorf("thread %d: %w", i, results[i].err)
      }
   }

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
                  etaDuration = time.Duration(remaining / speed * float64(time.Second))
               }
            }
            etaStr := "unknown"
            if etaDuration > 0 {
               etaStr = etaDuration.Round(time.Millisecond).String()
            }
            log.Printf("%s  %s / %s  elapsed %s  eta %s",
               strings.TrimSuffix(filepath.Base(filename), ".tmp"), formatBytes(downloaded), formatBytes(total), elapsed.String(), etaStr)
            lastLog = now
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

func downloadVideo(videoID, title, visitorID, outputDir string, threads int, maxETA time.Duration) error {
   wc, err := fetchWatchConfig(videoID)
   if err != nil {
      return fmt.Errorf("watch config: %w", err)
   }
   if wc.VisitorData != "" {
      visitorID = string(wc.VisitorData)
   }

   sts, err := signatureTimestamp(wc.PlayerJSURL)
   if err != nil {
      return fmt.Errorf("signature timestamp: %w", err)
   }

   pc := &PlaybackContext{}
   pc.ContentPlaybackContext.Html5Preference = "HTML5_PREF_WANTS"
   pc.ContentPlaybackContext.SignatureTimestamp = sts

   payload := PlayerRequest{
      VideoId: videoID,
      Context: PlayerContext{
         Client: PlayerClient{
            ClientName:    visionOSClientName,
            ClientVersion: visionOSClientVersion,
            DeviceMake:    "Apple",
            DeviceModel:   "RealityDevice17,1",
            UserAgent:     visionOSUserAgent,
            OsName:        "visionOS",
            OsVersion:     "26.5.23O471",
            Hl:            "en",
            TimeZone:      "UTC",
         },
      },
      PlaybackContext: pc,
      ContentCheckOk:  true,
      RacyCheckOk:     true,
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return fmt.Errorf("marshal payload: %w", err)
   }

   req, err := http.NewRequest("POST", "https://www.youtube.com/youtubei/v1/player?prettyPrint=false", bytes.NewReader(body))
   if err != nil {
      return fmt.Errorf("create request: %w", err)
   }
   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("X-Goog-Visitor-Id", visitorID)
   req.Header.Set("X-Youtube-Client-Name", visionOSClientID)
   req.Header.Set("X-Youtube-Client-Version", visionOSClientVersion)
   req.Header.Set("User-Agent", visionOSUserAgent)
   req.Header.Set("Origin", "https://www.youtube.com")

   resp, err := http.DefaultClient.Do(req)
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
      if player.PlayabilityStatus.Status == "LOGIN_REQUIRED" {
         return fmt.Errorf("%w: %s — %s", errVisitorExpired, player.PlayabilityStatus.Status, player.PlayabilityStatus.Reason)
      }
      return fmt.Errorf("playability: %s — %s", player.PlayabilityStatus.Status, player.PlayabilityStatus.Reason)
   }

   formats := player.StreamingData.AdaptiveFormats
   slices.SortFunc(formats, func(a, b *AdaptiveFormat) int {
      return cmp.Compare(b.Bitrate, a.Bitrate)
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

   outExt := getOutputExt(mimeType)
   ffFormat := getFFmpegFormat(mimeType)
   name := sanitizeFilename(title, outExt, outputDir)
   finalPath := filepath.Join(outputDir, name+outExt)
   dlPath := filepath.Join(outputDir, name+".tmp")
   ffTmp := filepath.Join(outputDir, name+".ff")

   if err := downloadFile(audioURL, dlPath, threads, maxETA); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      return err
   }

   cmd := exec.Command("ffmpeg", "-y", "-i", dlPath, "-c", "copy", "-f", ffFormat, ffTmp)
   var stderr bytes.Buffer
   cmd.Stderr = &stderr
   if err := cmd.Run(); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp: %w", err)
      }
      return fmt.Errorf("ffmpeg remux: %w\n%s", err, stderr.String())
   }
   if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
      return fmt.Errorf("remove download tmp: %w", err)
   }
   if err := os.Rename(ffTmp, finalPath); err != nil {
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp after rename fail: %w", err)
      }
      return fmt.Errorf("rename file: %w", err)
   }

   log.Printf("%s  remuxed", filepath.Base(finalPath))
   return nil
}

// download.go
