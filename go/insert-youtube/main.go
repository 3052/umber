package youtube

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "path"
   "sort"
   "strconv"
   "strings"
   "time"
)

func (p *Player) New(video_id string) error {
   value := map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "WEB",
            "clientVersion": "2.20231219.04.00",
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
   }
   data, err := json.MarshalIndent(value, "", " ")
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
   var imgs []youtube.YtImg
   for _, img := range youtube.YtImgs {
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
      fmt.Println(address)
      resp, err := http.Head(address)
      if err != nil {
         return "", err
      }
      if resp.StatusCode == http.StatusOK {
         if index == 0 {
            return "", nil
         }
         return path.Base(address), nil
      }
   }
   return "", nil
}

func (y *YtImg) String() string {
   var b strings.Builder
   b.WriteString("http://i.ytimg.com/vi")
   if strings.HasSuffix(y.Name, ".webp") {
      b.WriteString("_webp")
   }
   b.WriteByte('/')
   b.WriteString(y.VideoId)
   b.WriteByte('/')
   b.WriteString(y.Name)
   return b.String()
}

type YtImg struct {
   Height  int
   Name    string
   VideoId string
   Width   int
}

var YtImgs = []YtImg{
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

type Player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate Date
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

func (d *Date) UnmarshalText(data []byte) error {
   var err error
   d[0], err = time.Parse(time.RFC3339, string(data))
   if err != nil {
      return err
   }
   return nil
}

type Date [1]time.Time

type song struct {
   Q string
   S string
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

func main() {
   var f flags
   flag.StringVar(&f.name, "n", "umber.json", "name")
   flag.StringVar(&f.video_id, "v", "", "video ID")
   flag.Parse()
   if f.video_id != "" {
      err := f.do_youtube()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

type flags struct {
   name     string
   video_id string
}
