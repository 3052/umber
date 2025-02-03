package main

import (
   "41.neocities.org/platform/soundcloud"
   "flag"
   "net/url"
   "path"
   "strconv"
   "strings"
   "time"
)

func (s *soundcloud_set) parse(arg []string) (*record, error) {
   s.f.Parse(arg)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("p", "s")
   var row record
   //////////////////////////////////////////////////////////////////////////////
   var track soundcloud.ClientTrack
   err := track.Resolve(s.address)
   if err != nil {
      return nil, err
   }
   value.Set("c", path.Base(
      strings.Replace(track.Artwork(), "large", "t500x500", 1),
   ))
   //////////////////////////////////////////////////////////////////////////////
   value.Set("y", strconv.Itoa(
      track.DisplayDate.Year(),
   ))
   value.Set("b", strconv.FormatInt(track.Id, 10))
   row.S = track.Title
   row.Q = value.Encode()
   return &row, nil
}

type soundcloud_set struct {
   f *flag.FlagSet
   address string
}

func new_soundcloud() *soundcloud_set {
   var set soundcloud_set
   set.f = flag.NewFlagSet("soundcloud", flag.ExitOnError)
   set.f.StringVar(&set.address, "a", "", "address")
   return &set
}
