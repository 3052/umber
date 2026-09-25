// fix-bandcamp/main.go
//
// Standalone maintenance script for the umber songs JSON file.
// Walks every song whose "I" host ends with .bandcamp.com, re-fetches the
// track page, and overwrites A, R, T and Y. D, I, and entry order are
// preserved. Any failure aborts the run; the file is only written after
// every entry succeeds, so an abort leaves it unmodified.
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
   "os"
   "path/filepath"
   "strings"
   "time"
)

// releaseLayout matches Bandcamp dates like "12 Jun 2026 00:00:00 GMT".
const releaseLayout = "02 Jan 2006 15:04:05 GMT"

// fixSong overwrites A, R, T and Y from the Bandcamp track page.
// Any missing or unparseable value is an error.
func fixSong(s *song) error {
   details, err := fetchTralbum(s.I)
   if err != nil {
      return err
   }
   if details.Current.Title == "" {
      return errors.New("empty title")
   }
   if details.Artist == "" {
      return errors.New("empty artist")
   }
   if details.ArtId == 0 {
      return errors.New("empty art ID")
   }
   release, err := time.Parse(releaseLayout, details.AlbumReleaseDate)
   if err != nil {
      return err
   }
   // Artwork URL is derived from the art ID; the _2 suffix picks a
   // standard size.
   s.A = fmt.Sprintf("https://f4.bcbits.com/img/a%d_2", details.ArtId)
   s.R = details.Artist
   s.T = details.Current.Title
   s.Y = release.Year()
   return nil
}

// isBandcamp reports whether the song address is hosted on *.bandcamp.com.
func isBandcamp(address string) bool {
   host, _, _ := strings.Cut(strings.TrimPrefix(address, "https://"), "/")
   return strings.HasSuffix(host, ".bandcamp.com")
}

func main() {
   log.SetFlags(log.Ltime)

   name := flag.String("n", "", "songs JSON path (default: input_file from the umber config)")
   flag.Parse()

   inputPath, err := resolveInput(*name)
   if err != nil {
      log.Fatal(err)
   }
   log.Println("input:", inputPath)

   songs, err := readSongs(inputPath)
   if err != nil {
      log.Fatal(err)
   }

   for _, s := range songs {
      if !isBandcamp(s.I) {
         continue
      }
      log.Printf("bandcamp: %s", s.I)
      if err := fixSong(s); err != nil {
         log.Fatalf("%s: %v", s.I, err)
      }
   }

   if err := writeSongs(inputPath, songs); err != nil {
      log.Fatal(err)
   }
   log.Println("done")
}

// resolveInput returns the -n flag value, or falls back to the
// input_file recorded in os.UserConfigDir()/umber/umber.json.
func resolveInput(flagValue string) (string, error) {
   if flagValue != "" {
      return flagValue, nil
   }
   configDir, err := os.UserConfigDir()
   if err != nil {
      return "", err
   }
   data, err := os.ReadFile(filepath.Join(configDir, "umber", "umber.json"))
   if err != nil {
      if errors.Is(err, os.ErrNotExist) {
         return "", errors.New("-n is required when no umber config exists")
      }
      return "", err
   }
   var cfg struct {
      InputFile string `json:"input_file"`
   }
   if err := json.Unmarshal(data, &cfg); err != nil {
      return "", err
   }
   if cfg.InputFile == "" {
      return "", errors.New("umber config has no input_file; pass -n")
   }
   return cfg.InputFile, nil
}

// writeSongs rewrites the songs file with the same formatting umber
// uses (no HTML escaping, one-space indent) so diffs stay minimal.
func writeSongs(name string, songs []*song) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   if err := enc.Encode(songs); err != nil {
      return err
   }
   log.Println("WriteFile", name)
   return os.WriteFile(name, buf.Bytes(), 0644)
}

// song mirrors the schema written by umber's write_songs.
type song struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R,omitempty"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

func readSongs(name string) ([]*song, error) {
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

// fetchTralbum fetches a Bandcamp track page and extracts the track
// details from its data-tralbum attribute in a single request: artist,
// title, art ID and release date.
func fetchTralbum(address string) (*tralbum, error) {
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
