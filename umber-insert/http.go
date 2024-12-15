package main

import (
   "flag"
   "net/url"
   "path"
   "strconv"
   "time"
)

type http_set struct {
   f *flag.FlagSet
   year string
   audio string
   image string
}

func new_http() *http_set {
   var set http_set
   set.f = flag.NewFlagSet("http", flag.ExitOnError)
   set.f.StringVar(&set.audio, "a", "", "audio")
   set.f.StringVar(&set.image, "i", "", "image")
   set.f.StringVar(&set.year, "y", "", "year")
   return &set
}

func (h *http_set) parse(arg []string) (*record, error) {
   h.f.Parse(arg)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("p", "h")
   value.Set("y", h.year)
   var rec record
   value.Set("b", h.audio)
   value.Set("c", h.image)
   rec.Q = value.Encode()
   rec.S = path.Base(h.audio)
   return &rec, nil
}
