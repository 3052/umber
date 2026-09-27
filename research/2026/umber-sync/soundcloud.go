// soundcloud.go marker preserve
package main

import (
   "encoding/json"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
   "strings"
)

// clientID identifies us as SoundCloud's public web client — the same
// value the adder sends to the resolve endpoint, so both programs break
// together when SoundCloud rotates it.
const clientID = "KKzJxmw11tYpCs6T24P4uUYhqmjalG6M"

// downloadSoundCloud downloads the mp3 stream for a soundcloud.com track
// URL such as https://soundcloud.com/forss/flickermood, then remuxes it
// with ffmpeg to tag artist and title from the record. The resolve
// endpoint is fetched once; among its transcodings the progressive mp3
// is a plain signed file downloaded exactly like Bandcamp's stream —
// no HLS path is needed. Output is always .mp3. The max-eta limit does
// not apply here — it is YouTube-only.
func downloadSoundCloud(r *Record, title, outputDir string) error {
   track, err := fetchResolve(r.I)
   if err != nil {
      return fmt.Errorf("resolve track from %s: %w", r.I, err)
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

   for i := range track.Media.Transcodings {
      t := &track.Media.Transcodings[i]
      // Snipped transcodings are 30-second previews of Go+ tracks.
      if t.Snipped {
         continue
      }
      mime := strings.TrimSpace(strings.Split(t.Format.MimeType, ";")[0])
      if mime == "audio/mpeg" && t.Format.Protocol == "progressive" {
         return downloadSoundCloudFile(t, r, title, outputDir)
      }
   }
   return fmt.Errorf("no progressive mp3 stream found for %s", r.I)
}

// downloadSoundCloudFile downloads the progressive mp3 in one request,
// then remuxes it with ffmpeg to tag artist and title from the record
// before the rename.
func downloadSoundCloudFile(t *soundcloudTranscoding, r *Record, title, outputDir string) error {
   audioURL, err := fetchStreamURL(t.URL)
   if err != nil {
      return err
   }

   const ext = ".mp3"
   name := sanitizeFilename(title, outputDir)
   finalPath := filepath.Join(outputDir, name+ext)
   dlPath := filepath.Join(outputDir, name+".t")
   // Same ".remux." temp marker as YouTube's, so isTempFile covers it.
   ffTmp := filepath.Join(outputDir, name+".remux."+ext)

   // maxETA 0 disables the ETA check: SoundCloud items are never
   // skipped for slow transfers.
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

// fetchStreamURL exchanges a transcoding endpoint for the signed stream
// URL it hands out for this client id.
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
// asked with the client_id.
type soundcloudTranscoding struct {
   URL     string `json:"url"`
   Snipped bool   `json:"snipped"`
   Format  struct {
      Protocol string `json:"protocol"`
      MimeType string `json:"mime_type"`
   } `json:"format"`
}

// soundcloud.go marker preserve
