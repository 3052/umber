package main

import (
   "154.pages.dev/platform/soundcloud"
   "flag"
   "net/url"
   "path"
   "strconv"
   "strings"
   "time"
)

func (s *soundcloud_set) parse(arg []string) (*record, error) {
   s.Parse(arg)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("p", "s")
   track, err := soundcloud.Resolve(s.address)
   if err != nil {
      return nil, err
   }
   var row record
   row.S = track.Title
   value.Set("b", strconv.FormatInt(track.ID, 10))
   value.Set("c", path.Base(track.Artwork()))
   year, _, ok := strings.Cut(track.DisplayDate, "-")
   if ok {
      value.Set("y", year)
   }
   row.Q = value.Encode()
   return &row, nil
}

type soundcloud_set struct {
   *flag.FlagSet
   address string
}

func new_soundcloud() *soundcloud_set {
   var set soundcloud_set
   set.FlagSet = flag.NewFlagSet("soundcloud", flag.ExitOnError)
   set.StringVar(&set.address, "a", "", "address")
   return &set
}
