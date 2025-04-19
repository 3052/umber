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

func (i *InnerTube) Player() (*Player, error) {
   i.ContentCheckOk = true
   i.RacyCheckOk = true
   i.Context.Client.ClientName = "IOS"
   i.Context.Client.ClientVersion = "19.45.4"
   data, err := json.MarshalIndent(i, "", " ")
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
   req.Header.Set("user-agent", user_agent + i.Context.Client.ClientVersion)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   play := &Player{}
   err = json.NewDecoder(resp.Body).Decode(play)
   if err != nil {
      return nil, err
   }
   return play, nil
}

type Player struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author string
      LengthSeconds int64 `json:",string"`
      ShortDescription string
      Title string
      VideoId string
      ViewCount int64 `json:",string"`
   }
}

const user_agent = "com.google.android.youtube/"

// need `osVersion` this to get the correct:
// This video requires payment to watch
// instead of the invalid:
// This video can only be played on newer versions of Android or other
// supported devices.
type InnerTube struct {
   ContentCheckOk bool `json:"contentCheckOk,omitempty"`
   Context struct {
      Client struct {
         AndroidSdkVersion int `json:"androidSdkVersion"`
         ClientName string `json:"clientName"`
         ClientVersion string `json:"clientVersion"`
         OsVersion string `json:"osVersion"`
      } `json:"client"`
   } `json:"context"`
   RacyCheckOk bool `json:"racyCheckOk,omitempty"`
   VideoId string `json:"videoId"`
}
