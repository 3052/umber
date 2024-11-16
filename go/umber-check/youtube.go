package main

import (
   "bytes"
   "encoding/json"
   "net/http"
)

func (i *InnerTube) Player() (*Player, error) {
   i.Context.Client.AndroidSdkVersion = 32
   i.Context.Client.OsVersion = "12"
   switch i.Context.Client.ClientName {
   case android:
      i.ContentCheckOk = true
      i.Context.Client.ClientVersion = android_version
      i.RacyCheckOk = true
   case android_embedded_player:
      i.Context.Client.ClientVersion = android_version
   case web:
      i.Context.Client.ClientVersion = web_version
   }
   data, err := json.Marshal(i)
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

const (
   android_version = "19.33.35"
   web_version = "2.20231219.04.00"
)

const (
   android = "ANDROID"
   android_embedded_player = "ANDROID_EMBEDDED_PLAYER"
   web = "WEB"
)

var ClientName = []string{
   android,
   android_embedded_player,
   web,
}
