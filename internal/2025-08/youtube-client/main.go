package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "net/http"
   "time"
)

func (c *ClientVersion) player() (*player, error) {
   value := map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    c.name,
            "clientVersion": c.version,
         },
      },
      "racyCheckOk": true,
      "videoId":     c.video_id,
   }
   data, err := json.Marshal(value)
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
   play := &player{}
   err = json.NewDecoder(resp.Body).Decode(play)
   if err != nil {
      return nil, err
   }
   return play, nil
}

func main() {
   check_ok := flag.Bool("c", false, "check OK")
   flag.Parse()
   for _, client := range clients {
      var ok bool
      if *check_ok {
         ok = client.check_ok()
      } else {
         ok = client.check_not_ok()
      }
      if !ok {
         panic(client)
      }
      time.Sleep(100 * time.Millisecond)
   }
}

func (c *ClientVersion) check_not_ok() bool {
   if c.status == ok {
      return true
   }
   if c.status == no_longer_supported {
      return true
   }
   if c.status == sign_in {
      return true
   }
   if c.video_id == "" {
      return false
   }
   if c.version == "" {
      return false
   }
   play, err := c.player()
   if err != nil {
      fmt.Println(err, c)
   } else {
      fmt.Printf("%+v %v\n", play.PlayabilityStatus, c)
   }
   return true
}

func (c *ClientVersion) check_ok() bool {
   if c.status == ok {
      if c.version == "" {
         return false
      }
      if c.video_id == "" {
         return false
      }
      play, err := c.player()
      if err != nil {
         fmt.Println(err, c)
      } else {
         fmt.Printf("%+v %v\n", play.PlayabilityStatus, c)
      }
   }
   return true
}

type player struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   StreamingData struct {
      AdaptiveFormats []*adaptive_format
   }
   VideoDetails struct {
      Author  string
      Title   string
      VideoId string
   }
}

type adaptive_format struct {
   AudioQuality string
   Itag         int
   MimeType     string
   Url          string
}

//func get_status(url string) (string, error) {
//   resp, err := http.Get(url)
//   if err != nil {
//      return "", err
//   }
//   defer resp.Body.Close()
//   _, err = io.Copy(io.Discard, resp.Body)
//   if err != nil {
//      return "", err
//   }
//   return resp.Status, nil
//}

//func main() {
//   for _, client := range clients {
//      play, err := client.player()
//      if err != nil {
//         fmt.Println(err, client)
//      } else {
//         fmt.Println(play.PlayabilityStatus, client)
//      }
//      i := slices.IndexFunc(play.StreamingData.AdaptiveFormats,
//         func(a *adaptive_format) bool {
//            return a.AudioQuality == "AUDIO_QUALITY_MEDIUM"
//         },
//      )
//      status, err := get_status(play.StreamingData.AdaptiveFormats[i].Url)
//      if err != nil {
//         panic(err)
//      }
//      fmt.Println(status)
//      time.Sleep(100 * time.Millisecond)
//   }
//}
