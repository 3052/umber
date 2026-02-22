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
   "path"
   "slices"
   "sort"
   "strconv"
   "strings"
   "time"
)

func (p *player) New(video_id string) error {
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
      return err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return err
   }
   // data := base64.RawStdEncoding.EncodeToString([]byte("########"))
   // var message protobuf.Message
   // message.AddBytes(1, []byte(data))
   // return base64.RawStdEncoding.EncodeToString(message.Marshal())
   req.Header.Set("x-goog-visitor-id", "CgtJeU1qSXlNakl5TQ")
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return errors.New(resp.Status)
   }
   return json.NewDecoder(resp.Body).Decode(p)
}

func get_image(video_id string) (string, error) {
   var imgs []yt_img
   for _, img := range yt_imgs {
      if img.Height < 720 {
         imgs = append(imgs, img)
      }
   }
   sort.SliceStable(imgs, func(a, b int) bool {
      com := imgs[a].Height - imgs[b].Height
      if com != 0 {
         return com >= 1
      }
      def := func(i int) int {
         return strings.Index(imgs[i].Name, "default")
      }
      com = def(a) - def(b)
      if com != 0 {
         return com >= 1
      }
      def = func(i int) int {
         return strings.Index(imgs[i].Name, "webp")
      }
      return def(b) < def(a)
   })
   for index, img := range imgs {
      img.VideoId = video_id
      address := img.String()
      status, err := head(address)
      if err != nil {
         return "", err
      }
      if status == http.StatusOK {
         if index == 0 {
            return "", nil
         }
         return path.Base(address), nil
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

type yt_img struct {
   Height  int
   Name    string
   VideoId string
   Width   int
}

var yt_imgs = []yt_img{
   {Width: 120, Height: 90, Name: "default.jpg"},
   {Width: 120, Height: 90, Name: "1.jpg"},
   {Width: 120, Height: 90, Name: "2.jpg"},
   {Width: 120, Height: 90, Name: "3.jpg"},
   {Width: 120, Height: 90, Name: "default.webp"},
   {Width: 120, Height: 90, Name: "1.webp"},
   {Width: 120, Height: 90, Name: "2.webp"},
   {Width: 120, Height: 90, Name: "3.webp"},
   {Width: 320, Height: 180, Name: "mq1.jpg"},
   {Width: 320, Height: 180, Name: "mq2.jpg"},
   {Width: 320, Height: 180, Name: "mq3.jpg"},
   {Width: 320, Height: 180, Name: "mqdefault.jpg"},
   {Width: 320, Height: 180, Name: "mq1.webp"},
   {Width: 320, Height: 180, Name: "mq2.webp"},
   {Width: 320, Height: 180, Name: "mq3.webp"},
   {Width: 320, Height: 180, Name: "mqdefault.webp"},
   {Width: 480, Height: 360, Name: "0.jpg"},
   {Width: 480, Height: 360, Name: "hqdefault.jpg"},
   {Width: 480, Height: 360, Name: "hq1.jpg"},
   {Width: 480, Height: 360, Name: "hq2.jpg"},
   {Width: 480, Height: 360, Name: "hq3.jpg"},
   {Width: 480, Height: 360, Name: "0.webp"},
   {Width: 480, Height: 360, Name: "hqdefault.webp"},
   {Width: 480, Height: 360, Name: "hq1.webp"},
   {Width: 480, Height: 360, Name: "hq2.webp"},
   {Width: 480, Height: 360, Name: "hq3.webp"},
   {Width: 640, Height: 480, Name: "sddefault.jpg"},
   {Width: 640, Height: 480, Name: "sd1.jpg"},
   {Width: 640, Height: 480, Name: "sd2.jpg"},
   {Width: 640, Height: 480, Name: "sd3.jpg"},
   {Width: 640, Height: 480, Name: "sddefault.webp"},
   {Width: 640, Height: 480, Name: "sd1.webp"},
   {Width: 640, Height: 480, Name: "sd2.webp"},
   {Width: 640, Height: 480, Name: "sd3.webp"},
   {Width: 1280, Height: 720, Name: "hq720.jpg"},
   {Width: 1280, Height: 720, Name: "maxresdefault.jpg"},
   {Width: 1280, Height: 720, Name: "maxres1.jpg"},
   {Width: 1280, Height: 720, Name: "maxres2.jpg"},
   {Width: 1280, Height: 720, Name: "maxres3.jpg"},
   {Width: 1280, Height: 720, Name: "hq720.webp"},
   {Width: 1280, Height: 720, Name: "maxresdefault.webp"},
   {Width: 1280, Height: 720, Name: "maxres1.webp"},
   {Width: 1280, Height: 720, Name: "maxres2.webp"},
   {Width: 1280, Height: 720, Name: "maxres3.webp"},
}

func (y *yt_img) String() string {
   var data strings.Builder
   data.WriteString("http://i.ytimg.com/vi")
   if strings.HasSuffix(y.Name, ".webp") {
      data.WriteString("_webp")
   }
   data.WriteByte('/')
   data.WriteString(y.VideoId)
   data.WriteByte('/')
   data.WriteString(y.Name)
   return data.String()
}

func (d *date) UnmarshalText(data []byte) error {
   var err error
   d[0], err = time.Parse(time.RFC3339, string(data))
   if err != nil {
      return err
   }
   return nil
}

type date [1]time.Time

type player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate date
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
   video_id := flag.String("v", "", "video ID")
   flag.Parse()
   if *video_id != "" {
      err := do_video_id(*video_id, *name)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func do_video_id(video_id, name string) error {
   // 1 player
   var play player
   err := play.New(video_id)
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
      play.Microformat.PlayerMicroformatRenderer.PublishDate[0].Year(),
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
