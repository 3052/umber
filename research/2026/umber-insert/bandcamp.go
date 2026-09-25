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
   "net/url"
   "slices"
   "strconv"
   "strings"
   "time"
)

// do_bandcamp adds a Bandcamp track to the songs file.
func do_bandcamp(address, name string) error {
   trackID, err := fetch_tralbum_id(address)
   if err != nil {
      return err
   }
   details, err := fetch_tralbum_details(trackID)
   if err != nil {
      return err
   }
   if details.BandcampURL == "" {
      return errors.New("tralbum_details: bandcamp_url is empty")
   }

   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   if contains_song(songs, details.BandcampURL) {
      return fmt.Errorf("duplicate found: '%s' already exists in %s", details.BandcampURL, name)
   }

   songs = append(songs, &song{
      // Artwork URL is derived from the art ID; the _2 suffix picks a
      // standard size.
      A: fmt.Sprintf("https://f4.bcbits.com/img/a%d_2", details.ArtId),
      D: time.Now().Unix(),
      I: details.BandcampURL,
      R: details.TralbumArtist,
      T: details.Title,
      Y: time.Unix(details.ReleaseDate, 0).Year(),
   })
   slices.SortFunc(songs, func(a, b *song) int {
      return cmp.Compare(b.D, a.D)
   })

   return write_songs(name, songs)
}

// fetch_tralbum_id fetches a Bandcamp track page and extracts the track's
// numeric ID from its data-tou-report-params attribute.
func fetch_tralbum_id(address string) (int, error) {
   resp, err := http.Get(address)
   if err != nil {
      return 0, err
   }
   defer resp.Body.Close()

   var page strings.Builder
   if _, err := io.Copy(&page, resp.Body); err != nil {
      return 0, err
   }

   // The track ID lives in a JSON object on the report-account-vm tag,
   // stored as an HTML-entity-encoded attribute value.
   rest := page.String()

   _, rest, found := strings.Cut(rest, `<p id="report-account-vm"`)
   if !found {
      return 0, errors.New("report-account-vm not found")
   }

   _, rest, found = strings.Cut(rest, `data-tou-report-params="`)
   if !found {
      return 0, errors.New("data-tou-report-params not found")
   }

   // The attribute value ends at the next double quote.
   attr, _, found := strings.Cut(rest, `"`)
   if !found {
      return 0, errors.New("data-tou-report-params: closing quote not found")
   }

   var report struct {
      ID int `json:"i_id"`
   }
   if err := json.Unmarshal([]byte(html.UnescapeString(attr)), &report); err != nil {
      return 0, err
   }
   if report.ID == 0 {
      return 0, errors.New("data-tou-report-params: missing i_id")
   }
   return report.ID, nil
}

type tralbumDetails struct {
   ArtId         int    `json:"art_id"`
   BandcampURL   string `json:"bandcamp_url"`
   ReleaseDate   int64  `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

// fetch_tralbum_details fetches a track's details from Bandcamp's mobile API.
func fetch_tralbum_details(trackID int) (*tralbumDetails, error) {
   query := url.Values{
      "band_id":      {"1"},
      "tralbum_id":   {strconv.Itoa(trackID)},
      "tralbum_type": {"t"},
   }.Encode()
   endpoint := "https://bandcamp.com/api/mobile/24/tralbum_details?" + query
   resp, err := http.Get(endpoint)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("tralbum_details: %s", resp.Status)
   }
   details := new(tralbumDetails)
   if err := json.NewDecoder(resp.Body).Decode(details); err != nil {
      return nil, err
   }
   return details, nil
}

// bandcamp.go marker preserve
