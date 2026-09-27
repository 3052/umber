// youtube.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

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
   PlayerJSURL string `json:"PLAYER_JS_URL"`
}

type ytCfg struct {
   InnertubeContext struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
}

// playerData pulls just what we need from the /player response.
// AdaptiveFormats stays a RawMessage so the server's key order and
// formatting survive untouched.
type playerData struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   StreamingData struct {
      AdaptiveFormats json.RawMessage `json:"adaptiveFormats"`
   } `json:"streamingData"`
}

// main.go marker preserve
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

// visitorExpiredReason is what YouTube answers when the visitor ID has
// gone stale but the request itself is fine.
const visitorExpiredReason = "This content isn't available, try again later."

// errRateLimited is returned when YouTube answers HTTP 429.
// errVisitorExpired is reported when the visitor ID has expired.
var (
   errRateLimited    = fmt.Errorf("rate limited")
   errVisitorExpired = fmt.Errorf("visitor ID expired")
)

// stsCache holds the signature timestamp once extracted; it only changes
// when the player build changes. 0 means not fetched yet.
var stsCache int

var stsRe = regexp.MustCompile(`["']?signatureTimestamp["']?\s*[:=]\s*(\d+)`)

func main() {
   video := flag.String("v", "", "YouTube video ID")
   flag.Parse()
   if *video == "" {
      flag.Usage()
      os.Exit(2)
   }

   if err := doDump(*video); err != nil {
      log.Fatal(err)
   }
}

///

// doDump fetches the /player response for videoID and prints its
// adaptiveFormats array as indented JSON on stdout.
func doDump(videoID string) error {
   visitorID, err := fetchVisitorID()
   if err != nil {
      return err
   }

   body, err := playerResponse(videoID, visitorID)
   if err != nil {
      return err
   }

   var pd playerData
   if err := json.Unmarshal(body, &pd); err != nil {
      return err
   }

   switch {
   case pd.PlayabilityStatus.Status == "LOGIN_REQUIRED" &&
      strings.Contains(pd.PlayabilityStatus.Reason, "not a bot"):
      return fmt.Errorf("%v: %s — %s", errVisitorExpired,
         pd.PlayabilityStatus.Status, pd.PlayabilityStatus.Reason)
   case pd.PlayabilityStatus.Status == "UNPLAYABLE" &&
      pd.PlayabilityStatus.Reason == visitorExpiredReason:
      return fmt.Errorf("%v: %s — %s", errVisitorExpired,
         pd.PlayabilityStatus.Status, pd.PlayabilityStatus.Reason)
   }

   af := pd.StreamingData.AdaptiveFormats
   if len(af) == 0 || string(af) == "null" {
      return fmt.Errorf("no adaptiveFormats in response (playability %s — %s)",
         pd.PlayabilityStatus.Status, pd.PlayabilityStatus.Reason)
   }

   // json.Indent keeps the server's field order; MarshalIndent would
   // sort keys because the top level is a map.
   var out bytes.Buffer
   if err := json.Indent(&out, af, "", "  "); err != nil {
      return err
   }
   fmt.Println(out.String())
   return nil
}

// extractJSON isolates the JSON payload by balancing curly braces
// directly on a byte slice to avoid memory allocations.
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

// fetchVisitorID retrieves the X-Goog-Visitor-Id from YouTube's
// homepage by parsing the ytcfg JSON embedded in the HTML.
func fetchVisitorID() (string, error) {
   resp, err := http.Get("https://www.youtube.com")
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

// playerResponse performs the same innertube /player request the checker
// sends, but returns the raw response body instead of decoding it into
// a fixed struct.
func playerResponse(videoID, visitorID string) ([]byte, error) {
   sts, err := signatureTimestamp(videoID)
   if err != nil {
      return nil, fmt.Errorf("signature timestamp: %w", err)
   }
   data, err := json.Marshal(map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]any{
            "clientName":    visionOSClientName,
            "clientVersion": visionOSClientVersion,
            "deviceMake":    "Apple",
            "deviceModel":   "RealityDevice17,1",
            "userAgent":     visionOSUserAgent,
            "osName":        "visionOS",
            "osVersion":     "26.5.23O471",
            "hl":            "en",
            "timeZone":      "UTC",
         },
      },
      "playbackContext": map[string]any{
         "contentPlaybackContext": map[string]any{
            "html5Preference":    "HTML5_PREF_WANTS",
            "signatureTimestamp": sts,
         },
      },
      "racyCheckOk": true,
      "videoId":     videoID,
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
   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("X-Goog-Visitor-Id", visitorID)
   req.Header.Set("X-Youtube-Client-Name", visionOSClientID)
   req.Header.Set("X-Youtube-Client-Version", visionOSClientVersion)
   req.Header.Set("User-Agent", visionOSUserAgent)
   req.Header.Set("Origin", "https://www.youtube.com")
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

// signatureTimestamp extracts the signature timestamp (sts) from the player
// base.js. Current YouTube clients must send it in
// playbackContext.contentPlaybackContext, or /player returns UNPLAYABLE
// ("Video unavailable" / "The page needs to be reloaded."). The watch page
// of the video supplies the current base.js path; the result is cached for
// the rest of the run.
func signatureTimestamp(videoID string) (int, error) {
   if stsCache != 0 {
      return stsCache, nil
   }
   wc, err := fetchWatchConfig(videoID)
   if err != nil {
      return 0, err
   }
   jsURL := wc.PlayerJSURL
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
      return 0, fmt.Errorf("signatureTimestamp not found in %s", wc.PlayerJSURL)
   }
   sts, err := strconv.Atoi(string(m[1]))
   if err != nil {
      return 0, fmt.Errorf("parse signatureTimestamp: %w", err)
   }
   stsCache = sts
   return sts, nil
}

// fetchWatchConfig fetches the regular watch page. Its ytcfg provides
// PLAYER_JS_URL, which points at the player base.js build used to
// extract the signature timestamp.
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

// youtube.go marker preserve
