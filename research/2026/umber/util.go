package main

import (
   "fmt"
   "strings"
)

const visitorExpiredReason = "This content isn't available, try again later."

var errETASkipped = fmt.Errorf("skipped due to ETA")

var errVisitorExpired = fmt.Errorf("visitor ID expired")

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

type PlayerRequest struct {
   VideoId string        `json:"videoId"`
   Context PlayerContext `json:"context"`
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
      AdaptiveFormats []AdaptiveFormat `json:"adaptiveFormats"`
   } `json:"streamingData"`
}

// util.go
