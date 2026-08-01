package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

var yt_imgs = []string{
   0:  "sddefault.webp",
   1:  "sddefault.jpg",
   2:  "sd1.webp",
   3:  "sd2.webp",
   4:  "sd3.webp",
   5:  "sd1.jpg",
   6:  "sd2.jpg",
   7:  "sd3.jpg",
   8:  "hqdefault.webp",
   9:  "hqdefault.jpg",
   10: "hq1.webp",
   11: "hq2.webp",
   12: "hq3.webp",
   13: "0.webp",
   14: "0.jpg",
   15: "hq1.jpg",
   16: "hq2.jpg",
   17: "hq3.jpg",
   18: "mqdefault.webp",
   19: "mqdefault.jpg",
   20: "mq1.webp",
   21: "mq2.webp",
   22: "mq3.webp",
   23: "mq1.jpg",
   24: "mq2.jpg",
   25: "mq3.jpg",
   26: "default.webp",
   27: "default.jpg",
   28: "1.webp",
   29: "2.webp",
   30: "3.webp",
   31: "1.jpg",
   32: "2.jpg",
   33: "3.jpg",
}

func do_video_id(video_id, name, visitorID string) error {
   raw_songs, err := read_songs(name)
   if err != nil {
      return err
   }
   seen := make(map[string]bool)
   var songs []map[string]any
   input_exists := false

   // Iterate through ALL existing records to filter out duplicates
   for _, song := range raw_songs {
      if id, ok := song["I"].(string); ok {
         // Check if the input we are trying to add already exists
         if id == video_id {
            input_exists = true
         }
         // If we haven't seen this ID yet in the loop, keep it and mark it as seen
         if !seen[id] {
            seen[id] = true
            songs = append(songs, song)
         }
      } else {
         // Safety fallback: if the record is missing the "I" string, keep it to prevent data loss
         songs = append(songs, song)
      }
   }

   if input_exists {
      // If pre-existing duplicates were found and cleaned from the file, save the clean file before exiting.
      if len(songs) < len(raw_songs) {
         log.Printf("Cleaned up %d pre-existing duplicate(s) in %s\n", len(raw_songs)-len(songs), name)
         _ = write_songs(name, songs)
      }
      return fmt.Errorf("duplicate found: video ID '%s' already exists in %s", video_id, name)
   }

   play, err := fetch_player(video_id, visitorID)
   if err != nil {
      return err
   }
   fmt.Println(play.VideoDetails.ShortDescription)

   image, err := get_image(video_id)
   if err != nil {
      return err
   }

   // Insert native map data
   song_data := map[string]any{
      "D": time.Now().Unix(),
      "I": video_id,
      "T": play.VideoDetails.Author + " - " + play.VideoDetails.Title,
      "Y": play.Microformat.PlayerMicroformatRenderer.PublishDate.Year(),
   }
   if image != "" {
      song_data["A"] = image
   }

   songs = slices.Insert(songs, 0, song_data)

   // Save the newly cleaned and updated list
   return write_songs(name, songs)
}

func get_image(video_id string) (string, error) {
   for index, name := range yt_imgs {
      var address string
      if strings.HasSuffix(name, ".webp") {
         address = "http://i.ytimg.com/vi_webp/" + video_id + "/" + name
      } else {
         address = "http://i.ytimg.com/vi/" + video_id + "/" + name
      }
      status, err := head(address)
      if err != nil {
         return "", err
      }
      if status == http.StatusOK {
         if index == 0 {
            return "", nil
         }
         return name, nil
      }
   }
   return "", nil
}

func head(address string) (int, error) {
   fmt.Println(address)
   resp, err := http.Head(address)
   if err != nil {
      return 0, err
   }
   defer resp.Body.Close()
   return resp.StatusCode, nil
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "", "input JSON file path (required on first run)")
   video_url := flag.String("u", "", "video URL")
   flag.Parse()

   // ── Config ───────────────────────────────────────────────────────

   configDir, err := os.UserConfigDir()
   if err != nil {
      log.Fatalf("cannot determine config dir: %v", err)
   }
   configPath := filepath.Join(configDir, "umber", "umber.json")

   var cfg Config
   if data, err := os.ReadFile(configPath); err == nil {
      if err := json.Unmarshal(data, &cfg); err != nil {
         log.Fatalf("cannot parse config: %v", err)
      }
   }

   // ── Visitor ID ──────────────────────────────────────────────────

   if cfg.VisitorID == "" {
      visitorId, err := fetchVisitorID()
      if err != nil {
         log.Fatalf("cannot fetch visitor ID: %v", err)
      }
      cfg.VisitorID = visitorId
      log.Printf("visitor ID fetched")
      saveConfig(configPath, cfg)
   }

   // ── Input file ──────────────────────────────────────────────────

   var inputPath string
   if *name != "" {
      inputPath = *name
      cfg.InputFile = inputPath
      saveConfig(configPath, cfg)
   } else if cfg.InputFile != "" {
      inputPath = cfg.InputFile
   } else {
      log.Fatal("-n is required on first run")
   }

   if *video_url != "" {
      u, err := url.Parse(*video_url)
      if err != nil {
         log.Fatal("Invalid URL:", err)
      }

      video_id := u.Query().Get("v")
      if video_id == "" {
         log.Fatal("Could not extract 'v' parameter from URL")
      }

      err = do_video_id(video_id, inputPath, cfg.VisitorID)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func read_songs(name string) ([]map[string]any, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []map[string]any
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

// saveConfig writes the config to disk.
func saveConfig(configPath string, cfg Config) {
   if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
      log.Fatalf("cannot create config dir: %v", err)
   }
   data, err := json.MarshalIndent(cfg, "", "  ")
   if err != nil {
      log.Fatalf("cannot marshal config: %v", err)
   }
   if err := os.WriteFile(configPath, data, 0644); err != nil {
      log.Fatalf("cannot write config: %v", err)
   }
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

// Helper to handle the repeating logic of formatting and writing JSON
func write_songs(name string, songs []map[string]any) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err := enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
}

// Config is persisted to os.UserConfigDir()/umber/umber.json.
type Config struct {
   VisitorID string `json:"visitor_id"`
   InputFile string `json:"input_file"`
}

type player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate time.Time
      }
   }
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author           string
      LengthSeconds    int64 `json:",string"`
      ShortDescription string
      Title            string
      VideoId          string
      ViewCount        int64 `json:",string"`
   }
}

func fetch_player(video_id, visitorID string) (*player, error) {
   data, err := json.Marshal(map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "ANDROID_VR",
            "clientVersion": "1.65.10",
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
   })
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("X-Goog-Visitor-Id", visitorID)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   result := &player{}
   err = json.NewDecoder(resp.Body).Decode(result)
   if err != nil {
      return nil, err
   }
   return result, nil
}

// youtube-insert.go
