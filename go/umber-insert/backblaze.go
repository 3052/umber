package main

import (
   "flag"
   "net/url"
   "path/filepath"
   "strconv"
   "time"
)

type backblaze_set struct {
   *flag.FlagSet
   year string
   audio string
   image string
}

func new_backblaze() *backblaze_set {
   var set backblaze_set
   set.FlagSet = flag.NewFlagSet("backblaze", flag.ExitOnError)
   set.StringVar(&set.audio, "a", "", "audio")
   set.StringVar(&set.image, "i", "", "image")
   set.StringVar(&set.year, "y", "", "year")
   return &set
}

func (b *backblaze_set) parse(arg []string) (*record, error) {
   b.Parse(arg)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("b", filepath.Ext(b.audio))
   value.Set("c", filepath.Base(b.image))
   value.Set("p", "b2")
   value.Set("y", b.year)
   var rec record
   rec.Q = value.Encode()
   base := filepath.Base(b.audio)
   ext := filepath.Ext(base)
   rec.S = base[:len(base)-len(ext)]
   return &rec, nil
}
