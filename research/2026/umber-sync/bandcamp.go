// bandcamp.go marker preserve
package main

import (
   "encoding/json"
   "errors"
   "fmt"
   "html"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
   "strconv"
   "strings"
   "time"
)

// downloadBandcamp downloads the mp3-128 stream for a bandcamp.com track URL
// such as https://intlanthem.bandcamp.com/track/waiting. The page is resolved
// to a track ID via its report params, and the mobile API provides the
// streaming URL.
func downloadBandcamp(address, title, outputDir string, maxETA time.Duration) error {
   trackID, err := fetch_tralbum_id(address)
   if err != nil {
      return fmt.Errorf("resolve tralbum from %s: %w", address, err)
   }
   detail, err := fetch_tralbum_details(trackID)
   if err != nil {
      return fmt.Errorf("bandcamp api: %w", err)
   }
   if len(detail.Tracks) == 0 {
      return fmt.Errorf("no tracks in bandcamp response")
   }

   audioURL := detail.Tracks[0].StreamingURL["mp3-128"]
   if audioURL == "" {
      return fmt.Errorf("no mp3-128 stream URL found")
   }

   name := sanitizeFilename(title, ".mp3", outputDir)
   finalPath := filepath.Join(outputDir, name+".mp3")
   dlPath := filepath.Join(outputDir, name+".t")

   if err := downloadFileSingle(audioURL, dlPath, maxETA); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      return err
   }

   if err := os.Rename(dlPath, finalPath); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp after rename fail: %w", err)
      }
      return fmt.Errorf("rename file: %w", err)
   }

   log.Printf("%s  done", filepath.Base(finalPath))
   return nil
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

type bandcampTrack struct {
   StreamingURL map[string]string `json:"streaming_url"`
}

// tralbumDetails is the tralbum_details response subset, with Tracks added
// for the mp3-128 streaming URL.
type tralbumDetails struct {
   ArtId         int    `json:"art_id"`
   BandcampURL   string `json:"bandcamp_url"`
   ReleaseDate   int64  `json:"release_date"`
   Title         string
   TralbumArtist string          `json:"tralbum_artist"`
   Tracks        []bandcampTrack `json:"tracks"`
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
