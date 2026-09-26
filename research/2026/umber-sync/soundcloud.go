// soundcloud.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path/filepath"
   "strings"
   "time"
)

// clientID identifies us as SoundCloud's public web client — the same
// value the adder sends to the resolve endpoint, so both programs break
// together when SoundCloud rotates it.
const clientID = "KKzJxmw11tYpCs6T24P4uUYhqmjalG6M"

// downloadSoundCloud downloads the mp3 stream for a soundcloud.com track
// URL such as https://soundcloud.com/forss/flickermood. The resolve
// endpoint is fetched once; among its transcodings the progressive mp3
// is preferred — a plain signed file, downloaded like Bandcamp's — with
// the HLS mp3 as a fallback, remuxed by ffmpeg like the YouTube stream.
// Output is always .mp3.
func downloadSoundCloud(address, title, outputDir string, maxETA time.Duration) error {
   track, err := fetchResolve(address)
   if err != nil {
      return fmt.Errorf("resolve track from %s: %w", address, err)
   }
   if track.Kind != "track" {
      return fmt.Errorf("resolve returned %s, want a track URL", track.Kind)
   }
   switch track.Policy {
   case "", "ALLOW", "MONETIZE":
   default:
      // SNIPPET is a SoundCloud Go+ preview, BLOCK is geo-blocked or
      // taken down; neither offers a full stream.
      return fmt.Errorf("policy %s: no full stream available", track.Policy)
   }

   var hls *soundcloudTranscoding
   for i := range track.Media.Transcodings {
      t := &track.Media.Transcodings[i]
      // Snipped transcodings are 30-second previews of Go+ tracks.
      if t.Snipped {
         continue
      }
      mime := strings.TrimSpace(strings.Split(t.Format.MimeType, ";")[0])
      if mime != "audio/mpeg" {
         continue
      }
      switch t.Format.Protocol {
      case "progressive":
         return downloadSoundCloudFile(t, title, outputDir, maxETA)
      case "hls":
         if hls == nil {
            hls = t
         }
      }
   }
   if hls == nil {
      return fmt.Errorf("no mp3 stream found for %s", address)
   }
   return downloadSoundCloudHLS(hls, title, outputDir)
}

// downloadSoundCloudFile downloads the progressive mp3 in one request,
// the same way Bandcamp tracks are saved.
func downloadSoundCloudFile(t *soundcloudTranscoding, title, outputDir string, maxETA time.Duration) error {
   audioURL, err := fetchStreamURL(t.URL)
   if err != nil {
      return err
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

// downloadSoundCloudHLS handles tracks that only offer the HLS mp3
// transcoding: ffmpeg downloads the playlist and remuxes the segments
// into one mp3, like the YouTube remux. The max-eta guard does not
// apply to this path.
func downloadSoundCloudHLS(t *soundcloudTranscoding, title, outputDir string) error {
   playlistURL, err := fetchStreamURL(t.URL)
   if err != nil {
      return err
   }

   name := sanitizeFilename(title, ".mp3", outputDir)
   finalPath := filepath.Join(outputDir, name+".mp3")
   ffTmp := filepath.Join(outputDir, name+".ff")

   cmd := exec.Command("ffmpeg", "-y", "-i", playlistURL, "-c", "copy", "-f", "mp3", ffTmp)
   var stderr bytes.Buffer
   cmd.Stderr = &stderr
   if err := cmd.Run(); err != nil {
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp: %w", err)
      }
      return fmt.Errorf("ffmpeg hls remux: %w\n%s", err, stderr.String())
   }
   if err := os.Rename(ffTmp, finalPath); err != nil {
      if err := os.Remove(ffTmp); err != nil && !os.IsNotExist(err) {
         return fmt.Errorf("remove ff tmp after rename fail: %w", err)
      }
      return fmt.Errorf("rename file: %w", err)
   }

   log.Printf("%s  remuxed", filepath.Base(finalPath))
   return nil
}

// fetchStreamURL exchanges a transcoding endpoint for the signed stream
// URL it hands out for this client id. Progressive answers with a plain
// CDN file; HLS answers with an m3u8 playlist.
func fetchStreamURL(transcodingURL string) (string, error) {
   u, err := url.Parse(transcodingURL)
   if err != nil {
      return "", err
   }
   q := u.Query()
   q.Set("client_id", clientID)
   u.RawQuery = q.Encode()

   resp, err := http.Get(u.String())
   if err != nil {
      return "", err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return "", fmt.Errorf("transcoding endpoint: %s", resp.Status)
   }

   var s struct {
      URL string `json:"url"`
   }
   if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
      return "", err
   }
   if s.URL == "" {
      return "", fmt.Errorf("transcoding endpoint returned no stream URL")
   }
   return s.URL, nil
}

// soundcloudTrack mirrors the subset of the resolve response needed to
// stream a track: the transcoding list, the resource kind, and the
// policy gate that marks Go+ preview (SNIPPET) and blocked tracks.
type soundcloudTrack struct {
   Kind   string `json:"kind"`
   Policy string `json:"policy"`
   Media  struct {
      Transcodings []soundcloudTranscoding `json:"transcodings"`
   } `json:"media"`
}

// fetchResolve runs a track address through SoundCloud's public resolve
// endpoint, which answers with the track data in a single request — the
// same call the adder makes; here only the stream information is needed.
func fetchResolve(address string) (*soundcloudTrack, error) {
   req, err := http.NewRequest(http.MethodGet, "https://api-v2.soundcloud.com/resolve", nil)
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

   result := new(soundcloudTrack)
   if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
      return nil, err
   }
   return result, nil
}

// soundcloudTranscoding mirrors one entry of the media.transcodings
// array; URL points at an endpoint that hands out the stream URL when
// asked with the client_id. Snipped marks 30-second Go+ previews.
type soundcloudTranscoding struct {
   URL     string `json:"url"`
   Snipped bool   `json:"snipped"`
   Format  struct {
      Protocol string `json:"protocol"`
      MimeType string `json:"mime_type"`
   } `json:"format"`
}

// soundcloud.go marker preserve
