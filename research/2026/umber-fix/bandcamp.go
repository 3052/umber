// fixbandcamp/main.go
package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "html"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "strconv"
   "strings"
)

func do_fix_bandcamp(name string) error {
   songs, err := read_songs(name)
   if err != nil {
      return err
   }
   for _, s := range songs {
      u, err := url.Parse(s.I)
      if err != nil {
         return err
      }
      if !strings.HasSuffix(u.Host, ".bandcamp.com") {
         continue
      }
      if err := fix_song(s); err != nil {
         return fmt.Errorf("%s: %w", s.I, err)
      }
   }
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

// fix_song re-fetches a Bandcamp track and overwrites its artist and title.
// D, Y, A and I are left untouched.
func fix_song(s *song) error {
   trackID, err := fetch_tralbum_id(s.I)
   if err != nil {
      return err
   }
   details, err := fetch_tralbum_details(trackID)
   if err != nil {
      return err
   }
   if details.Title == "" || details.TralbumArtist == "" {
      return errors.New("tralbum_details: missing artist or title")
   }
   log.Printf("%s: R %q, T %q", s.I, details.TralbumArtist, details.Title)
   s.R = details.TralbumArtist
   s.T = details.Title
   return nil
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "", "input JSON file path (required)")
   flag.Parse()
   if *name == "" {
      flag.Usage()
      log.Fatal("-n is required")
   }
   if err := do_fix_bandcamp(*name); err != nil {
      log.Fatal(err)
   }
}

func write_songs(name string, songs []*song) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   if err := enc.Encode(songs); err != nil {
      return err
   }
   log.Println("WriteFile", name)
   return os.WriteFile(name, buf.Bytes(), os.ModePerm)
}

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
   if err := json.Unmarshal(data, &songs); err != nil {
      return nil, err
   }
   return songs, nil
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
