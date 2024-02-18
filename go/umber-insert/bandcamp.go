package main

import (
   "154.pages.dev/platform/bandcamp"
   "flag"
   "net/url"
   "strconv"
   "time"
)

type bandcamp_set struct {
   *flag.FlagSet
   address string
}

func new_bandcamp() *bandcamp_set {
   var set bandcamp_set
   set.FlagSet = flag.NewFlagSet("bandcamp", flag.ExitOnError)
   set.StringVar(&set.address, "a", "", "address")
   return &set
}

func (b *bandcamp_set) parse(arg []string) (*record, error) {
   b.Parse(arg)
   val := make(url.Values)
   now := strconv.FormatInt(time.Now().Unix(), 36)
   val.Set("a", now)
   val.Set("p", "bandcamp")
   var rec record
   param, err := bandcamp.New_Params(b.address)
   if err != nil {
      return nil, err
   }
   val.Set("b", strconv.Itoa(param.I_ID))
   track, err := param.Tralbum()
   if err != nil {
      return nil, err
   }
   val.Set("c", strconv.FormatInt(track.Art_ID, 10))
   rec.S = track.Tralbum_Artist + " - " + track.Title
   val.Set("y", strconv.Itoa(track.Date().Year()))
   rec.Q = val.Encode()
   return &rec, nil
}
