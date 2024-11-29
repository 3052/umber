package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "net/http"
   "net/url"
   "os"
   "time"
)

const start = 0

func main() {
   file, err := os.Open("../../docs/umber.json")
   if err != nil {
      panic(err)
   }
   defer file.Close()
   var songs []struct {
      Q string
   }
   err = json.NewDecoder(file).Decode(&songs)
   if err != nil {
      panic(err)
   }
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
         fmt.Println(tube.VideoId, len(songs)-i)
         if play.PlayabilityStatus.Status != "OK" {
            break
         }
         time.Sleep(99 * time.Millisecond)
      }
   }
}
