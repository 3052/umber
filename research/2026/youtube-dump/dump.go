// ytdump.go marker preserve
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

// VISIONOS client constants, identical to youtube.go so this tool sends
// exactly the same /player request as the checker does.
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

// stsCache holds the signature timestamp once extracted. 0 = not fetched.
var stsCache int

var stsRe = regexp.MustCompile(`["']?signatureTimestamp["']?\s*[:=]\s*(\d+)`)

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

func main() {
   video := flag.String("v", "", "YouTube video ID or watch URL")
   flag.Parse()
   if *video == "" {
      flag.Usage()
      os.Exit(2)
   }

   videoID, err := normalizeVideoID(*video)
   if err != nil {
      log.Fatal(err)
   }

   visitorID, err := fetchVisitorID()
   if err != nil {
      log.Fatal(err)
   }

   body, err := playerResponse(videoID, visitorID)
   if err != nil {
      log.Fatal(err)
   }

   // Mirror the playability checks from youtube.go, but only as a warning
   // on stderr: the full JSON is still dumped on stdout either way.
   var probe struct {
      PlayabilityStatus struct {
         Status string
         Reason string
      }
   }
   if json.Unmarshal(body, &probe) == nil {
      switch {
      case probe.PlayabilityStatus.Status == "LOGIN_REQUIRED" &&
         strings.Contains(probe.PlayabilityStatus.Reason, "not a bot"):
         fmt.Fprintf(os.Stderr, "%v: %s — %s\n", errVisitorExpired,
            probe.PlayabilityStatus.Status, probe.PlayabilityStatus.Reason)
      case probe.PlayabilityStatus.Status == "UNPLAYABLE" &&
         probe.PlayabilityStatus.Reason == visitorExpiredReason:
         fmt.Fprintf(os.Stderr, "%v: %s — %s\n", errVisitorExpired,
            probe.PlayabilityStatus.Status, probe.PlayabilityStatus.Reason)
      case probe.PlayabilityStatus.Status != "OK":
         fmt.Fprintf(os.Stderr, "playability %s — %s\n",
            probe.PlayabilityStatus.Status, probe.PlayabilityStatus.Reason)
      }
   }

   // json.Indent keeps the field order exactly as the server sent it,
   // unlike unmarshal + MarshalIndent (which sorts map keys).
   var out bytes.Buffer
   if err := json.Indent(&out, body, "", "  "); err != nil {
      // Not valid JSON (shouldn't happen); fall back to the raw body.
      os.Stdout.Write(body)
      fmt.Println()
      return
   }
   fmt.Println(out.String())
}

// normalizeVideoID accepts a bare video ID or a youtube.com / youtu.be
// watch URL and returns the bare ID.
func normalizeVideoID(arg string) (string, error) {
   arg = strings.TrimSpace(arg)
   if arg == "" {
      return "", errors.New("empty video ID")
   }
   if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
      u, err := url.Parse(arg)
      if err != nil {
         return "", fmt.Errorf("parse %q: %w", arg, err)
      }
      switch u.Hostname() {
      case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com":
         if id := u.Query().Get("v"); id != "" {
            return id, nil
         }
         for _, p := range []string{"/shorts/", "/live/", "/embed/"} {
            if strings.HasPrefix(u.Path, p) {
               if id := strings.Trim(strings.TrimPrefix(u.Path, p), "/"); id != "" {
                  return id, nil
               }
            }
         }
      case "youtu.be":
         if id := strings.Trim(u.Path, "/"); id != "" {
            return id, nil
         }
      }
      return "", fmt.Errorf("no video ID in %q", arg)
   }
   if strings.ContainsAny(arg, "/?&# ") {
      return "", fmt.Errorf("%q does not look like a video ID or watch URL", arg)
   }
   return arg, nil
}

// playerResponse performs the same innertube /player request as
// fetch_player in youtube.go, but returns the raw response body.
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
   body, err := io.ReadAll(resp.Body)
   if err != nil {
      return nil, err
   }
   if len(body) == 0 {
      return nil, fmt.Errorf("empty response body from /player")
   }
   return body, nil
}

// signatureTimestamp extracts the signature timestamp (sts) from the
// player base.js; the result is cached for the rest of the run.
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

type ytCfg struct {
   InnertubeContext struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
}

// ytdump.go marker preserve
