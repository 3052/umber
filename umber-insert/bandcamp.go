package main

import (
   "41.neocities.org/platform/bandcamp"
   "errors"
   "flag"
   "net/url"
   "strconv"
   "time"
)

type bandcamp_set struct {
   address string
   f       *flag.FlagSet
}

func new_bandcamp() *bandcamp_set {
   var set bandcamp_set
   set.f = flag.NewFlagSet("bandcamp", flag.ExitOnError)
   set.f.StringVar(&set.address, "a", "", "address")
   return &set
}

func (b *bandcamp_set) parse(args []string) (*song, error) {
   b.f.Parse(args)
   var params bandcamp.ReportParams
   err := params.New(b.address)
   if err != nil {
      return nil, err
   }
   tralbum, ok := params.Tralbum()
   if !ok {
      return nil, errors.New("Tralbum")
   }
   detail, err := tralbum.Tralbum()
   if err != nil {
      return nil, err
   }
   var song1 song
   song1.S = detail.TralbumArtist + " - " + detail.Title
   song1.Q = url.Values{
      "a": {strconv.FormatInt(time.Now().Unix(), 36)},
      "b": {strconv.Itoa(params.Iid)},
      "c": {strconv.FormatInt(detail.ArtId, 10)},
      "p": {"bandcamp"},
      "y": {
         strconv.Itoa(detail.Time().Year()),
      },
   }.Encode()
   return &song1, nil
}
