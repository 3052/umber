package main

import (
   "encoding/json"
   "flag"
   "log"
   "os"
   "path/filepath"
   "strings"
)

func main() {
   log.SetFlags(log.Ltime)

   inputFile := flag.String("input", "", "input JSON file path (required on first run)")
   flag.Parse()

   // ── Output directory ─────────────────────────────────────────────

   outputDir := filepath.Join(".", "umber")
   if err := os.MkdirAll(outputDir, 0755); err != nil {
      log.Fatalf("cannot create output dir: %v", err)
   }

   // ── Config ───────────────────────────────────────────────────────

   configDir, err := os.UserConfigDir()
   if err != nil {
      log.Fatalf("cannot determine config dir: %v", err)
   }
   configPath := filepath.Join(configDir, "umber", "umber.json")

   // Load existing config if it exists
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
   if *inputFile != "" {
      inputPath = *inputFile
      cfg.InputFile = inputPath
      saveConfig(configPath, cfg)
   } else if cfg.InputFile != "" {
      inputPath = cfg.InputFile
   } else {
      log.Fatal("-input is required on first run")
   }

   // ── Read input ──────────────────────────────────────────────────

   fileData, err := os.ReadFile(inputPath)
   if err != nil {
      log.Fatalf("cannot read input file: %v", err)
   }

   var records []Record
   if err := json.Unmarshal(fileData, &records); err != nil {
      log.Fatalf("cannot parse input JSON: %v", err)
   }

   // ── Collect video IDs (records without P) ──────────────────────

   videoIDs := make(map[string]bool)
   for _, r := range records {
      if r.P != "" {
         continue
      }
      if r.I != "" {
         videoIDs[r.I] = true
      }
   }

   // ── Scan output directory for existing audio files ───────────────

   entries, err := os.ReadDir(outputDir)
   if err != nil {
      log.Fatalf("cannot read output directory: %v", err)
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
         path := filepath.Join(outputDir, filename)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove %s: %v", path, err)
         } else {
            log.Printf("removed %s", path)
         }
      }
   }

   // ── Download missing / empty files ────────────────────────────────

   for vid := range videoIDs {
      if nonEmpty[vid] {
         continue
      }
      if err := downloadVideo(vid, cfg.VisitorID, outputDir); err != nil {
         log.Printf("error downloading %s: %v", vid, err)
      }
   }
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

// Config is persisted to os.UserConfigDir()/umber/umber.json.
type Config struct {
   VisitorID string `json:"visitor_id"`
   InputFile string `json:"input_file"`
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
