package youtube

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "net/http"
   "net/url"
   "os"
   "path"
   "sort"
   "strconv"
   "strings"
   "time"
   "umber/youtube"
)

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
   flag.Parse()
   songs, err := read_songs("umber.json")
   if err != nil {
      panic(err)
   }
   if len(os.Args) >= 3 {
      args := os.Args[2:]
      var song1 *song
      song1, err = new_youtube().parse(args)
      if err != nil {
         panic(err)
      }
      songs = append([]*song{song1}, songs...)
      var buf bytes.Buffer
      enc := json.NewEncoder(&buf)
      enc.SetEscapeHTML(false)
      enc.SetIndent("", " ")
      err := enc.Encode(songs)
      if err != nil {
         panic(err)
      }
      err = os.WriteFile("umber.json", buf.Bytes(), os.ModePerm)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

const web_version     = "2.20231219.04.00"

func (y *youtube_set) parse(args []string) (*song, error) {
   y.f.Parse(args)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("p", "y")
   value.Set("b", y.tube.VideoId)
   base, err := get_image(y.tube.VideoId)
   if err != nil {
      return nil, err
   }
   if base != "" {
      value.Set("c", base)
   }
   play, err := y.tube.Player()
   if err != nil {
      return nil, err
   }
   var song1 song
   song1.S = play.VideoDetails.Author + " - " + play.VideoDetails.Title
   fmt.Println(play.VideoDetails.ShortDescription)
   value.Set("y", strconv.Itoa(
      play.Microformat.PlayerMicroformatRenderer.PublishDate[0].Year(),
   ))
   song1.Q = value.Encode()
   return &song1, nil
}

func new_youtube() *youtube_set {
   set.f.StringVar(&set.tube.VideoId, "b", "", "video ID")
   set.tube.Context.Client.ClientName = "WEB"
}

func (i *InnerTube) Player() (*Player, error) {
   type InnerTube struct {
      ContentCheckOk bool `json:"contentCheckOk,omitempty"`
      Context        struct {
         Client struct {
            AndroidSdkVersion int    `json:"androidSdkVersion"`
            ClientName        string `json:"clientName"`
            ClientVersion     string `json:"clientVersion"`
            OsVersion         string `json:"osVersion"`
         } `json:"client"`
      } `json:"context"`
      RacyCheckOk bool   `json:"racyCheckOk,omitempty"`
      VideoId     string `json:"videoId"`
   }
   i.Context.Client.AndroidSdkVersion = 32
   i.Context.Client.OsVersion = "12"
   i.Context.Client.ClientVersion = web_version
   data, err := json.Marshal(i)
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
   req.Header.Set("user-agent", user_agent+i.Context.Client.ClientVersion)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   play := &Player{}
   err = json.NewDecoder(resp.Body).Decode(play)
   if err != nil {
      return nil, err
   }
   return play, nil
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
