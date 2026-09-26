// youtube.go marker preserve
package main

import (
   "bytes"
   "cmp"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path/filepath"
   "regexp"
   "slices"
   "strconv"
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

const sep = "\nytcfg.set("

const visitorExpiredReason = "This content isn't available, try again later."

var (
   stsCache = map[string]int{}
   stsMu    sync.Mutex
   stsRe    = regexp.MustCompile(`["']?signatureTimestamp["']?\s*[:=]\s*(\d+)`)
)

var errVisitorExpired = fmt.Errorf("visitor ID expired")

// downloadYouTube downloads the AUDIO_QUALITY_MEDIUM audio stream for a
// YouTube watch URL (youtube.com host) and remuxes it with ffmpeg. title is
// the record's filename stem, built by baseName: "author - title", except a
// YouTube " - Topic" author is stripped, and the title alone is used when
// it already contains the author.
func downloadYouTube(pageURL, title, visitorID, outputDir string, threads int, maxETA time.Duration) error {
   videoID, err := videoIDFromURL(pageURL)
   if err != nil {
      return err
   }

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

   pc := PlaybackContext{}
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
   name := sanitizeFilename(title, outExt, outputDir)
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

   cmd := exec.Command("ffmpeg", "-i", dlPath, "-c", "copy", ffTmp)
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

// signatureTimestamp extracts the signature timestamp (sts) from the player
// base.js. Current YouTube clients must send it in
// playbackContext.contentPlaybackContext, or /player returns UNPLAYABLE.
// Results are cached: sts only changes when the player build changes.
func signatureTimestamp(playerJSPath string) (int, error) {
   stsMu.Lock()
   defer stsMu.Unlock()

   if sts, ok := stsCache[playerJSPath]; ok {
      return sts, nil
   }

   jsURL := playerJSPath
   if jsURL[0] == '/' {
      jsURL = "https://www.youtube.com" + jsURL
   }
   resp, err := http.Get(jsURL)
   if err != nil {
      return 0, err
   }
   defer resp.Body.Close()

   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return 0, err
   }

   m := stsRe.FindSubmatch(data)
   if m == nil {
      return 0, fmt.Errorf("signatureTimestamp not found in %s", playerJSPath)
   }
   sts, err := strconv.Atoi(string(m[1]))
   if err != nil {
      return 0, fmt.Errorf("parse signatureTimestamp: %w", err)
   }
   stsCache[playerJSPath] = sts
   return sts, nil
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

type visitorData string

func (v *visitorData) UnmarshalText(data []byte) error {
   visitor, err := url.PathUnescape(string(data))
   if err != nil {
      return err
   }
   *v = visitorData(visitor)
   return nil
}

type watchConfig struct {
   PlayerJSURL string      `json:"PLAYER_JS_URL"`
   VisitorData visitorData `json:"VISITOR_DATA"`
}

// fetchWatchConfig fetches the regular watch page. Its ytcfg provides a fresh
// visitor ID and PLAYER_JS_URL, which points at the player base.js build used
// to extract the signature timestamp.
func fetchWatchConfig(videoID string) (*watchConfig, error) {
   resp, err := http.Get("https://www.youtube.com/watch?v=" + videoID)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()

   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return nil, err
   }

   data, err = extractJSON(data, []byte(sep))
   if err != nil {
      return nil, err
   }

   var cfg watchConfig
   if err := json.Unmarshal(data, &cfg); err != nil {
      return nil, err
   }
   if cfg.PlayerJSURL == "" {
      return nil, fmt.Errorf("no PLAYER_JS_URL in watch config")
   }
   return &cfg, nil
}

type ytCfg struct {
   InnertubeClientName    string `json:"INNERTUBE_CLIENT_NAME"`
   InnertubeClientVersion string `json:"INNERTUBE_CLIENT_VERSION"`
   InnertubeContext       struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
   InnertubeContextClientName    int    `json:"INNERTUBE_CONTEXT_CLIENT_NAME"`
   InnertubeContextClientVersion string `json:"INNERTUBE_CONTEXT_CLIENT_VERSION"`
}

// youtube.go marker preserve
