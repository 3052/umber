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

type soundcloud_set struct {
   address string
   f       *flag.FlagSet
}

func new_soundcloud() *soundcloud_set {
   var set soundcloud_set
   set.f = flag.NewFlagSet("soundcloud", flag.ExitOnError)
   set.f.StringVar(&set.address, "a", "", "address")
   return &set
}

func (s *soundcloud_set) parse(args []string) (*song, error) {
   s.f.Parse(args)
   var resolve soundcloud.Resolve
   err := resolve.New(s.address)
   if err != nil {
      return nil, err
   }
   var song0 song
   song0.S = resolve.Title
   song0.Q = url.Values{
      "a": {strconv.FormatInt(time.Now().Unix(), 36)},
      "b": {strconv.FormatInt(resolve.Id, 10)},
      "c": {
         path.Base(strings.Replace(resolve.Artwork(), "large", "t500x500", 1)),
      },
      "p": {"s"},
      "y": {
         strconv.Itoa(resolve.DisplayDate.Year()),
      },
   }.Encode()
   return &song0, nil
}
