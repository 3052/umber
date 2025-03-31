package youtube

import (
   "bytes"
   "encoding/json"
   "net/http"
   "strings"
   "time"
)

func (d *Date) UnmarshalText(data []byte) error {
   var err error
   d[0], err = time.Parse(time.RFC3339, string(data))
   if err != nil {
      return err
   }
   return nil
}

type Date [1]time.Time

const user_agent = "com.google.android.youtube/"

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

///

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

type YtImg struct {
   Height int
   Name string
   VideoId string
   Width int
}

var YtImgs = []YtImg{
   {Width:120, Height:90, Name:"default.jpg"},
   {Width:120, Height:90, Name:"1.jpg"},
   {Width:120, Height:90, Name:"2.jpg"},
   {Width:120, Height:90, Name:"3.jpg"},
   {Width:120, Height:90, Name:"default.webp"},
   {Width:120, Height:90, Name:"1.webp"},
   {Width:120, Height:90, Name:"2.webp"},
   {Width:120, Height:90, Name:"3.webp"},
   {Width:320, Height:180, Name:"mq1.jpg"},
   {Width:320, Height:180, Name:"mq2.jpg"},
   {Width:320, Height:180, Name:"mq3.jpg"},
   {Width:320, Height:180, Name:"mqdefault.jpg"},
   {Width:320, Height:180, Name:"mq1.webp"},
   {Width:320, Height:180, Name:"mq2.webp"},
   {Width:320, Height:180, Name:"mq3.webp"},
   {Width:320, Height:180, Name:"mqdefault.webp"},
   {Width:480, Height:360, Name:"0.jpg"},
   {Width:480, Height:360, Name:"hqdefault.jpg"},
   {Width:480, Height:360, Name:"hq1.jpg"},
   {Width:480, Height:360, Name:"hq2.jpg"},
   {Width:480, Height:360, Name:"hq3.jpg"},
   {Width:480, Height:360, Name:"0.webp"},
   {Width:480, Height:360, Name:"hqdefault.webp"},
   {Width:480, Height:360, Name:"hq1.webp"},
   {Width:480, Height:360, Name:"hq2.webp"},
   {Width:480, Height:360, Name:"hq3.webp"},
   {Width:640, Height:480, Name:"sddefault.jpg"},
   {Width:640, Height:480, Name:"sd1.jpg"},
   {Width:640, Height:480, Name:"sd2.jpg"},
   {Width:640, Height:480, Name:"sd3.jpg"},
   {Width:640, Height:480, Name:"sddefault.webp"},
   {Width:640, Height:480, Name:"sd1.webp"},
   {Width:640, Height:480, Name:"sd2.webp"},
   {Width:640, Height:480, Name:"sd3.webp"},
   {Width:1280, Height:720, Name:"hq720.jpg"},
   {Width:1280, Height:720, Name:"maxresdefault.jpg"},
   {Width:1280, Height:720, Name:"maxres1.jpg"},
   {Width:1280, Height:720, Name:"maxres2.jpg"},
   {Width:1280, Height:720, Name:"maxres3.jpg"},
   {Width:1280, Height:720, Name:"hq720.webp"},
   {Width:1280, Height:720, Name:"maxresdefault.webp"},
   {Width:1280, Height:720, Name:"maxres1.webp"},
   {Width:1280, Height:720, Name:"maxres2.webp"},
   {Width:1280, Height:720, Name:"maxres3.webp"},
}

func (y *YtImg) String() string {
   var data strings.Builder
   data.WriteString("http://i.ytimg.com/vi")
   if strings.HasSuffix(y.Name, ".webp") {
      data.WriteString("_webp")
   }
   data.WriteByte('/')
   data.WriteString(y.VideoId)
   data.WriteByte('/')
   data.WriteString(y.Name)
   return data.String()
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

type Player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate Date
      }
   }
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
