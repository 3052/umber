package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "time"
)

func fetch_player(visitor_id, video_id string) (*player, error) {
   data, err := json.Marshal(map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "IOS",
            "clientVersion": "20.03.02",
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
   })
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("x-goog-visitor-id", visitor_id)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   result := &player{}
   err = json.NewDecoder(resp.Body).Decode(result)
   if err != nil {
      return nil, err
   }
   return result, nil
}

func main() {
   visitor_id := flag.String("v", "", "visitor ID")
   name := flag.String("n", "umber.json", "name")
   start := flag.Int("s", 0, "start")
   flag.Parse()
   if *visitor_id != "" {
      err := do_check(*visitor_id, *name, *start)
      if err != nil {
         log.Fatal(err)
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
            play, err := fetch_player(visitor_id, video_id)
            if err != nil {
               return err
            }
            fmt.Println(i, len(songs)-i, video_id, song.S)
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
