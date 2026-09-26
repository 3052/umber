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
   "os"
   "path/filepath"
   "strings"
)

// downloadBandcamp downloads the mp3-128 stream for a bandcamp.com track URL
// such as https://intlanthem.bandcamp.com/track/waiting, then remuxes it
// with ffmpeg to tag artist and title from the record. The track page is
// fetched once; its data-tralbum attribute embeds the streaming URL, so no separate
// ID-resolution or mobile-API request is needed. title is the record's
// filename stem. The max-eta limit does not apply here — it is
// YouTube-only.
func downloadBandcamp(r *Record, title, outputDir string) error {
   details, err := fetch_tralbum(r.I)
   if err != nil {
      return fmt.Errorf("resolve tralbum from %s: %w", r.I, err)
   }
   if len(details.TrackInfo) == 0 {
      return fmt.Errorf("no tracks in bandcamp response")
   }

   audioURL := details.TrackInfo[0].File["mp3-128"]
   if audioURL == "" {
      return fmt.Errorf("no mp3-128 stream URL found")
   }

   const ext = ".mp3"
   name := sanitizeFilename(title, ext, outputDir)
   finalPath := filepath.Join(outputDir, name+ext)
   dlPath := filepath.Join(outputDir, name+".t")
   // Same ".remux." temp marker as YouTube's, so isTempFile covers it.
   ffTmp := filepath.Join(outputDir, name+".remux."+ext)

   // maxETA 0 disables the ETA check: Bandcamp items are never skipped
   // for slow transfers.
   if err := downloadFileSingle(audioURL, dlPath, 0); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      return err
   }

   if err := remuxTagged(dlPath, ffTmp, r); err != nil {
      if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove download tmp: %w", err)
      }
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp: %w", err)
      }
      return err
   }

   if err := os.Remove(dlPath); err != nil && !os.IsNotExist(err) {
      return fmt.Errorf("remove download tmp: %w", err)
   }
   if err := os.Rename(ffTmp, finalPath); err != nil {
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp after rename fail: %w", err)
      }
      return fmt.Errorf("rename file: %w", err)
   }

   log.Printf("%s  done", filepath.Base(finalPath))
   return nil
}

// bandcampTrack mirrors one entry of the trackinfo array in the
// data-tralbum blob; File carries the mp3-128 streaming URL.
type bandcampTrack struct {
   File map[string]string `json:"file"`
}

// tralbum mirrors the subset of the data-tralbum JSON blob embedded in a
// Bandcamp track page that is needed to stream the audio.
type tralbum struct {
   TrackInfo []bandcampTrack `json:"trackinfo"`
}

// fetch_tralbum fetches a Bandcamp track page and extracts the track data
// from its data-tralbum attribute in a single request. The blob carries
// everything the old two-request flow gathered — including the mp3-128
// stream URL in trackinfo[0].file.
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
