package main

import (
   "encoding/json"
   "fmt"
   "net/url"
   "os"
   "time"
)

const start = 721

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
      if i < start {
         continue
      }
      query, err := url.ParseQuery(song.Q)
      if err != nil {
         panic(err)
      }
      if query.Get("p") == "y" {
         var tube InnerTube
         tube.VideoId = query.Get("b")
         tube.Context.Client.ClientName = "ANDROID"
         play, err := tube.Player()
         if err != nil {
            panic(err)
         }
         fmt.Println(play.PlayabilityStatus.Status, tube.VideoId, len(songs)-i)
         time.Sleep(99 * time.Millisecond)
      }
   }
}
