// youtube.go marker preserve
package main

import (
   "bytes"
   "cmp"
   "encoding/json"
   "errors"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

// VISIONOS client constants, matching what current yt-dlp sends (verified
// against a mitmproxy capture of yt-dlp 2026.08). The simplified /player
// request sends only the name and version, in the context; no client-ID
// header, device fields, or user agent are needed.
const (
   visionOSClientName    = "VISIONOS"
   visionOSClientVersion = "1.02"
)

const sep = "\nytcfg.set("

// visitorExpiredReason is what YouTube answers when the visitor ID has
// gone stale but the request itself is fine.
const visitorExpiredReason = "This content isn't available, try again later."

// errRateLimited is returned when YouTube answers HTTP 429.
// errVisitorExpired is reported when the visitor ID has expired.
var (
   errRateLimited    = fmt.Errorf("rate limited")
   errVisitorExpired = fmt.Errorf("visitor ID expired")
)

// downloadYouTube downloads the AUDIO_QUALITY_MEDIUM audio stream for a
// YouTube watch URL (youtube.com host), remuxes it with ffmpeg, and tags
// artist and title from the record. The stream must be an audio/webm or
// audio/mp4 container — if the player response carries no medium format
// in either, that is an error. When several qualify, the preferred
// container wins: audio/webm, then audio/mp4. visitorID is the saved
// visitor ID from the config; the request is the simplified one the
// checker sends, so no watch page is fetched and no signature timestamp
// is needed. A stale visitor ID surfaces as errVisitorExpired, which run
// treats by clearing it and ending the run early. title is the record's
// filename stem, built by baseName: "author - title", except a YouTube
// " - Topic" author is stripped, and the title alone is used when it
// already contains the author.
func downloadYouTube(r *Record, title, visitorID, outputDir string, threads int, maxETA time.Duration) error {
   videoID, err := videoIDFromURL(r.I)
   if err != nil {
      return err
   }

   body, err := playerResponse(videoID, visitorID)
   if err != nil {
      return err
   }

   var player PlayerResponse
   if err := json.Unmarshal(body, &player); err != nil {
      return fmt.Errorf("decode player response: %w", err)
   }

   if ps := player.PlayabilityStatus; ps.Status != "OK" {
      switch {
      case ps.Status == "LOGIN_REQUIRED" && strings.Contains(ps.Reason, "not a bot"):
         return fmt.Errorf("%w: %s — %s", errVisitorExpired, ps.Status, ps.Reason)
      case ps.Status == "UNPLAYABLE" && ps.Reason == visitorExpiredReason:
         return fmt.Errorf("%w: %s — %s", errVisitorExpired, ps.Status, ps.Reason)
      }
      return fmt.Errorf("playability: %s — %s", ps.Status, ps.Reason)
   }

   medium, err := pickMediumFormat(player.StreamingData.AdaptiveFormats)
   if err != nil {
      return err
   }
   audioURL, mimeType := medium.URL, medium.MimeType

   outExt := getOutputExt(mimeType)
   name := sanitizeFilename(title, outputDir)
   finalPath := filepath.Join(outputDir, name+outExt)
   dlPath := filepath.Join(outputDir, name+".tmp")
   // The remux temp keeps the real extension so ffmpeg picks the muxer
   // from the output file name — no -f needed. The ".remux." marker
   // ends in a dot, which no sanitized stem can end in, so temps and
   // final files can never collide (see isTempFile).
   ffTmp := filepath.Join(outputDir, name+".remux."+outExt)

   if err := downloadFile(audioURL, dlPath, threads, maxETA); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      return err
   }

   if err := remuxTagged(dlPath, ffTmp, r); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp: %w", err)
      }
      return err
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

func extractJSON(content []byte, prefix []byte) ([]byte, error) {
   _, after, found := bytes.Cut(content, prefix)
   if !found {
      return nil, fmt.Errorf("prefix %q not found in file", prefix)
   }
   if len(after) == 0 {
      return nil, fmt.Errorf("content ends abruptly after prefix")
   }
   if after[0] != '{' {
      return nil, fmt.Errorf("expected '{' at the start of JSON, got %c", after[0])
   }
   openBraces := 0
   inString := false
   escapeNext := false
   for i, char := range after {
      if escapeNext {
         escapeNext = false
         continue
      }
      if char == '\\' {
         escapeNext = true
         continue
      }
      if char == '"' {
         inString = !inString
         continue
      }
      if !inString {
         if char == '{' {
            openBraces++
         } else if char == '}' {
            openBraces--
            if openBraces == 0 {
               return after[:i+1], nil
            }
         }
      }
   }
   return nil, fmt.Errorf("could not find the matching closing brace for the JSON object")
}

