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

type player struct {
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

func (p *player) New(visitor_id, video_id string) error {
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

func main() {
   visitor_id := flag.String("v", "", "visitor ID")
   name := flag.String("n", "umber.json", "name")
   start := flag.Int("s", 0, "start")
   flag.Parse()
   if *visitor_id != "" {
      err := do_check(*visitor_id, *name, *start)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

func do_check(visitor_id, name string, start int) error {
   file, err := os.Open(name)
   if err != nil {
      return err
   }
   defer file.Close()
   var songs []struct {
      Q string
      S string
   }
   err = json.NewDecoder(file).Decode(&songs)
   if err != nil {
      return err
   }
   for i, song := range songs {
      if i >= start {
         query, err := url.ParseQuery(song.Q)
         if err != nil {
            return err
         }
         if query.Get("p") == "y" {
            video_id := query.Get("b")
            var play player
            err = play.New(visitor_id, video_id)
            if err != nil {
               return err
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
   return nil
}
