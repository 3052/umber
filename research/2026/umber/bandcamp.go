package main

import (
   "encoding/json"
   "fmt"
   "log"
   "net/http"
   "os"
   "path/filepath"
   "time"
)

func downloadBandcamp(trackID, title, outputDir string, maxETA time.Duration) error {
   endpoint := fmt.Sprintf(
      "https://bandcamp.com/api/mobile/24/tralbum_details?band_id=1&tralbum_id=%s&tralbum_type=t",
      trackID,
   )

   resp, err := http.Get(endpoint)
   if err != nil {
      return fmt.Errorf("bandcamp api request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("bandcamp api returned status %d", resp.StatusCode)
   }

   var data bandcampResponse
   if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
      return fmt.Errorf("decode bandcamp response: %w", err)
   }
   if len(data.Tracks) == 0 {
      return fmt.Errorf("no tracks in bandcamp response")
   }

   audioURL := data.Tracks[0].StreamingURL["mp3-128"]
   if audioURL == "" {
      return fmt.Errorf("no mp3-128 stream URL found")
   }

   finalPath := filepath.Join(outputDir, sanitizeFilename(title)+".mp3")
   dlPath := filepath.Join(outputDir, sanitizeFilename(title)+".mp3.tmp")

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

type bandcampResponse struct {
   Tracks []bandcampTrack `json:"tracks"`
}

type bandcampTrack struct {
   StreamingURL map[string]string `json:"streaming_url"`
}

// bandcamp.go
