package youtube

import (
   "bytes"
   "encoding/json"
   "net/http"
   "net/url"
   "strings"
   "time"
)

func (i *InnerTube) Player(token *AuthToken) (*Player, error) {
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
   if token != nil {
      req.Header.Set("authorization", "Bearer " + token.AccessToken)
   }
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
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate Date
      }
   }
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   StreamingData struct {
      AdaptiveFormats []AdaptiveFormat
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

type Date struct {
   Time time.Time
}

func (d *Date) UnmarshalText(text []byte) error {
   var err error
   d.Time, err = time.Parse(time.RFC3339, string(text))
   if err != nil {
      return err
   }
   return nil
}
func (d *DeviceCode) String() string {
   var b strings.Builder
   b.WriteString("1. Go to\n")
   b.WriteString(d.VerificationUrl)
   b.WriteString("\n\n2. Enter this code\n")
   b.WriteString(d.UserCode)
   return b.String()
}

func (d *DeviceCode) New() error {
   resp, err := http.PostForm(
      "https://oauth2.googleapis.com/device/code",
      url.Values{
         "client_id": {client_id},
         "scope": {"https://www.googleapis.com/auth/youtube"},
      },
   )
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   return json.NewDecoder(resp.Body).Decode(d)
}

func (d *DeviceCode) Marshal() ([]byte, error) {
   return json.Marshal(d)
}

type DeviceCode struct {
   DeviceCode string `json:"device_code"`
   UserCode string `json:"user_code"`
   VerificationUrl string `json:"verification_url"`
}

func (d *DeviceCode) Unmarshal(text []byte) error {
   return json.Unmarshal(text, d)
}
// YouTube on TV
const (
   client_id = "861556708454-d6dlm3lh05idd8npek18k6be8ba3oc68.apps.googleusercontent.com"
   client_secret = "SboVhoG9s0rNafixCSGGKXAT"
)

func (a *AuthToken) Refresh() error {
   resp, err := http.PostForm(
      "https://oauth2.googleapis.com/token", url.Values{
         "client_id": {client_id},
         "client_secret": {client_secret},
         "grant_type": {"refresh_token"},
         "refresh_token": {a.RefreshToken},
      },
   )
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   return json.NewDecoder(resp.Body).Decode(a)
}

func (a *AuthToken) Marshal() ([]byte, error) {
   return json.Marshal(a)
}

type AuthToken struct {
   AccessToken string `json:"access_token"`
   RefreshToken string `json:"refresh_token"`
}

func (a *AuthToken) Unmarshal(text []byte) error {
   return json.Unmarshal(text, a)
}

func (d *DeviceCode) Token() (*AuthToken, error) {
   resp, err := http.PostForm(
      "https://oauth2.googleapis.com/token", url.Values{
         "client_id": {client_id},
         "client_secret": {client_secret},
         "device_code": {d.DeviceCode},
         "grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
      },
   )
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   token := &AuthToken{}
   err = json.NewDecoder(resp.Body).Decode(token)
   if err != nil {
      return nil, err
   }
   return token, nil
}
