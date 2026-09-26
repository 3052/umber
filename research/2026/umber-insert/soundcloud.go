// soundcloud.go marker preserve
package main

import (
   "cmp"
   "encoding/json"
   "fmt"
   "net/http"
   "net/url"
   "slices"
   "strings"
   "time"
)

// clientID identifies us as SoundCloud's public web client — the same
// value the downloader sends to the resolve endpoint, so both programs
// break together when SoundCloud rotates it.
const clientID = "KKzJxmw11tYpCs6T24P4uUYhqmjalG6M"

// do_soundcloud adds a SoundCloud track to the songs file. The duplicate
// check runs first, before any network request: its key is the pasted
// address. The insert is allowed only when the resolve response itself
// shows the plain progressive mp3 stream the downloader fetches — no
// second request is needed to learn that.
func do_soundcloud(address, name string) error {
   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   if contains_song(songs, address) {
      return fmt.Errorf("duplicate found: '%s' already exists in %s", address, name)
   }

   track, err := fetch_resolve(address)
   if err != nil {
      return err
   }
   if track.Kind != "track" {
      return fmt.Errorf("%s: resolve returned %s, want a track URL", address, track.Kind)
   }
   if track.Title == "" {
      return fmt.Errorf("%s: resolve response has empty title", address)
   }

   // The insert is allowed only when the track offers the stream the
   // downloader fetches.
   if err := verify_audio(track); err != nil {
      return err
   }

   // Every track has artwork, or at worst the artist avatar as fallback.
   song_data := song{
      A: track.artwork(),
      D: time.Now().Unix(),
      I: address,
      R: track.User.Username,
      T: track.Title,
      Y: track.DisplayDate.Year(),
   }

   songs = append(songs, &song_data)
   slices.SortFunc(songs, func(a, b *song) int {
      return cmp.Compare(b.D, a.D)
   })

   return write_songs(name, songs)
}

// verify_audio reports whether the track offers the progressive mp3
// stream the downloader fetches, using only what the resolve response
// already carries. It walks the same selection the downloader does: no
// DRM, a policy that allows a full stream, and a non-snipped progressive
// mp3 transcoding.
func verify_audio(track *resolve) error {
   // DRM: SoundCloud serves some tracks (major-label catalog) only via
   // encrypted protocols (cbc-encrypted-hls, ctr-encrypted-hls). The API
   // still lists the plain legacy transcodings, but their stream
   // endpoints answer 404 — tombstones. yt-dlp detects the same tracks
   // by skipping protocols starting with "ctr-"/"cbc-".
   for i := range track.Media.Transcodings {
      if strings.Contains(track.Media.Transcodings[i].Format.Protocol, "encrypted") {
         return fmt.Errorf("DRM track: only encrypted streams available")
      }
   }

   switch track.Policy {
   case "", "ALLOW", "MONETIZE":
   default:
      // SNIPPET is a SoundCloud Go+ preview, BLOCK is geo-blocked or
      // taken down; neither offers a full stream.
      return fmt.Errorf("policy %s: no full stream available", track.Policy)
   }

   for i := range track.Media.Transcodings {
      t := &track.Media.Transcodings[i]
      // Snipped transcodings are 30-second previews of Go+ tracks.
      if t.Snipped {
         continue
      }
      mime := strings.TrimSpace(strings.Split(t.Format.MimeType, ";")[0])
      if mime == "audio/mpeg" && t.Format.Protocol == "progressive" {
         return nil
      }
   }
   return fmt.Errorf("no progressive mp3 stream found")
}

// resolve mirrors the fields we need from the JSON answer of the
// SoundCloud resolve endpoint: the song metadata, plus the kind, policy
// and transcoding list needed to tell whether the track is downloadable
// before an insert.
type resolve struct {
   ArtworkUrl  string    `json:"artwork_url"`
   DisplayDate time.Time `json:"display_date"`
   Id          int64
   Kind        string
   Policy      string
   Title       string
   User        struct {
      AvatarUrl string `json:"avatar_url"`
      Username  string
   }
   Media struct {
      Transcodings []transcoding `json:"transcodings"`
   } `json:"media"`
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
      if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
         return nil, fmt.Errorf("resolve: %s (client_id rejected; SoundCloud may have rotated it)", resp.Status)
      }
      return nil, fmt.Errorf("resolve: %s", resp.Status)
   }

   result := &resolve{}
   if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
      return nil, err
   }
   return result, nil
}

// artwork returns the track artwork at the 500x500 size, falling back to
// the artist avatar when the track has no artwork of its own. The API
// serves a "-large" size; "-t500x500" is the larger square variant.
// https://i1.sndcdn.com/artworks-xYAsXaVtwPIt-0-t500x500.jpg
func (r *resolve) artwork() string {
   var address string
   if r.ArtworkUrl != "" {
      address = r.ArtworkUrl
   } else {
      address = r.User.AvatarUrl
   }
   return strings.Replace(address, "-large", "-t500x500", 1)
}

// transcoding mirrors one entry of the media.transcodings array; only
// the fields needed to tell whether the plain progressive mp3 stream
// exists are kept.
type transcoding struct {
   Snipped bool `json:"snipped"`
   Format  struct {
      Protocol string `json:"protocol"`
      MimeType string `json:"mime_type"`
   } `json:"format"`
}

// soundcloud.go marker preserve
