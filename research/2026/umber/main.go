package main

import (
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "sort"
   "strings"
   "time"
)

func cleanupTmpFiles(outputDir string) {
   entries, err := os.ReadDir(outputDir)
   if err != nil {
      return
   }
   for _, entry := range entries {
      if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmp") {
         continue
      }
      path := filepath.Join(outputDir, entry.Name())
      if err := os.Remove(path); err != nil {
         log.Printf("cannot remove tmp file %s: %v", path, err)
      } else {
         log.Printf("removed tmp file %s", path)
      }
   }
}

func generateCue(outputDir string, records []Record) error {
   var items []Record
   for _, r := range records {
      if r.P != "" || r.I == "" || r.T == "" {
         continue
      }
      items = append(items, r)
   }

   sort.Slice(items, func(i, j int) bool {
      return items[i].D > items[j].D
   })

   entries, err := os.ReadDir(outputDir)
   if err != nil {
      return fmt.Errorf("read output dir: %w", err)
   }
   titleToFile := make(map[string]string)
   for _, entry := range entries {
      if entry.IsDir() {
         continue
      }
      name := entry.Name()
      if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".cue") {
         continue
      }
      base := strings.TrimSuffix(name, filepath.Ext(name))
      titleToFile[base] = name
   }

   cuePath := filepath.Join(outputDir, "umber.cue")
   out, err := os.Create(cuePath)
   if err != nil {
      return fmt.Errorf("create cue file: %w", err)
   }
   defer out.Close()

   trackNum := 0
   for _, item := range items {
      filename, exists := titleToFile[sanitizeFilename(item.T)]
      if !exists {
         continue
      }
      trackNum++
      fmt.Fprintf(out, "FILE %q WAVE\n", filename)
      fmt.Fprintf(out, "  TRACK %02d AUDIO\n", trackNum)
      fmt.Fprintf(out, "    TITLE %q\n", item.T)
      fmt.Fprintf(out, "    INDEX 01 00:00:00\n")
   }

   log.Printf("CUE file generated: %s (%d tracks)", cuePath, trackNum)
   return nil
}

func main() {
   log.SetFlags(log.Ltime)

   inputFile := flag.String("input", "", "input JSON file path (required on first run)")
   outputDir := flag.String("output", "", "output directory (required)")
   threads := flag.Int("threads", 2, "number of download threads per item")
   maxETA := flag.Duration("max-eta", time.Minute, "maximum ETA; items exceeding this are skipped")
   flag.Parse()

   if *threads < 1 || *outputDir == "" {
      flag.Usage()
      return
   }

   if err := os.MkdirAll(*outputDir, 0755); err != nil {
      log.Fatalf("cannot create output dir: %v", err)
   }

   cleanupTmpFiles(*outputDir)

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

   if cfg.VisitorID == "" {
      visitorId, err := fetchVisitorID()
      if err != nil {
         log.Fatalf("cannot fetch visitor ID: %v", err)
      }
      cfg.VisitorID = visitorId
      log.Printf("visitor ID fetched")
      saveConfig(configPath, cfg)
   }

   var inputPath string
   if *inputFile != "" {
      inputPath = *inputFile
      cfg.InputFile = inputPath
      saveConfig(configPath, cfg)
   } else if cfg.InputFile != "" {
      inputPath = cfg.InputFile
   } else {
      flag.Usage()
      return
   }

   fileData, err := os.ReadFile(inputPath)
   if err != nil {
      log.Fatalf("cannot read input file: %v", err)
   }

   var records []Record
   if err := json.Unmarshal(fileData, &records); err != nil {
      log.Fatalf("cannot parse input JSON: %v", err)
   }

   titleToVideoID := make(map[string]string)
   for _, r := range records {
      if r.P != "" || r.I == "" || r.T == "" {
         continue
      }
      titleToVideoID[sanitizeFilename(r.T)] = r.I
   }

   entries, err := os.ReadDir(*outputDir)
   if err != nil {
      log.Fatalf("cannot read output directory: %v", err)
   }

   allFiles := make(map[string]string)
   nonEmpty := make(map[string]bool)

   for _, entry := range entries {
      if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") {
         continue
      }
      name := entry.Name()
      base := strings.TrimSuffix(name, filepath.Ext(name))
      allFiles[base] = name
      if info, err := entry.Info(); err == nil && info.Size() > 0 {
         nonEmpty[base] = true
      }
   }

   for title, filename := range allFiles {
      if _, exists := titleToVideoID[title]; !exists {
         path := filepath.Join(*outputDir, filename)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove %s: %v", path, err)
         } else {
            log.Printf("removed %s", path)
         }
      }
   }

   for title, videoID := range titleToVideoID {
      if nonEmpty[title] {
         continue
      }
      err := downloadVideo(videoID, title, cfg.VisitorID, *outputDir, *threads, *maxETA)
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

   if err := generateCue(*outputDir, records); err != nil {
      log.Printf("error generating CUE file: %v", err)
   }
}

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
