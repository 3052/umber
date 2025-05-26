package main

import (
   "41.neocities.org/platform/soundcloud"
   "bytes"
   "encoding/json"
   "flag"
   "net/url"
   "os"
   "path"
   "strconv"
   "strings"
   "time"
)

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

///

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
      var song1 *song
      song1, err = new_soundcloud().parse(args)
      if err != nil {
         panic(err)
      }
      songs = append([]*song{song1}, songs...)
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
   var song1 song
   song1.S = resolve.Title
   song1.Q = url.Values{
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
   return &song1, nil
}
