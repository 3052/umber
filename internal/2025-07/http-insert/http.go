package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "log"
   "net/url"
   "os"
   "path"
   "slices"
   "strconv"
   "time"
)

func (f *flags) do_http() error {
   // 1 values
   now := strconv.FormatInt(time.Now().Unix(), 36)
   values := url.Values{}
   values.Set("a", now)
   values.Set("b", f.audio)
   values.Set("c", f.image)
   values.Set("p", "h")
   values.Set("y", f.year)
   // 2 song
   var songVar song
   songVar.Q = values.Encode()
   songVar.S = path.Base(f.audio)
   // 3 songs
   songs, err := read_songs(f.name)
   if err != nil {
      return err
   }
   songs = slices.Insert(songs, 0, songVar)
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err = enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file("umber.json", buf.Bytes())
}

type song struct {
   Q string
   S string
}

func read_songs(name string) ([]song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

func main() {
   log.SetFlags(log.Ltime)
   var f flags
   flag.StringVar(&f.audio, "a", "", "audio")
   flag.StringVar(&f.image, "i", "", "image")
   flag.StringVar(&f.name, "n", "umber.json", "name")
   flag.StringVar(&f.year, "y", "", "year")
   flag.Parse()
   if f.audio != "" {
      err := f.do_http()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

type flags struct {
   audio string
   image string
   name  string
   year  string
}
