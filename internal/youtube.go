package main

import (
   "41.neocities.org/protobuf"
   "encoding/base64"
   "encoding/json"
   "errors"
   "io"
   "log"
   "net/http"
   "net/url"
   "strings"
   "time"
)

func visitor_id() string {
   var message protobuf.Message
   message.AddBytes(1, []byte("EAhlBRb7X70"))
   return base64.URLEncoding.EncodeToString(message.Marshal())
}

func main() {
   var req http.Request
   req.Header = http.Header{}
   req.Header["Accept"] = []string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"}
   req.Header["Accept-Language"] = []string{"en-us,en;q=0.5"}
   req.Header["Connection"] = []string{"close"}
   req.Header["Content-Type"] = []string{"application/json"}
   req.Header["Host"] = []string{"www.youtube.com"}
   req.Header["Origin"] = []string{"https://www.youtube.com"}
   req.Header["Sec-Fetch-Mode"] = []string{"navigate"}
   req.Header["User-Agent"] = []string{"com.google.ios.youtube/20.03.02 (iPhone16,2; U; CPU iOS 18_2_1 like Mac OS X;)"}
   req.Header["X-Youtube-Client-Name"] = []string{"5"}
   req.Header["X-Youtube-Client-Version"] = []string{"20.03.02"}
   req.Method = "POST"
   req.ProtoMajor = 1
   req.ProtoMinor = 1
   req.URL = &url.URL{}
   req.URL.Host = "www.youtube.com"
   req.URL.Path = "/youtubei/v1/player"
   req.URL.Scheme = "https"
   req.Header["X-Goog-Visitor-Id"] = []string{visitor_id()}
   for range 9 {
      var play player
      play.New(&req)
      audio, found := play.audio_quality_medium()
      if !found {
         panic(".audio_quality_medium()")
      }
      err := audio.get()
      if err != nil {
         panic(err)
      }
      time.Sleep(time.Second)
   }
}

type player struct {
   StreamingData struct {
      AdaptiveFormats []adaptive_format
   }
}

type adaptive_format struct {
   AudioQuality string
   Url          string
}

func (a *adaptive_format) get() error {
   resp, err := get(a.Url)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return errors.New(resp.Status)
   }
   return nil
}

func (p *player) New(req *http.Request) error {
   req.Body = io.NopCloser(strings.NewReader(body))
   log.Println(req.Method, req.URL)
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

func (p player) audio_quality_medium() (*adaptive_format, bool) {
   for _, format := range p.StreamingData.AdaptiveFormats {
      if format.AudioQuality == "AUDIO_QUALITY_MEDIUM" {
         return &format, true
      }
   }
   return nil, false
}

func get(s string) (*http.Response, error) {
   log.Println("GET", s)
   return http.Get(s)
}

const body = `
{
   "videoId": "bQtPzo-7AHs",
   "contentCheckOk": true,
   "racyCheckOk": true,
   "context": {
      "client": {
         "clientName": "IOS",
         "clientVersion": "20.03.02",
         "deviceMake": "Apple",
         "deviceModel": "iPhone16,2",
         "userAgent": "com.google.ios.youtube/20.03.02 (iPhone16,2; U; CPU iOS 18_2_1 like Mac OS X;)",
         "osName": "iPhone",
         "osVersion": "18.2.1.22C161",
         "hl": "en",
         "timeZone": "UTC",
         "utcOffsetMinutes": 0
      }
   },
   "playbackContext": {
      "contentPlaybackContext": {
         "html5Preference": "HTML5_PREF_WANTS"
      }
   }
}
`
