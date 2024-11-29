package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "net/url"
   "os"
   "time"
)

func main() {
   start := flag.Int("s", -1, "start")
   flag.Parse()
   if *start <= -1 {
      flag.Usage()
      return
   }
   file, err := os.Open("umber.json")
   if err != nil {
      panic(err)
   }
   defer file.Close()
   var songs []struct {
      Q string
      S string
   }
   err = json.NewDecoder(file).Decode(&songs)
   if err != nil {
      panic(err)
   }
   for i, song := range songs {
      if i < *start {
         continue
      }
      query, err := url.ParseQuery(song.Q)
      if err != nil {
         panic(err)
      }
      if query.Get("p") == "y" {
         var tube InnerTube
         tube.VideoId = query.Get("b")
         play, err := tube.Player()
         if err != nil {
            panic(err)
         }
         fmt.Println(i, len(songs), tube.VideoId, song.S)
         if play.PlayabilityStatus.Status != "OK" {
            fmt.Println(play.PlayabilityStatus.Status)
            break
         }
         time.Sleep(99 * time.Millisecond)
      }
   }
}