func fetchVisitorID() (string, error) {
   targetUrl := url.URL{Scheme: "https", Host: "www.youtube.com"}
   req := http.Request{
      Method: http.MethodGet,
      URL:    &targetUrl,
   }
   log.Println("fetching visitor ID from", req.URL)
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      return "", err
   }
   defer resp.Body.Close()

   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return "", err
   }

   data, err = extractJSON(data, []byte(sep))
   if err != nil {
      return "", err
   }

   var result ytCfg
   if err := json.Unmarshal(data, &result); err != nil {
      return "", err
   }

   return string(result.InnertubeContext.Client.VisitorData), nil
}

// getOutputExt returns the file extension for the FFmpeg remuxed output based
// on the container format from the MIME type. Only .opus and .m4a are
// produced. The extension is used both for the final file and for the remux
// temp, so FFmpeg picks the muxer from the output file name.
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

// mediumMimeRank ranks a stream's container for pickMediumFormat: 0 for
// audio/webm, 1 for audio/mp4, and ok false for anything else — those are
// not candidates. Only the MIME part before the semicolon counts; codec
// parameters such as codecs="opus" are ignored, the same split
// getOutputExt makes.
func mediumMimeRank(mimeType string) (rank int, ok bool) {
   main := strings.TrimSpace(strings.Split(mimeType, ";")[0])
   switch main {
   case "audio/webm":
      return 0, true
   case "audio/mp4":
      return 1, true
   }
   return 0, false
}

// playerResponse performs the simplified innertube /player request — the
// same one the checker sends: the context carries just the client name
// and version, and the only header beyond the defaults is the visitor ID.
// No signature timestamp, client-ID header, user agent, or content type
// is sent. It returns the raw response body for the caller to decode;
// HTTP 429 comes back wrapped in errRateLimited.
func playerResponse(videoID, visitorID string) ([]byte, error) {
   data, err := json.Marshal(map[string]any{
      "context": map[string]any{
         "client": map[string]any{
            "clientName":    visionOSClientName,
            "clientVersion": visionOSClientVersion,
         },
      },
      "videoId": videoID,
   })
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player?prettyPrint=false",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("X-Goog-Visitor-Id", visitorID)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode == http.StatusTooManyRequests {
      return nil, fmt.Errorf("%w: HTTP 429", errRateLimited)
   }
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   return io.ReadAll(resp.Body)
}

// videoIDFromURL extracts the video ID from a YouTube watch URL such as
// https://youtube.com/watch?v=dKJfJMMsqX4.
func videoIDFromURL(raw string) (string, error) {
   u, err := url.Parse(raw)
   if err != nil {
      return "", fmt.Errorf("parse url: %w", err)
   }
   if id := u.Query().Get("v"); id != "" {
      return id, nil
   }
   return "", fmt.Errorf("no video ID in %s", raw)
}

type AdaptiveFormat struct {
   Bitrate      int    `json:"bitrate"`
   AudioQuality string `json:"audioQuality"`
   URL          string `json:"url"`
   MimeType     string `json:"mimeType"`
}

// pickMediumFormat selects the audio stream to download from a player
// response's adaptive formats. A candidate must be AUDIO_QUALITY_MEDIUM,
// carry a URL, and use a supported container — audio/webm or audio/mp4;
// formats in any other container are ignored, and if none remains that
// is an error. With several candidates the preferred container wins:
// audio/webm, then audio/mp4. The sort is stable, so equally-ranked
// streams keep response order and the first one is picked.
func pickMediumFormat(formats []*AdaptiveFormat) (*AdaptiveFormat, error) {
   var medium []*AdaptiveFormat
   for _, f := range formats {
      if f.AudioQuality != "AUDIO_QUALITY_MEDIUM" || f.URL == "" {
         continue
      }
      if _, ok := mediumMimeRank(f.MimeType); ok {
         medium = append(medium, f)
      }
   }
   if len(medium) == 0 {
      return nil, fmt.Errorf("no AUDIO_QUALITY_MEDIUM audio/webm or audio/mp4 format found")
   }
   slices.SortStableFunc(medium, func(a, b *AdaptiveFormat) int {
      ra, _ := mediumMimeRank(a.MimeType)
      rb, _ := mediumMimeRank(b.MimeType)
      return cmp.Compare(ra, rb)
   })
   return medium[0], nil
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

type visitorData string

func (v *visitorData) UnmarshalText(data []byte) error {
   visitor, err := url.PathUnescape(string(data))
   if err != nil {
      return err
   }
   *v = visitorData(visitor)
   return nil
}

type ytCfg struct {
   InnertubeContext struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
}

// youtube.go marker preserve
