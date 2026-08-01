package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "net/url"
   "os"
   "path/filepath"
   "slices"
   "time"
)

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
         if errors.Is(err, errVisitorExpired) {
            log.Printf("visitor ID expired, clearing from config: %v", err)
            cfg.VisitorID = ""
            saveConfig(configPath, cfg)
            return
         }
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

// youtube-insert.go
