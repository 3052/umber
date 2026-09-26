// update.go marker preserve

// update refreshes SoundCloud entries in a songs file. For each item
// whose "I" host is exactly soundcloud.com, it makes the SoundCloud
// resolve request and overwrites A, R, T and Y in place. D, I and the
// file order are left untouched. The first error aborts the run.
//
// go run update/update.go -n /path/to/songs.json

package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "strings"
   "time"
)

// clientID identifies us as SoundCloud's public web client.
const clientID = "KKzJxmw11tYpCs6T24P4uUYhqmjalG6M"

// do_update rewrites A, R, T and Y for every song whose host is exactly
// soundcloud.com. Any error is returned immediately, before the file is
// written.
func do_update(name string) error {
   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   for _, s := range songs {
      u, err := url.Parse(s.I)
      if err != nil || u.Host != "soundcloud.com" {
         continue
      }

      log.Println("updating", s.I)
      track, err := fetch_resolve(s.I)
      if err != nil {
         return err
      }
      if track.Title == "" {
         return fmt.Errorf("%s: resolve response has empty title", s.I)
      }

      s.A = track.artwork()
      s.R = track.User.Username
      s.T = track.Title
      s.Y = track.DisplayDate.Year()
   }

   return write_songs(name, songs)
}

func main() {
   log.SetFlags(log.Ltime)

   name := flag.String("n", "", "input JSON file path (required)")
   flag.Parse()
   if *name == "" {
      log.Fatal("-n is required")
   }

   if err := do_update(*name); err != nil {
      log.Fatal(err)
   }
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

// write_songs keeps the exact serialization of the main program.
func write_songs(name string, songs []*song) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err := enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
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

// artwork returns the track artwork at the 500x500 size, falling back to
// the artist avatar when the track has no artwork of its own.
func (r *resolve) artwork() string {
   var address string
   if r.ArtworkUrl != "" {
      address = r.ArtworkUrl
   } else {
      address = r.User.AvatarUrl
   }
   return strings.Replace(address, "-large", "-t500x500", 1)
}

// song mirrors the JSON shape the main program reads and writes.
type song struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R,omitempty"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

func read_songs(name string) ([]*song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []*song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

// update.go marker preserve
