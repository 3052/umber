// soundcloud.go marker preserve
package main

import (
   "cmp"
   "encoding/json"
   "fmt"
   "net/http"
   "net/url"
   "slices"
   "time"
)

// clientID identifies us as SoundCloud's public web client.
const clientID = "KKzJxmw11tYpCs6T24P4uUYhqmjalG6M"

// do_soundcloud adds a SoundCloud track to the songs file.
func do_soundcloud(address, name string) error {
   track, err := fetch_resolve(address)
   if err != nil {
      return err
   }
   if track.Title == "" {
      return fmt.Errorf("%s: resolve response has empty title", address)
   }

   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   if contains_song(songs, address) {
      return fmt.Errorf("duplicate found: '%s' already exists in %s", address, name)
   }

   song_data := song{
      D: time.Now().Unix(),
      I: address,
      R: track.User.Username,
      T: track.Title,
      Y: track.DisplayDate.Year(),
   }
   image := track.artwork()
   if image != "" {
      song_data.A = image
   }

   songs = append(songs, &song_data)
   slices.SortFunc(songs, func(a, b *song) int {
      return cmp.Compare(b.D, a.D)
   })

   return write_songs(name, songs)
}

// resolve mirrors the fields we need from the JSON answer of the
// SoundCloud resolve endpoint.
type resolve struct {
   ArtworkUrl  string    `json:"artwork_url"`
   DisplayDate time.Time `json:"display_date"`
   Id          int64
   Title       string
   User        struct {
      AvatarUrl string `json:"avatar_url"`
      Username  string
   }
}

// fetch_resolve runs an address through the SoundCloud resolve endpoint,
// which answers with the canonical track data in a single request.
func fetch_resolve(address string) (*resolve, error) {
   req, err := http.NewRequest(
      http.MethodGet, "https://api-v2.soundcloud.com/resolve", nil,
   )
   if err != nil {
      return nil, err
   }
   req.URL.RawQuery = url.Values{
      "client_id": {clientID},
      "url":       {address},
   }.Encode()

   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("resolve: %s", resp.Status)
   }

   result := &resolve{}
   if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
      return nil, err
   }
   return result, nil
}

// artwork returns the track artwork, falling back to the artist avatar
// when the track has no artwork of its own.
func (r *resolve) artwork() string {
   if r.ArtworkUrl != "" {
      return r.ArtworkUrl
   }
   return r.User.AvatarUrl
}

// soundcloud.go marker preserve
