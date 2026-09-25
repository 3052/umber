// fix.go marker preserve
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
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
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

// visitorExpiredReason is what YouTube answers when the visitor ID has
// gone stale but the request itself is fine.
const visitorExpiredReason = "This content isn't available, try again later."

// errRateLimited is returned when YouTube answers HTTP 429.
// errVisitorExpired is returned when the visitor ID has expired.
// The run aborts; the next run fetches a fresh visitor ID.
var (
   errRateLimited    = fmt.Errorf("rate limited")
   errVisitorExpired = fmt.Errorf("visitor ID expired")
)

// stsCache holds the signature timestamp once extracted; it only changes
// when the player build changes. 0 means not fetched yet.
var stsCache int

var stsRe = regexp.MustCompile(`["']?signatureTimestamp["']?\s*[:=]\s*(\d+)`)

// do_fix sets the R (artist) and T (title) fields of every youtube.com song
// to the exact values reported by the innertube player API, and writes the
// result back to the same file. Because processed items cannot be told apart
// from unprocessed ones in the songs file itself (R may be missing or just
// truncated), progress is tracked in a separate resume file. If the run is
// interrupted (for example by rate limiting), running the same command again
// picks up where it left off.
func do_fix(name string) error {
   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   resume := resume_path(name)
   done, err := read_resume(resume)
   if err != nil {
      return err
   }

   type job struct {
      song *song
      id   string
   }
   var jobs []job
   for _, s := range songs {
      if done[s.I] {
         continue
      }
      id, err := youtube_id(s.I)
      if err != nil {
         return err
      }
      if id == "" {
         continue // not on youtube.com
      }
      jobs = append(jobs, job{s, id})
   }
   if len(jobs) == 0 {
      log.Println("nothing to fix:", name)
      return nil
   }
   log.Println(len(jobs), "songs to fix")

   // A fresh visitor ID is fetched each run, since a cached one may have
   // expired.
   visitorID, err := fetchVisitorID()
   if err != nil {
      return err
   }

   for i, j := range jobs {
      fmt.Println(j.song.I)
      play, err := fetch_player(j.id, visitorID)
      if err != nil {
         if errors.Is(err, errRateLimited) || errors.Is(err, errVisitorExpired) {
            return fmt.Errorf(
               "%w (%v of %v processed, run the same command again to resume)",
               err, i, len(jobs),
            )
         }
         return err
      }
      if play.PlayabilityStatus.Status != "OK" {
         return fmt.Errorf(
            "%s: %s — %s",
            j.song.I, play.PlayabilityStatus.Status, play.PlayabilityStatus.Reason,
         )
      }
      if play.VideoDetails.Title == "" {
         return fmt.Errorf("%s: player response has empty title", j.song.I)
      }
      if play.VideoDetails.Author == "" {
         return fmt.Errorf("%s: player response has empty author", j.song.I)
      }
      j.song.R = play.VideoDetails.Author
      j.song.T = play.VideoDetails.Title
      if err := write_songs(name, songs); err != nil {
         return err
      }
      done[j.song.I] = true
      if err := write_resume(resume, done); err != nil {
         return err
      }
   }
   log.Println("processed", len(jobs), "YouTube songs")
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

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "", "input JSON file path (required)")
   flag.Parse()
   if *name == "" {
      flag.Usage()
      os.Exit(1)
   }
   if err := do_fix(*name); err != nil {
      log.Fatal(err)
   }
}

func read_resume(name string) (map[string]bool, error) {
   data, err := os.ReadFile(name)
   if errors.Is(err, os.ErrNotExist) {
      return map[string]bool{}, nil
   }
   if err != nil {
      return nil, err
   }
   var done map[string]bool
   if err := json.Unmarshal(data, &done); err != nil {
      return nil, fmt.Errorf("cannot parse %v (delete it to start over): %w", name, err)
   }
   if done == nil {
      done = map[string]bool{}
   }
   return done, nil
}

// resume_path returns the path of the file that tracks which songs have
// already been processed. It sits next to the songs file.
func resume_path(name string) string {
   base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
   return filepath.Join(filepath.Dir(name), base+".resume.json")
}

// signatureTimestamp extracts the signature timestamp (sts) from the player
// base.js. Current YouTube clients must send it in
// playbackContext.contentPlaybackContext, or /player returns UNPLAYABLE
// ("Video unavailable" / "The page needs to be reloaded."). The watch page
// of the first video supplies the current base.js path; the result is
// cached for the rest of the run.
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

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   // Write to a temp file first, then rename, so an interrupted run cannot
   // leave a half-written songs file behind.
   temp := name + ".tmp"
   if err := os.WriteFile(temp, data, os.ModePerm); err != nil {
      return err
   }
   return os.Rename(temp, name)
}

func write_resume(name string, done map[string]bool) error {
   // json.Marshal sorts map keys, so the output is stable across runs.
   data, err := json.MarshalIndent(done, "", " ")
   if err != nil {
      return err
   }
   data = append(data, '\n')
   return write_file(name, data)
}

// write_songs writes the songs in the same format as the main script.
func write_songs(name string, songs []*song) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err := enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
}

// youtube_id extracts the video ID from a watch URL,
// e.g. https://youtube.com/watch?v=Q0ifFtMCFv8 -> Q0ifFtMCFv8.
// It returns an empty string if the link is on any other host.
func youtube_id(link string) (string, error) {
   u, err := url.Parse(link)
   if err != nil {
      return "", fmt.Errorf("parse %q: %w", link, err)
   }
   if u.Hostname() != "youtube.com" {
      return "", nil
   }
   id := u.Query().Get("v")
   if id == "" {
      return "", fmt.Errorf("no video ID in %q", link)
   }
   return id, nil
}

type player struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author string
      Title  string
   }
}

func fetch_player(video_id, visitorID string) (*player, error) {
   sts, err := signatureTimestamp(video_id)
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
      "videoId":     video_id,
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
   result := &player{}
   if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
      return nil, err
   }
   if result.PlayabilityStatus.Status == "LOGIN_REQUIRED" &&
      strings.Contains(result.PlayabilityStatus.Reason, "not a bot") {
      return nil, fmt.Errorf("%w: %s — %s",
         errVisitorExpired, result.PlayabilityStatus.Status, result.PlayabilityStatus.Reason)
   }
   if result.PlayabilityStatus.Status == "UNPLAYABLE" &&
      result.PlayabilityStatus.Reason == visitorExpiredReason {
      return nil, fmt.Errorf("%w: %s — %s",
         errVisitorExpired, result.PlayabilityStatus.Status, result.PlayabilityStatus.Reason)
   }
   return result, nil
}

type song struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R,omitempty"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

func read_songs(name string) ([]*song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []*song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
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

// fix.go marker preserve
