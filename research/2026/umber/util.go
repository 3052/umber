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

// getSourceExt returns the file extension for the downloaded file based on
// the container format from the MIME type.
func getSourceExt(mimeType string) string {
   parts := strings.Split(mimeType, ";")
   main := strings.TrimSpace(parts[0])
   switch main {
   case "audio/webm":
      return ".webm"
   case "audio/mp4":
      return ".m4a"
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
   VideoId         string           `json:"videoId"`
   Context         PlayerContext    `json:"context"`
   PlaybackContext *PlaybackContext `json:"playbackContext,omitempty"`
   ContentCheckOk  bool             `json:"contentCheckOk"`
   RacyCheckOk     bool             `json:"racyCheckOk"`
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
