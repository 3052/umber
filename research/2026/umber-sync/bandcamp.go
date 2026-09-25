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
// to a tralbum ID via its report params, and the mobile API provides the
// streaming URL. An album URL would download its first track.
func downloadBandcamp(address, title, outputDir string, maxETA time.Duration) error {
   var params ReportParams
   if err := params.New(address); err != nil {
      return fmt.Errorf("resolve tralbum from %s: %w", address, err)
   }
   tralbum, ok := params.Tralbum()
   if !ok {
      return fmt.Errorf("unsupported tralbum type %q", params.Itype)
   }
   detail, err := tralbum.Details()
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

// ReportParams holds the tralbum reference embedded in a bandcamp page's
// "report-account-vm" tag.
type ReportParams struct {
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

// New fetches the track page and extracts the data-tou-report-params
// attribute, whose HTML-entity-encoded JSON identifies the tralbum.
func (r *ReportParams) New(address string) error {
   resp, err := http.Get(address)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("fetch page: status %d", resp.StatusCode)
   }

   var b strings.Builder
   if _, err := io.Copy(&b, resp.Body); err != nil {
      return err
   }
   data := b.String()

   // Locate the report-account-vm <p> tag.
   _, data, found := strings.Cut(data, `<p id="report-account-vm"`)
   if !found {
      return errors.New("report-account-vm not found")
   }

   // Locate the data-tou-report-params attribute inside that tag.
   _, data, found = strings.Cut(data, `data-tou-report-params="`)
   if !found {
      return errors.New("data-tou-report-params not found")
   }

   // The attribute value ends at the next double quote. The value is
   // HTML-entity-encoded JSON (e.g. &quot; for "), so unescape it.
   value, _, found := strings.Cut(data, `"`)
   if !found {
      return errors.New("data-tou-report-params: closing quote not found")
   }

   return json.Unmarshal([]byte(html.UnescapeString(value)), r)
}

// Tralbum returns the tralbum reference for known item types
// ("a" album, "t" track).
func (r *ReportParams) Tralbum() (*Tralbum, bool) {
   switch r.Itype {
   case "a":
      return &Tralbum{Id: r.Iid, Type: 'a'}, true
   case "t":
      return &Tralbum{Id: r.Iid, Type: 't'}, true
   }
   return nil, false
}

// Tralbum identifies a bandcamp album or track by numeric ID.
type Tralbum struct {
   Id   int
   Type byte
}

// Details fetches the tralbum details from bandcamp's mobile API.
func (t *Tralbum) Details() (*TralbumDetails, error) {
   query := url.Values{
      "band_id":      {"1"},
      "tralbum_id":   {strconv.Itoa(t.Id)},
      "tralbum_type": {string(t.Type)},
   }
   resp, err := http.Get("https://bandcamp.com/api/mobile/24/tralbum_details?" + query.Encode())
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("bandcamp api returned status %d", resp.StatusCode)
   }

   detail := &TralbumDetails{}
   if err := json.NewDecoder(resp.Body).Decode(detail); err != nil {
      return nil, err
   }
   return detail, nil
}

// TralbumDetails is the subset of the tralbum_details response the
// downloader needs; Tracks carries the mp3-128 streaming URL.
type TralbumDetails struct {
   ArtId         int    `json:"art_id"`
   BandcampURL   string `json:"bandcamp_url"`
   ReleaseDate   int64  `json:"release_date"`
   Title         string
   TralbumArtist string          `json:"tralbum_artist"`
   Tracks        []bandcampTrack `json:"tracks"`
}

type bandcampTrack struct {
   StreamingURL map[string]string `json:"streaming_url"`
}

// bandcamp.go marker preserve
