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
            req.VideoId = query.Get("b")
            req.Android()
            var play youtube.Player
            err := play.Post(req, nil)
            if err != nil {
               panic(err)
            }
            fmt.Println(play.PlayabilityStatus.Status, req.VideoId, len(songs)-i)
            time.Sleep(99*time.Millisecond)
         }
      }
   }
}
