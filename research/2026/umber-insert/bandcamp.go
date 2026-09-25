// bandcamp.go marker preserve
package main

import (
   "cmp"
   "encoding/json"
   "errors"
   "fmt"
   "html"
   "io"
   "net/http"
   "slices"
   "strings"
   "time"
)

// do_bandcamp adds a Bandcamp track to the songs file.
func do_bandcamp(address, name string) error {
   details, err := fetch_tralbum(address)
   if err != nil {
      return err
   }

   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   if contains_song(songs, details.URL) {
      return fmt.Errorf("duplicate found: '%s' already exists in %s", details.URL, name)
   }

   // Dates look like "12 Jun 2026 00:00:00 GMT".
   release, err := time.Parse("02 Jan 2006 15:04:05 GMT", details.AlbumReleaseDate)
   if err != nil {
      return err
   }

   songs = append(songs, &song{
      // Artwork URL is derived from the art ID; the _2 suffix picks a
      // standard size.
      A: fmt.Sprintf("https://f4.bcbits.com/img/a%d_2", details.ArtId),
      D: time.Now().Unix(),
      I: details.URL,
      R: details.Artist,
      T: details.Current.Title,
      Y: release.Year(),
   })
   slices.SortFunc(songs, func(a, b *song) int {
      return cmp.Compare(b.D, a.D)
   })

   return write_songs(name, songs)
}

// tralbum mirrors the fields we need from the data-tralbum JSON blob
// embedded in a Bandcamp track page.
type tralbum struct {
   Artist           string `json:"artist"`
   ArtId            int    `json:"art_id"`
   URL              string `json:"url"`
   AlbumReleaseDate string `json:"album_release_date"`
   Current          struct {
      Title string `json:"title"`
   } `json:"current"`
}

// fetch_tralbum fetches a Bandcamp track page and extracts the track
// details from its data-tralbum attribute in a single request. The JSON
// blob carries everything the old two-request flow gathered: artist,
// title, art ID, release date and the canonical URL.
func fetch_tralbum(address string) (*tralbum, error) {
   resp, err := http.Get(address)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("tralbum page: %s", resp.Status)
   }

   var page strings.Builder
   if _, err := io.Copy(&page, resp.Body); err != nil {
      return nil, err
   }

   // The tralbum data is attached to a <script> tag as an HTML-entity-
   // encoded JSON attribute. The attribute value ends at the next double
   // quote. Note that data-tralbum-collect-info doesn't match this cut,
   // because of the required quote right after the attribute name.
   rest := page.String()

   _, rest, found := strings.Cut(rest, `data-tralbum="`)
   if !found {
      return nil, errors.New("data-tralbum not found")
   }

   attr, _, found := strings.Cut(rest, `"`)
   if !found {
      return nil, errors.New("data-tralbum: closing quote not found")
   }

   result := new(tralbum)
   if err := json.Unmarshal([]byte(html.UnescapeString(attr)), result); err != nil {
      return nil, err
   }
   return result, nil
}

// bandcamp.go marker preserve
