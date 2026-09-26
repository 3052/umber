// util.go marker preserve
package main

import (
   "context"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
   "strings"
   "sync"
   "time"
   "unicode/utf8"
)

var errETASkipped = fmt.Errorf("skipped due to ETA")

// astralRune reports whether r is outside the Basic Multilingual Plane.
// HiBy players fail to open files whose names contain such runes (emoji,
// "fancy text" letters), reporting "playback failed file not found".
// BMP-only names are never rewritten.
func astralRune(r rune) bool {
   return r > 0xFFFF
}

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

// fixAstralRunes rewrites s (which must contain astral runes) into a
// HiBy-safe string: mathematical letters/digits become ASCII, every other
// astral rune (emoji, flags, ...) is dropped, along with the zero-width
// emoji joiners they leave dangling. All other runes pass through as-is.
func fixAstralRunes(s string) string {
   var b strings.Builder
   b.Grow(len(s))
   for _, r := range s {
      switch {
      case r <= 0xFFFF:
         if r == 0x200D || r == 0xFE0F {
            continue
         }
         b.WriteRune(r)
      default:
         if a, ok := mathAlnumASCII(r); ok {
            b.WriteRune(a)
         }
      }
   }
   return b.String()
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

// isTempFile reports whether name is one of this program's in-flight
// temp files: <stem>.tmp (raw YouTube stream), <stem>.t (Bandcamp /
// SoundCloud stream), or <stem>.remux.<ext> (FFmpeg remux output). The
// remux temp ends in the real extension so FFmpeg picks the muxer from
// the name; its marker ends in a dot, and since sanitizeFilename
// strips trailing dots from stems, a final file can never land on a
// temp name.
func isTempFile(name string) bool {
   if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".t") {
      return true
   }
   if i := strings.LastIndex(name, "."); i >= 0 {
      return strings.HasSuffix(name[:i], ".remux.")
   }
   return false
}

// mathAlnumASCII maps Mathematical Alphanumeric Symbols (U+1D400–U+1D7FF,
// the 𝗯𝗼𝗹𝗱 / 𝘀𝗰𝗿𝗶𝗽𝘁 / 𝟭𝟮𝟯 code points emitted by fancy-text generators)
// to their ASCII equivalents, matching their Unicode NFKC decompositions.
// ok is false for runes with no ASCII equivalent.
func mathAlnumASCII(r rune) (ascii rune, ok bool) {
   if r >= 0x1D400 && r <= 0x1D6A3 { // Latin letter styles, all 26+26 runs
      off := (r - 0x1D400) % 52
      if off < 26 {
         return 'A' + off, true
      }
      return 'a' + off - 26, true
   }
   if r >= 0x1D7CE && r <= 0x1D7FF { // digit styles
      return '0' + (r-0x1D7CE)%10, true
   }
   return 0, false
}

// recordID returns the URL-derived identifier used to disambiguate records
// whose filename stems collide: the YouTube video ID, or the final path
// segment of a Bandcamp URL (e.g. "waiting" in .../track/waiting) or a
// SoundCloud URL (e.g. "flickermood" in .../forss/flickermood). p must be
// the record's platform as classified by platformOf.
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

// sanitizeFilename sanitizes a title for use as a filename, then truncates
// the result so that name+ext fits within both the NTFS component limit
// (255 chars) and the Windows MAX_PATH limit (259 usable chars). The
// extension and output directory determine the per-file cap. BMP-only
// names pass through unchanged; only names containing astral runes are
// rewritten first.
func sanitizeFilename(s string, ext string, outputDir string) string {
   if strings.ContainsFunc(s, astralRune) {
      s = fixAstralRunes(s)
   }
   invalid := `\/:*?"<>|`
   var b strings.Builder
   for _, c := range s {
      if strings.ContainsRune(invalid, c) {
         b.WriteByte('_')
      } else {
         b.WriteRune(c)
      }
   }
   result := strings.TrimRight(b.String(), ". ")

   capComponent := 255 - len(ext)
   capPath := 259 - len(outputDir) - 1 - len(ext)
   cap := capComponent
   if capPath < cap {
      cap = capPath
   }
   if cap < 1 {
      cap = 1
   }

   if len(result) > cap {
      result = result[:cap]
      for !utf8.ValidString(result) {
         result = result[:len(result)-1]
      }
      result = strings.TrimRight(result, ". ")
   }
   return result
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

// util.go marker preserve
