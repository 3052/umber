package soundcloud

import (
   "encoding/json"
   "net/http"
   "net/url"
   "time"
)

const client_id = "KKzJxmw11tYpCs6T24P4uUYhqmjalG6M"

type Resolve struct {
   ArtworkUrl  string    `json:"artwork_url"`
   DisplayDate time.Time `json:"display_date"`
   Id          int64
   Title       string
   User        struct {
      AvatarUrl string `json:"avatar_url"`
      Username  string
   }
}

// i1.sndcdn.com/artworks-000308141235-7ep8lo-large.jpg
func (r *Resolve) Artwork() string {
   if r.ArtworkUrl != "" {
      return r.ArtworkUrl
   }
   return r.User.AvatarUrl
}

func (r *Resolve) New(url2 string) error {
   req, _ := http.NewRequest("", "https://api-v2.soundcloud.com/resolve", nil)
   req.URL.RawQuery = url.Values{
      "client_id": {client_id},
      "url":       {url2},
   }.Encode()
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   return json.NewDecoder(resp.Body).Decode(r)
}
