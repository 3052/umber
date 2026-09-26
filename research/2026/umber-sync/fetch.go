// fetch.go marker preserve
package main

import (
   "bytes"
   "context"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path/filepath"
   "strings"
   "sync"
   "time"
)

var errETASkipped = fmt.Errorf("skipped due to ETA")

// downloadFile downloads url to filename, using a Range-request probe to
// split the transfer across threads parallel chunks when the server
// supports ranges, and falling back to downloadFileSingle otherwise.
// Items whose estimated time to completion exceeds maxETA are abandoned.
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

   for i := range threads {
      wg.Go(func() {
         startByte := int64(i) * chunkSize
         endByte := startByte + chunkSize - 1
         if endByte > total-1 {
            endByte = total - 1
         }
         if startByte > endByte {
            return
         }
         chunkReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
         if err != nil {
            results[i].err = err
            return
         }
         chunkReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))
         chunkResp, err := http.DefaultClient.Do(chunkReq)
         if err != nil {
            results[i].err = err
            return
         }
         defer chunkResp.Body.Close()
         if chunkResp.StatusCode != http.StatusOK && chunkResp.StatusCode != http.StatusPartialContent {
            results[i].err = fmt.Errorf("chunk %d returned status %d", i, chunkResp.StatusCode)
            return
         }
         buf := make([]byte, 32*1024)
         for {
            n, rerr := chunkResp.Body.Read(buf)
            if n > 0 {
               results[i].data = append(results[i].data, buf[:n]...)
               mu.Lock()
               downloaded += int64(n)
               logProgress()
               mu.Unlock()
            }
            if rerr == io.EOF {
               break
            }
            if rerr != nil {
               results[i].err = rerr
               return
            }
         }
      })
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

// downloadFileSingle streams url to filename in one request, the fallback
// for servers without range support. Items whose estimated time to
// completion exceeds maxETA are abandoned mid-transfer.
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

// recordID returns the URL-derived identifier used to disambiguate records
// whose filename stems collide: the YouTube video ID, or the final path
// segment of a Bandcamp URL (e.g. "waiting" in .../track/waiting) or a
// SoundCloud URL (e.g. "flickermood" in .../forss/flickermood). p must
// be the record's platform as classified by platformOf.
func recordID(p platform, raw string) (string, error) {
   switch p {
   case platformYouTube:
      id, err := videoIDFromURL(raw)
      if err != nil {
         return "", err
      }
      return id, nil
   case platformBandcamp, platformSoundCloud:
      u, err := url.Parse(raw)
      if err != nil {
         return "", fmt.Errorf("parse url: %w", err)
      }
      path := strings.Trim(u.Path, "/")
      if i := strings.LastIndex(path, "/"); i >= 0 {
         return path[i+1:], nil
      }
      return path, nil
   }
   return "", nil
}

// remuxTagged remuxes src to dst with ffmpeg, stream-copying the audio
// and tagging artist and title from the record's R and T values. mp3
// output gets ID3 tags; m4a/opus get their native tag formats. The
// muxer is picked from dst's extension, same contract as the downloaders'
// remux temps.
func remuxTagged(src, dst string, r Record) error {
   args := []string{"-i", src, "-c", "copy"}
   if r.R != "" {
      args = append(args, "-metadata", "artist="+r.R)
   }
   if r.T != "" {
      args = append(args, "-metadata", "title="+r.T)
   }
   args = append(args, dst)

   cmd := exec.Command("ffmpeg", args...)
   var stderr bytes.Buffer
   cmd.Stderr = &stderr
   if err := cmd.Run(); err != nil {
      return fmt.Errorf("ffmpeg tag: %w\n%s", err, stderr.String())
   }
   return nil
}

// platform identifies which service a record's I URL points at.
type platform int

const (
   platformBandcamp platform = iota
   platformYouTube
   platformSoundCloud
   platformOther
)

// platformOf classifies a record URL by host: bandcamp.com (or a
// *.bandcamp.com subdomain) is bandcamp, soundcloud.com (or a
// *.soundcloud.com subdomain) is soundcloud, youtube.com exactly is
// YouTube, and anything else is other. A URL that does not parse is an
// error; callers must not silently treat it as platformOther.
func platformOf(raw string) (platform, error) {
   u, err := url.Parse(raw)
   if err != nil {
      return platformOther, fmt.Errorf("parse url: %w", err)
   }
   host := strings.ToLower(u.Hostname())
   switch {
   case host == "bandcamp.com" || strings.HasSuffix(host, ".bandcamp.com"):
      return platformBandcamp, nil
   case host == "soundcloud.com" || strings.HasSuffix(host, ".soundcloud.com"):
      return platformSoundCloud, nil
   case host == "youtube.com":
      return platformYouTube, nil
   }
   return platformOther, nil
}

// fetch.go marker preserve
