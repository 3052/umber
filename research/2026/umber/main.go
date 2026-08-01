package main

import (
   "encoding/json"
   "errors"
   "flag"
   "log"
   "os"
   "path/filepath"
   "strings"
   "time"
)

// cleanupTmpFiles removes any leftover .tmp files from interrupted downloads.
func cleanupTmpFiles(outputDir string) {
   entries, err := os.ReadDir(outputDir)
   if err != nil {
      return
   }
   for _, entry := range entries {
      if entry.IsDir() {
         continue
      }
      if strings.HasSuffix(entry.Name(), ".tmp") {
         path := filepath.Join(outputDir, entry.Name())
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove tmp file %s: %v", path, err)
         } else {
            log.Printf("removed tmp file %s", path)
         }
      }
   }
}

func main() {
   log.SetFlags(log.Ltime)

   inputFile := flag.String("input", "", "input JSON file path (required on first run)")
   threads := flag.Int("threads", 2, "number of download threads per item")
   maxETA := flag.Duration("max-eta", time.Minute, "maximum ETA; items exceeding this are skipped")
   flag.Parse()

   if *threads < 1 {
      log.Fatal("-threads must be at least 1")
   }

   // ── Output directory ─────────────────────────────────────────────

   outputDir := filepath.Join(".", "umber")
   if err := os.MkdirAll(outputDir, 0755); err != nil {
      log.Fatalf("cannot create output dir: %v", err)
   }

   // ── Clean up leftover .tmp files from interrupted downloads ───────

   cleanupTmpFiles(outputDir)

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

   // ── Collect sanitized title -> videoID (records without P) ────

   titleToVideoID := make(map[string]string)
   for _, r := range records {
      if r.P != "" {
         continue
      }
      if r.I != "" && r.T != "" {
         titleToVideoID[sanitizeFilename(r.T)] = r.I
      }
   }

   // ── Scan output directory for existing audio files ───────────────

   entries, err := os.ReadDir(outputDir)
   if err != nil {
      log.Fatalf("cannot read output directory: %v", err)
   }

   allFiles := make(map[string]string) // sanitized title -> filename
   nonEmpty := make(map[string]bool)   // sanitized titles with non-empty files

   for _, entry := range entries {
      if entry.IsDir() {
         continue
      }
      name := entry.Name()
      if strings.HasSuffix(name, ".tmp") {
         continue
      }
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

   for title, filename := range allFiles {
      if _, exists := titleToVideoID[title]; !exists {
         path := filepath.Join(outputDir, filename)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove %s: %v", path, err)
         } else {
            log.Printf("removed %s", path)
         }
      }
   }

   // ── Download missing / empty files (sequential) ───────────────────

   for title, videoID := range titleToVideoID {
      if nonEmpty[title] {
         continue
      }
      err := downloadVideo(videoID, title, cfg.VisitorID, outputDir, *threads, *maxETA)
      if err != nil {
         if errors.Is(err, errVisitorExpired) {
            log.Printf("visitor ID expired, clearing from config: %v", err)
            cfg.VisitorID = ""
            saveConfig(configPath, cfg)
            return
         }
         log.Printf("error downloading %s: %v", title, err)
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
