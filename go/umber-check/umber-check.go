package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "net/http"
   "net/url"
   "os"
   "time"
)

const visitor_id = "CgtveHIwVmJ4Z3pySSjF3JHABjIKCgJVUxIEGgAgKw=="

func main() {
   name := flag.String("n", "umber.json", "name")
   start := flag.Int("s", -1, "start")
   flag.Parse()
   if *start <= -1 {
      flag.Usage()
      return
   }
   file, err := os.Open(*name)
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
         video_id := query.Get("b")
         var play Player
         err = play.New(video_id)
         if err != nil {
            panic(err)
         }
         fmt.Println(i, len(songs), video_id, song.S)
         if play.PlayabilityStatus.Status != "OK" {
            fmt.Printf("%+v\n", play.PlayabilityStatus)
            break
         }
         time.Sleep(99 * time.Millisecond)
      }
   }
}

type Player struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author           string
      LengthSeconds    int64 `json:",string"`
      ShortDescription string
      Title            string
      VideoId          string
      ViewCount        int64 `json:",string"`
   }
}

func (p *Player) New(video_id string) error {
   value := map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "IOS",
            "clientVersion": "20.03.02",
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
   }
   data, err := json.MarshalIndent(value, "", " ")
   if err != nil {
      return err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return err
   }
   req.Header.Set("x-goog-visitor-id", visitor_id)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return errors.New(resp.Status)
   }
   return json.NewDecoder(resp.Body).Decode(p)
}
