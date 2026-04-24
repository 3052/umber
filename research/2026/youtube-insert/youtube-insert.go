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
   "slices"
   "strconv"
   "strings"
   "time"
)

// Explicitly indexed string array replicating the exact stable sort order
// of the original logic (Height < 720, Default, Webp).
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

func get_image(video_id string) (string, error) {
   for index, name := range yt_imgs {
      var address string
      if strings.HasSuffix(name, ".webp") {
         address = "http://i.ytimg.com/vi_webp/" + video_id + "/" + name
      } else {
         address = "http://i.ytimg.com/vi/" + video_id + "/" + name
      }
      status, err := head(address)
      if err != nil {
         return "", err
      }
      if status == http.StatusOK {
         if index == 0 {
            return "", nil
         }
         return name, nil
      }
   }
   return "", nil
}

func fetch_player(video_id string) (*player, error) {
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
   req.Header.Set("x-goog-visitor-id", "CgtJeU1qSXlNakl5TQ")
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

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

type song struct {
   Q string
   S string
}

func read_songs(name string) ([]song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "umber.json", "name")
   video_url := flag.String("u", "", "video URL")
   flag.Parse()
   if *video_url != "" {
      raw_url := *video_url

      // Just in case an encoded URL gets passed in occasionally
      if strings.Contains(raw_url, "%3A%2F%2F") {
         if unescaped, err := url.QueryUnescape(raw_url); err == nil {
            raw_url = unescaped
         }
      }

      u, err := url.Parse(raw_url)
      if err != nil {
         log.Fatal("Invalid URL:", err)
      }

      video_id := u.Query().Get("v")
      if video_id == "" {
         log.Fatal("Could not extract 'v' parameter from URL")
      }

      err = do_video_id(video_id, *name)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func do_video_id(video_id, name string) error {
   // 1 player
   play, err := fetch_player(video_id)
   if err != nil {
      return err
   }
   fmt.Println(play.VideoDetails.ShortDescription)
   // 2 image
   image, err := get_image(video_id)
   if err != nil {
      return err
   }
   // 3 values
   values := url.Values{}
   values.Set("a", strconv.FormatInt(time.Now().Unix(), 36))
   values.Set("b", video_id)
   if image != "" {
      values.Set("c", image)
   }
   values.Set("p", "y")
   values.Set("y", strconv.Itoa(
      play.Microformat.PlayerMicroformatRenderer.PublishDate.Year(),
   ))
   // 4 song
   var song_var song
   song_var.Q = values.Encode()
   song_var.S = play.VideoDetails.Author + " - " + play.VideoDetails.Title
   // 5 songs
   songs, err := read_songs(name)
   if err != nil {
      return err
   }
   songs = slices.Insert(songs, 0, song_var)
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err = enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
}
