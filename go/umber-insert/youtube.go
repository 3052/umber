package main

import (
   "154.pages.dev/platform/youtube"
   "flag"
   "fmt"
   "net/http"
   "net/url"
   "path"
   "sort"
   "strconv"
   "strings"
   "time"
)

func (y *youtube_set) parse(arg []string) (*record, error) {
   y.Parse(arg)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("p", "y")
   value.Set("b", y.r.VideoId)
   base, err := get_image(y.r.VideoId)
   if err != nil {
      return nil, err
   }
   if base != "" {
      value.Set("c", base)
   }
   var play youtube.Player
   if err := play.Post(y.r, nil); err != nil {
      return nil, err
   }
   var rec record
   rec.S = play.VideoDetails.Author + " - " + play.VideoDetails.Title
   fmt.Println(play.VideoDetails.ShortDescription)
   year, _, ok := strings.Cut(
      play.Microformat.PlayerMicroformatRenderer.PublishDate, "-",
   )
   if ok {
      value.Set("y", year)
   }
   rec.Q = value.Encode()
   return &rec, nil
}

type youtube_set struct {
   *flag.FlagSet
   r youtube.Request
}

func new_youtube() *youtube_set {
   var y youtube_set
   y.r.Web()
   y.FlagSet = flag.NewFlagSet("youtube", flag.ExitOnError)
   y.StringVar(&y.r.VideoId, "b", "", "video ID")
   y.Var(&y.r, "a", "address")
   return &y
}

func get_image(video_id string) (string, error) {
   var imgs []youtube.Image
   for _, img := range youtube.Images {
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
