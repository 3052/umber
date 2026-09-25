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
   "strings"
)

const sep = "\nytcfg.set("

// errRateLimited is returned when YouTube throttles us. Progress is
// already saved, so running the same command again resumes.
var errRateLimited = errors.New("rate limited")

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
      id, ok := youtube_video_id(s.I)
      if !ok {
         continue
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
         if errors.Is(err, errRateLimited) {
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

// fetchVisitorID retrieves the X-Goog-Visitor-Id from YouTube's homepage
// by parsing the ytcfg JSON embedded in the HTML.
func fetchVisitorID() (string, error) {
   targetUrl := &url.URL{Scheme: "https", Host: "www.youtube.com"}
   req := &http.Request{
      Method: http.MethodGet,
      URL:    targetUrl,
   }
   log.Println("fetching visitor ID from", req.URL)
   resp, err := http.DefaultClient.Do(req)
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

// youtube_video_id reports whether address is a youtube.com watch URL, and
// returns its video ID. The host must be exactly youtube.com, so
// www.youtube.com and music.youtube.com are ignored.
func youtube_video_id(address string) (string, bool) {
   u, err := url.Parse(address)
   if err != nil {
      return "", false
   }
   if u.Host != "youtube.com" {
      return "", false
   }
   id := u.Query().Get("v")
   if id == "" {
      return "", false
   }
   return id, true
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

// fetch_player requests video details from the innertube player API.
func fetch_player(video_id, visitorID string) (*player, error) {
   data, err := json.Marshal(map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "WEB",
            "clientVersion": "2.20231219.04.00",
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
   })
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
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
      return nil, fmt.Errorf("%w: %s", errRateLimited, resp.Status)
   }
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   result := &player{}
   err = json.NewDecoder(resp.Body).Decode(result)
   if err != nil {
      return nil, err
   }
   reason := strings.ToLower(result.PlayabilityStatus.Reason)
   if strings.Contains(reason, "not a bot") ||
      strings.Contains(reason, "too many requests") ||
      strings.Contains(reason, "throttl") {
      return nil, fmt.Errorf(
         "%w: %s — %s", errRateLimited,
         result.PlayabilityStatus.Status, result.PlayabilityStatus.Reason,
      )
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

type ytCfg struct {
   InnertubeContext struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
}

// fix.go marker preserve
