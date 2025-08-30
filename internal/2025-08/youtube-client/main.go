package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "io"
   "log"
   "net/http"
   "time"
)

func main() {
   log.SetFlags(log.Ltime)
   for _, client := range clients {
      play, err := client.player()
      if err != nil {
         log.Println(err, client)
      } else {
         log.Println(play.PlayabilityStatus, client)
      }
      //i := slices.IndexFunc(play.StreamingData.AdaptiveFormats,
      //   func(a *adaptive_format) bool {
      //      return a.AudioQuality == "AUDIO_QUALITY_MEDIUM"
      //   },
      //)
      //status, err := get_status(play.StreamingData.AdaptiveFormats[i].Url)
      //if err != nil {
      //   panic(err)
      //}
      //fmt.Println(status)
      time.Sleep(time.Second)
   }
}

func (c *ClientVersion) player() (*player, error) {
   value := map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":   c.Name,
            "clientVersion": c.Version,
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
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

func get_status(url string) (string, error) {
   resp, err := http.Get(url)
   if err != nil {
      return "", err
   }
   defer resp.Body.Close()
   _, err = io.Copy(io.Discard, resp.Body)
   if err != nil {
      return "", err
   }
   return resp.Status, nil
}

type adaptive_format struct {
   AudioQuality string
   Itag         int
   MimeType     string
   Url          string
}

const (
   // youtube.com/watch?v=fix-RSKlccw
   video_id   = "fix-RSKlccw"
   visitor_id = "CgtNbzlJR19GY24tNCjl_pDABjIKCgJVUxIEGgAgDA=="
)
