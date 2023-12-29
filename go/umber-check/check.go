package main

import (
   "154.pages.dev/media/youtube"
   "encoding/json"
   "fmt"
   "net/url"
   "os"
   "time"
)

func main() {
   file, err := os.Open("../../docs/umber.json")
   if err != nil {
      panic(err)
   }
   defer file.Close()
   var songs []struct {
      Q string
   }
   json.NewDecoder(file).Decode(&songs)
   for i, song := range songs {
      if i >= 1033 {
         query, err := url.ParseQuery(song.Q)
         if err != nil {
            panic(err)
         }
         if query.Get("p") == "y" {
            var req youtube.Request
            req.Video_ID = query.Get("b")
            req.Android()
            play, err := req.Player(nil)
            if err != nil {
               panic(err)
            }
            fmt.Println(play.Playability.Status, req.Video_ID, len(songs)-i)
            time.Sleep(99*time.Millisecond)
         }
      }
   }
}
