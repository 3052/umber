package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "strings"
   "time"
)

func do_check(name string, start int) error {
   file, err := os.Open(name)
   if err != nil {
      return err
   }
   defer file.Close()

   var songs []struct {
      A string `json:"A,omitempty"` // image URL
      D int64  `json:"D"`
      I string `json:"I"` // source URL
      R string `json:"R"` // artist
      T string `json:"T"` // title
      Y int    `json:"Y"`
   }

   err = json.NewDecoder(file).Decode(&songs)
   if err != nil {
      return err
   }

   // Count the YouTube entries in range, so the remaining count
   // stays accurate since non-YouTube rows get skipped.
   remaining := 0
   for i, song := range songs {
      if i >= start && is_youtube(song.I) {
         remaining++
      }
   }

   for i, song := range songs {
      if i >= start && is_youtube(song.I) {
         video_id, err := video_id_from_url(song.I)
         if err != nil {
            return err
         }

         play, err := fetch_player(video_id)
         if err != nil {
            return err
         }

         fmt.Println(i, remaining, video_id, song.R, "-", song.T)
         remaining--

         if play.PlayabilityStatus.Status != "OK" {
            fmt.Printf("%+v\n", play.PlayabilityStatus)
            break
         }
         time.Sleep(99 * time.Millisecond)
      }
   }
   return nil
}

// is_youtube reports whether the source URL points at YouTube.
func is_youtube(link string) bool {
   u, err := url.Parse(link)
   if err != nil {
      return false
   }
   host := strings.ToLower(u.Hostname())
   return host == "youtube.com" || host == "youtu.be" ||
      strings.HasSuffix(host, ".youtube.com") // www., m., music., ...
}

func main() {
   name := flag.String("n", "umber.json", "name")
   start := flag.Int("s", -1, "start")
   flag.Parse()
   if *start >= 0 {
      err := do_check(*name, *start)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

// video_id_from_url extracts the video ID from a watch URL,
// e.g. https://youtube.com/watch?v=Q0ifFtMCFv8 -> Q0ifFtMCFv8
func video_id_from_url(link string) (string, error) {
   u, err := url.Parse(link)
   if err != nil {
      return "", fmt.Errorf("parse %q: %w", link, err)
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
      Author           string
      LengthSeconds    int64 `json:",string"`
      ShortDescription string
      Title            string
      VideoId          string
      ViewCount        int64 `json:",string"`
   }
}

func fetch_player(video_id string) (*player, error) {
   data, err := json.Marshal(map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "IOS",
            "clientVersion": "20.03.02",
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
   return result, nil
}
