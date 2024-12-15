package main

import (
   "41.neocities.org/platform/bandcamp"
   "flag"
   "net/url"
   "strconv"
   "time"
)

func (b *bandcamp_set) parse(arg []string) (*record, error) {
   b.f.Parse(arg)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   value := url.Values{}
   value.Set("a", now)
   value.Set("p", "bandcamp")
   var params bandcamp.ReportParams
   err := params.New(b.address)
   if err != nil {
      return nil, err
   }
   value.Set("b", strconv.Itoa(params.Iid))
   track, err := params.Tralbum()
   if err != nil {
      return nil, err
   }
   value.Set("c", strconv.FormatInt(track.ArtId, 10))
   var rec record
   rec.S = track.TralbumArtist + " - " + track.Title
   value.Set("y", strconv.Itoa(track.Date().Year()))
   rec.Q = value.Encode()
   return &rec, nil
}

type bandcamp_set struct {
   f *flag.FlagSet
   address string
}

func new_bandcamp() *bandcamp_set {
   var set bandcamp_set
   set.f = flag.NewFlagSet("bandcamp", flag.ExitOnError)
   set.f.StringVar(&set.address, "a", "", "address")
   return &set
}
