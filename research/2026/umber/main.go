package main

import (
   "encoding/json"
   "flag"
   "log"
   "os"
   "path/filepath"
   "regexp"
   "strings"
)

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func main() {
   inputFile := flag.String("input", "", "input JSON file path")
   visitorID := flag.String("visitor-id", "", "X-Goog-Visitor-Id value (required on first run)")
   flag.Parse()

   if *inputFile == "" {
      log.Fatal("-input is required")
   }

   // ── Visitor ID / Config ───────────────────────────────────────────

   configDir, err := os.UserConfigDir()
   if err != nil {
      log.Fatalf("cannot determine config dir: %v", err)
   }
   configPath := filepath.Join(configDir, "umber", "umber.json")

   var visitorId string

   if *visitorID != "" {
      visitorId = *visitorID
      if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
         log.Fatalf("cannot create config dir: %v", err)
      }
      cfg := Config{VisitorID: visitorId}
      data, err := json.MarshalIndent(cfg, "", "  ")
      if err != nil {
         log.Fatalf("cannot marshal config: %v", err)
      }
      if err := os.WriteFile(configPath, data, 0644); err != nil {
         log.Fatalf("cannot write config: %v", err)
      }
      log.Printf("visitor ID saved to %s", configPath)
   } else {
      data, err := os.ReadFile(configPath)
      if err != nil {
         log.Fatalf("cannot read config (%s) — provide -visitor-id on first run: %v", configPath, err)
      }
      var cfg Config
      if err := json.Unmarshal(data, &cfg); err != nil {
         log.Fatalf("cannot parse config: %v", err)
      }
      if cfg.VisitorID == "" {
         log.Fatal("visitor_id is empty in config — provide -visitor-id flag")
      }
      visitorId = cfg.VisitorID
   }

   // ── Read input ────────────────────────────────────────────────────

   data, err := os.ReadFile(*inputFile)
   if err != nil {
      log.Fatalf("cannot read input file: %v", err)
   }

   var records []Record
   if err := json.Unmarshal(data, &records); err != nil {
      log.Fatalf("cannot parse input JSON: %v", err)
   }

   // ── Collect video IDs (records without P) ────────────────────────

   videoIDs := make(map[string]bool)
   for _, r := range records {
      if r.P != "" {
         continue
      }
      if r.I != "" {
         videoIDs[r.I] = true
      }
   }

   // ── Scan current directory for existing audio files ───────────────

   entries, err := os.ReadDir(".")
   if err != nil {
      log.Fatalf("cannot read current directory: %v", err)
   }

   allFiles := make(map[string]string) // videoID -> filename
   nonEmpty := make(map[string]bool)   // videoIDs with non-empty files

   for _, entry := range entries {
      if entry.IsDir() {
         continue
      }
      name := entry.Name()
      ext := filepath.Ext(name)
      base := strings.TrimSuffix(name, ext)
      if !videoIDPattern.MatchString(base) {
         continue
      }
      allFiles[base] = name
      info, err := entry.Info()
      if err != nil {
         continue
      }
      if info.Size() > 0 {
         nonEmpty[base] = true
      }
   }

   // ── Delete files not in input ─────────────────────────────────────

   for vid, filename := range allFiles {
      if !videoIDs[vid] {
         if err := os.Remove(filename); err != nil {
            log.Printf("cannot remove %s: %v", filename, err)
         } else {
            log.Printf("removed %s", filename)
         }
      }
   }

   // ── Download missing / empty files ────────────────────────────────

   for vid := range videoIDs {
      if nonEmpty[vid] {
         continue
      }
      if err := downloadVideo(vid, visitorId); err != nil {
         log.Printf("error downloading %s: %v", vid, err)
      }
   }
}

// Config is persisted to os.UserConfigDir()/umber/umber.json.
type Config struct {
   VisitorID string `json:"visitor_id"`
}

// Record represents one entry in the input JSON.
type Record struct {
   A string `json:"A"`
   D int64  `json:"D"`
   I string `json:"I"`
   P string `json:"P"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

// main.go
