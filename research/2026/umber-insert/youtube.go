// youtube.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "math"
   "net/http"
   "net/url"
   "slices"
   "strings"
   "time"
)

const sep = "\nytcfg.set("

// artist_title derives the artist and song title from a YouTube video's
// channel name and video title, the same way umber-update.go does. If the
// video title already contains the channel name it is used as-is, otherwise
// the channel is prepended, and the combined line is split at the space
// closest to its center. A line with no usable split point is an error,
// since that should never happen for a real video.
func artist_title(author, video_title string) (artist, title string, err error) {
   line := video_title
   if !containsAuthor(video_title, author) {
      line = author + ", " + video_title
   }
   left, right, ok := splitAtCenter(line)
   if !ok {
      return "", "", fmt.Errorf("no usable split point in %q", line)
   }
   return left, right, nil
}

// containsAuthor reports whether the title contains the author,
// ignoring case and spaces.
func containsAuthor(title, author string) bool {
   fold := func(s string) string {
      return strings.ToLower(strings.ReplaceAll(s, " ", ""))
   }
   return strings.Contains(fold(title), fold(author))
}

func do_video_id(video_id, name, visitorID string) error {
   watch := "https://youtube.com/watch?v=" + video_id

   raw_songs, err := read_songs(name)
   if err != nil {
      return err
   }
   seen := make(map[string]bool)
   var songs []Song
   input_exists := false

   // Iterate through ALL existing records to filter out duplicates
   for _, song := range raw_songs {
      if song.I == "" {
         // Safety fallback: if the record is missing the "I" string, keep it to prevent data loss
         songs = append(songs, song)
         continue
      }
      // Check if the input we are trying to add already exists
      if song.I == watch {
         input_exists = true
      }
      // If we haven't seen this ID yet in the loop, keep it and mark it as seen
      if !seen[song.I] {
         seen[song.I] = true
         songs = append(songs, song)
      }
   }

   if input_exists {
      // If pre-existing duplicates were found and cleaned from the file, save the clean file before exiting.
      if len(songs) < len(raw_songs) {
         log.Printf("Cleaned up %d pre-existing duplicate(s) in %s\n", len(raw_songs)-len(songs), name)
         _ = write_songs(name, songs)
      }
      return fmt.Errorf("duplicate found: '%s' already exists in %s", watch, name)
   }

   play, err := fetch_player(video_id, visitorID)
   if err != nil {
      return err
   }
   fmt.Println(play.VideoDetails.ShortDescription)

   author := play.VideoDetails.Author
   video_title := play.VideoDetails.Title
   if author == "" || video_title == "" {
      return fmt.Errorf("%s: player response has empty author or title", watch)
   }
   artist, title, err := artist_title(author, video_title)
   if err != nil {
      return fmt.Errorf("%s: %w", watch, err)
   }

   image, err := get_image(video_id)
   if err != nil {
      return err
   }

   // Insert native map data
   song_data := Song{
      D: time.Now().Unix(),
      I: watch,
      R: artist,
      T: title,
      Y: play.Microformat.PlayerMicroformatRenderer.PublishDate.Year(),
   }
   if image != "" {
      song_data.A = image
   }

   songs = slices.Insert(songs, 0, song_data)

   // Save the newly cleaned and updated list
   return write_songs(name, songs)
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

// splitAtCenter splits s on the space closest to the middle of the string and
// returns the trimmed halves. It works on runes, not bytes, so titles with
// non-ASCII characters split in the right place. Ties go to the leftmost
// space. ok is false when there is no usable split point.
func splitAtCenter(s string) (left, right string, ok bool) {
   runes := []rune(s)
   center := float64(len(runes)) / 2

   best, bestDist := -1, math.MaxFloat64
   for i, r := range runes {
      if r != ' ' {
         continue
      }
      if d := math.Abs(float64(i) - center); d < bestDist {
         best, bestDist = i, d
      }
   }
   if best < 0 {
      return "", "", false
   }

   left = strings.TrimSpace(string(runes[:best]))
   right = strings.TrimSpace(string(runes[best+1:]))
   if left == "" || right == "" {
      return "", "", false
   }
   return left, right, true
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
