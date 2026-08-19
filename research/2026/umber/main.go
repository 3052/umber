package main

import (
   "cmp"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "time"
)

const m3uFileName = "!playlist.m3u"

// validExts are the audio file extensions this program produces.
var validExts = map[string]bool{
   ".opus": true,
   ".m4a":  true,
   ".mp3":  true,
}

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

func generateM3U(outputDir string, records []Record) error {
   var items []*Record
   for i, r := range records {
      if (r.P != "" && r.P != "bandcamp") || r.I == "" || r.T == "" {
         continue
      }
      items = append(items, &records[i])
   }

   slices.SortFunc(items, func(a, b *Record) int {
      return cmp.Compare(b.D, a.D)
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
      if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".m3u") {
         continue
      }
      base := strings.TrimSuffix(name, filepath.Ext(name))
      titleToFile[base] = name
   }

   m3uPath := filepath.Join(outputDir, m3uFileName)
   out, err := os.Create(m3uPath)
   if err != nil {
      return fmt.Errorf("create m3u file: %w", err)
   }
   defer out.Close()

   fmt.Fprintln(out, "#EXTM3U")

   trackNum := 0
   for _, item := range items {
      filename, exists := titleToFile[sanitizeFilename(item.T)]
      if !exists {
         continue
      }
      trackNum++
      fmt.Fprintf(out, "#EXTINF:0,%s\n", item.T)
      fmt.Fprintf(out, "%s\n", filename)
   }

   log.Printf("M3U file generated: %s (%d tracks)", m3uPath, trackNum)
   return nil
}

func main() {
   log.SetFlags(log.Ltime)

   inputFile := flag.String("input", "", "input JSON file path (required)")
   outputDir := flag.String("output", "", "output directory (required)")
   threads := flag.Int("threads", 2, "number of download threads per item")
   maxETA := flag.Duration("max-eta", time.Minute, "maximum ETA; items exceeding this are skipped")
   flag.Parse()

   if *threads < 1 || *outputDir == "" || *inputFile == "" {
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

   fileData, err := os.ReadFile(*inputFile)
   if err != nil {
      log.Fatalf("cannot read input file: %v", err)
   }

   var records []Record
   if err := json.Unmarshal(fileData, &records); err != nil {
      log.Fatalf("cannot parse input JSON: %v", err)
   }

   titleToRecord := make(map[string]Record)
   for _, r := range records {
      if (r.P != "" && r.P != "bandcamp") || r.I == "" || r.T == "" {
         continue
      }
      titleToRecord[sanitizeFilename(r.T)] = r
   }

   // ── Delete files not in input ────────────────────────────────────

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
      if name == m3uFileName {
         continue
      }
      ext := strings.ToLower(filepath.Ext(name))
      if !validExts[ext] {
         path := filepath.Join(*outputDir, name)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove non-audio file %s: %v", path, err)
         } else {
            log.Printf("removed non-audio file %s", path)
         }
         continue
      }
      base := strings.TrimSuffix(name, filepath.Ext(name))
      allFiles[base] = name
      if info, err := entry.Info(); err == nil && info.Size() > 0 {
         nonEmpty[base] = true
      }
   }

   for title, filename := range allFiles {
      if _, exists := titleToRecord[title]; !exists {
         path := filepath.Join(*outputDir, filename)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove %s: %v", path, err)
         } else {
            log.Printf("removed %s", path)
         }
      }
   }

   // ── Download missing / empty files ────────────────────────────────

   for title, r := range titleToRecord {
      if nonEmpty[title] {
         continue
      }
      var err error
      switch r.P {
      case "bandcamp":
         err = downloadBandcamp(r.I, title, *outputDir, *maxETA)
      default:
         err = downloadVideo(r.I, title, cfg.VisitorID, *outputDir, *threads, *maxETA)
      }
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

   if err := generateM3U(*outputDir, records); err != nil {
      log.Printf("error generating M3U file: %v", err)
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
