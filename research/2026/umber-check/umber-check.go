package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "net/http"
   "os"
   "time"
)

func do_check(name string, start int) error {
   file, err := os.Open(name)
   if err != nil {
      return err
   }
   defer file.Close()

   // Updated struct to match the new JSON format
   var songs []struct {
      D int64  `json:"D"`
      I string `json:"I"`
      T string `json:"T"`
      Y int    `json:"Y"`
      A string `json:"A,omitempty"`
      P string `json:"P,omitempty"`
   }

   err = json.NewDecoder(file).Decode(&songs)
   if err != nil {
      return err
   }

   for i, song := range songs {
      if i >= start {
         // If P is missing (empty string), then it's YouTube
         if song.P == "" {
            video_id := song.I // The ID is now stored in 'I'
            play, err := fetch_player(video_id)
            if err != nil {
               return err
            }

            // Output using the new title variable 'song.T'
            fmt.Println(i, len(songs)-i, video_id, song.T)

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

func main() {
   name := flag.String("n", "umber.json", "name")
   start := flag.Int("s", -1, "start")
   flag.Parse()
   if *start >= 0 {
      err := do_check(*name, *start)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
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

func fetch_player(video_id string) (*player, error) {
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
