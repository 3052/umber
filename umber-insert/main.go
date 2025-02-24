package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "os"
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
      var song1 *song
      switch os.Args[1] {
      case "http":
         song1, err = new_http().parse(args)
      case "bandcamp":
         song1, err = new_bandcamp().parse(args)
      case "soundcloud":
         song1, err = new_soundcloud().parse(args)
      case "youtube":
         song1, err = new_youtube().parse(args)
      }
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
