package main

import (
   "41.neocities.org/platform/bandcamp"
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "net/url"
   "os"
   "strconv"
   "time"
)

type song struct {
   Q string
   S string
}

func read_songs(name string) ([]*song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []*song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

func main() {
   flag.Parse()
   songs, err := read_songs("umber.json")
   if err != nil {
      panic(err)
   }
   if len(os.Args) >= 3 {
      args := os.Args[2:]
      var songVar *song
      songVar, err = new_bandcamp().parse(args)
      if err != nil {
         panic(err)
      }
      songs = append([]*song{songVar}, songs...)
      var buf bytes.Buffer
      enc := json.NewEncoder(&buf)
      enc.SetEscapeHTML(false)
      enc.SetIndent("", " ")
      err := enc.Encode(songs)
      if err != nil {
         panic(err)
      }
      err = os.WriteFile("umber.json", buf.Bytes(), os.ModePerm)
      if err != nil {
         panic(err)
      }
   } else {
      new_http().f.Usage()
      new_bandcamp().f.Usage()
      new_soundcloud().f.Usage()
      new_youtube().f.Usage()
   }
}

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
   var songVar song
   songVar.S = detail.TralbumArtist + " - " + detail.Title
   songVar.Q = url.Values{
      "a": {strconv.FormatInt(time.Now().Unix(), 36)},
      "b": {strconv.Itoa(params.Iid)},
      "c": {strconv.FormatInt(detail.ArtId, 10)},
      "p": {"b"},
      "y": {
         strconv.Itoa(detail.Time().Year()),
      },
   }.Encode()
   return &songVar, nil
}
