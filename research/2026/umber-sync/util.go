package main

import (
   "fmt"
   "strings"
   "unicode/utf8"
)

const visitorExpiredReason = "This content isn't available, try again later."

var errETASkipped = fmt.Errorf("skipped due to ETA")

var errVisitorExpired = fmt.Errorf("visitor ID expired")

// astralRune reports whether r is outside the Basic Multilingual Plane.
// HiBy players fail to open files whose names contain such runes (emoji,
// "fancy text" letters), reporting "playback failed file not found".
// BMP-only names are never rewritten.
func astralRune(r rune) bool {
   return r > 0xFFFF
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

// getFFmpegFormat returns the FFmpeg format name (-f) for the output file
// based on the MIME type. This is needed because the temp file extension
// (.tmp) is not recognized by FFmpeg's format auto-detection.
func getFFmpegFormat(mimeType string) string {
   parts := strings.Split(mimeType, ";")
   main := strings.TrimSpace(parts[0])
   switch main {
   case "audio/webm":
      return "ogg"
   case "audio/mp4":
      return "ipod"
   default:
      return "ipod"
   }
}

// getOutputExt returns the file extension for the FFmpeg remuxed output based
// on the container format from the MIME type. Only .opus and .m4a are
// produced.
func getOutputExt(mimeType string) string {
   parts := strings.Split(mimeType, ";")
   main := strings.TrimSpace(parts[0])
   switch main {
   case "audio/webm":
      return ".opus"
   case "audio/mp4":
      return ".m4a"
   default:
      return ".m4a"
   }
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

type AdaptiveFormat struct {
   Bitrate      int    `json:"bitrate"`
   AudioQuality string `json:"audioQuality"`
   URL          string `json:"url"`
   MimeType     string `json:"mimeType"`
}

type PlaybackContext struct {
   ContentPlaybackContext struct {
      Html5Preference    string `json:"html5Preference,omitempty"`
      SignatureTimestamp int    `json:"signatureTimestamp,omitempty"`
   } `json:"contentPlaybackContext"`
}

type PlayerClient struct {
   ClientName       string `json:"clientName"`
   ClientVersion    string `json:"clientVersion"`
   DeviceMake       string `json:"deviceMake,omitempty"`
   DeviceModel      string `json:"deviceModel,omitempty"`
   UserAgent        string `json:"userAgent,omitempty"`
   OsName           string `json:"osName,omitempty"`
   OsVersion        string `json:"osVersion,omitempty"`
   Hl               string `json:"hl,omitempty"`
   TimeZone         string `json:"timeZone,omitempty"`
   UtcOffsetMinutes int    `json:"utcOffsetMinutes"`
}

type PlayerContext struct {
   Client PlayerClient `json:"client"`
}

type PlayerRequest struct {
   VideoId         string          `json:"videoId"`
   Context         PlayerContext   `json:"context"`
   PlaybackContext PlaybackContext `json:"playbackContext"`
   ContentCheckOk  bool            `json:"contentCheckOk"`
   RacyCheckOk     bool            `json:"racyCheckOk"`
}

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
      AdaptiveFormats []*AdaptiveFormat `json:"adaptiveFormats"`
      HlsManifestURL  string            `json:"hlsManifestUrl"`
   } `json:"streamingData"`
}

// util.go
