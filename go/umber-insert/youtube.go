package main

import (
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
   y.f.Parse(arg)
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
   play, err := y.tube.Player(nil)
   if err != nil {
      return nil, err
   }
   var rec record
   rec.S = play.VideoDetails.Author + " - " + play.VideoDetails.Title
   fmt.Println(play.VideoDetails.ShortDescription)
   value.Set("y", strconv.Itoa(
      play.Microformat.PlayerMicroformatRenderer.PublishDate.Time.Year(),
   ))
   rec.Q = value.Encode()
   return &rec, nil
}

type youtube_set struct {
   f *flag.FlagSet
   tube youtube.InnerTube
}

func new_youtube() *youtube_set {
   var set youtube_set
   set.f = flag.NewFlagSet("youtube", flag.ExitOnError)
   set.f.StringVar(&set.tube.VideoId, "b", "", "video ID")
   set.tube.Context.Client.ClientName = "WEB"
   return &set
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
