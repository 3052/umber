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
   "slices"
   "strings"
   "time"
)

const sep = "\nytcfg.set("

// errVisitorExpired is returned when the visitor ID has expired and needs refresh.
var errVisitorExpired = fmt.Errorf("visitor ID expired")

var yt_imgs = []string{
   0:  "sddefault.webp",
   1:  "sddefault.jpg",
   2:  "sd1.webp",
   3:  "sd2.webp",
   4:  "sd3.webp",
   5:  "sd1.jpg",
   6:  "sd2.jpg",
   7:  "sd3.jpg",
   8:  "hqdefault.webp",
   9:  "hqdefault.jpg",
   10: "hq1.webp",
   11: "hq2.webp",
   12: "hq3.webp",
   13: "0.webp",
   14: "0.jpg",
   15: "hq1.jpg",
   16: "hq2.jpg",
   17: "hq3.jpg",
   18: "mqdefault.webp",
   19: "mqdefault.jpg",
   20: "mq1.webp",
   21: "mq2.webp",
   22: "mq3.webp",
   23: "mq1.jpg",
   24: "mq2.jpg",
   25: "mq3.jpg",
   26: "default.webp",
   27: "default.jpg",
   28: "1.webp",
   29: "2.webp",
   30: "3.webp",
   31: "1.jpg",
   32: "2.jpg",
   33: "3.jpg",
}

func do_video_id(video_id, name, visitorID string) error {
   watch := "https://youtube.com/watch?v=" + video_id

   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   if contains_song(songs, watch) {
      return fmt.Errorf("duplicate found: '%s' already exists in %s", watch, name)
   }

   play, err := fetch_player(video_id, visitorID)
   if err != nil {
      return err
   }
   fmt.Println(play.VideoDetails.ShortDescription)

   video_title := play.VideoDetails.Title
   if video_title == "" {
      return fmt.Errorf("%s: player response has empty title", watch)
   }

   image, err := get_image(video_id)
   if err != nil {
      return err
   }

   song_data := song{
      D: time.Now().Unix(),
      I: watch,
      R: play.VideoDetails.Author,
      T: video_title,
      Y: play.Microformat.PlayerMicroformatRenderer.PublishDate.Year(),
   }
   if image != "" {
      song_data.A = image
   }

   songs = append(songs, &song_data)
   slices.SortFunc(songs, func(a, b *song) int {
      return cmp.Compare(b.D, a.D)
   })

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

func get_image(video_id string) (string, error) {
   for index, name := range yt_imgs {
      var address string
      if strings.HasSuffix(name, ".webp") {
         address = "https://i.ytimg.com/vi_webp/" + video_id + "/" + name
      } else {
         address = "https://i.ytimg.com/vi/" + video_id + "/" + name
      }
      status, err := head(address)
      if err != nil {
         return "", err
      }
      if status == http.StatusOK {
         if index == 0 {
            return "", nil
         }
         return address, nil
      }
   }
   return "", nil
}

func head(address string) (int, error) {
   fmt.Println(address)
   resp, err := http.Head(address)
   if err != nil {
      return 0, err
   }
   defer resp.Body.Close()
   return resp.StatusCode, nil
}

type player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate time.Time
      }
   }
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author           string
      LengthSeconds    int64 `json:",string"`
      ShortDescription string
      Title            string
      VideoId          string
      ViewCount        int64 `json:",string"`
   }
}

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
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   result := &player{}
   err = json.NewDecoder(resp.Body).Decode(result)
   if err != nil {
      return nil, err
   }
   if result.PlayabilityStatus.Status == "LOGIN_REQUIRED" && strings.Contains(result.PlayabilityStatus.Reason, "not a bot") {
      return nil, fmt.Errorf("%w: %s — %s", errVisitorExpired, result.PlayabilityStatus.Status, result.PlayabilityStatus.Reason)
   }
   return result, nil
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
