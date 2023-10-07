package main

import (
   "154.pages.dev/media/youtube"
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
   val := make(url.Values)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   val.Set("a", now)
   val.Set("p", "y")
   val.Set("b", y.r.Video_ID)
   base, err := get_image(y.r.Video_ID)
   if err != nil {
      return nil, err
   }
   if base != "" {
      val.Set("c", base)
   }
   play, err := y.r.Player(nil)
   if err != nil {
      return nil, err
   }
   var rec record
   rec.S = play.Video_Details.Author + " - " + play.Video_Details.Title
   fmt.Println(play.Video_Details.Short_Description)
   year, _, ok := strings.Cut(play.Publish_Date(), "-")
   if ok {
      val.Set("y", year)
   }
   rec.Q = val.Encode()
   return &rec, nil
}

type youtube_set struct {
   *flag.FlagSet
   r youtube.Request
}

func new_youtube() *youtube_set {
   var y youtube_set
   y.r.Mobile_Web()
   y.FlagSet = flag.NewFlagSet("youtube", flag.ExitOnError)
   y.StringVar(&y.r.Video_ID, "b", "", "video ID")
   y.Var(&y.r, "a", "address")
   return &y
}

func get_image(video_ID string) (string, error) {
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
   req, err := http.NewRequest("HEAD", "", nil)
   if err != nil {
      return "", err
   }
   for index, img := range imgs {
      req.URL = img.URL(video_ID)
      fmt.Println("HEAD", req.URL)
      res, err := new(http.Transport).RoundTrip(req)
      if err == nil && res.StatusCode == http.StatusOK {
         if index == 0 {
            return "", nil
         }
         return path.Base(req.URL.Path), nil
      }
   }
   return "", nil
}
