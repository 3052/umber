// main.go marker preserve
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
      if entry.IsDir() {
         continue
      }
      name := entry.Name()
      if !strings.HasSuffix(name, ".tmp") && !strings.HasSuffix(name, ".ff") && !strings.HasSuffix(name, ".t") {
         continue
      }
      path := filepath.Join(outputDir, name)
      if err := os.Remove(path); err != nil {
         log.Printf("cannot remove tmp file %s: %v", path, err)
      } else {
         log.Printf("removed tmp file %s", path)
      }
   }
}

// countStems counts, for every supported record, how many records sanitize
// to the same filename stem. Stems are compared lowercased (so names
// differing only in case collide, as they do on case-insensitive
// filesystems) and keyed with the expected extension (so a Bandcamp or
// SoundCloud .mp3 and a YouTube .opus sharing a title are not duplicates
// of each other).
func countStems(records []Record, outputDir string) map[string]int {
   counts := make(map[string]int)
   for _, r := range records {
      if r.I == "" || r.T == "" {
         continue
      }
      ext, ok := stemExt(r.I)
      if !ok {
         continue
      }
      stem := sanitizeFilename(r.baseName(), ext, outputDir)
      counts[strings.ToLower(stem)+ext]++
   }
   return counts
}

// fileStem returns the filename stem for r: the sanitized base name, plus
// " " + recordID when stemCounts shows another record sharing that stem
// case-insensitively. The suffix lets distinct items whose names differ
// only in case coexist on case-insensitive filesystems; non-duplicates keep
// their exact names. Identical records (same ID) collapse to one stem.
func fileStem(r Record, stemCounts map[string]int, outputDir string) string {
   ext, ok := stemExt(r.I)
   if !ok {
      return ""
   }
   stem := sanitizeFilename(r.baseName(), ext, outputDir)
   if stemCounts[strings.ToLower(stem)+ext] < 2 {
      return stem
   }
   if id := recordID(r.I); id != "" {
      stem = sanitizeFilename(stem+" "+id, ext, outputDir)
   }
   return stem
}

func generateM3U(outputDir string, records []Record) error {
   stemCounts := countStems(records, outputDir)

   var items []*Record
   for i, r := range records {
      if r.I == "" || r.T == "" {
         continue
      }
      switch platformOf(r.I) {
      case platformBandcamp, platformYouTube, platformSoundCloud:
         items = append(items, &records[i])
      }
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
      if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".ff") || strings.HasSuffix(name, ".t") || strings.HasSuffix(name, ".m3u") {
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
      stem := fileStem(*item, stemCounts, outputDir)
      if stem == "" {
         continue
      }
      filename, exists := titleToFile[stem]
      if !exists {
         continue
      }
      trackNum++
      fmt.Fprintf(out, "#EXTINF:0,%s\n", item.baseName())
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
      saveConfig(configPath, &cfg)
   }

   fileData, err := os.ReadFile(*inputFile)
   if err != nil {
      log.Fatalf("cannot read input file: %v", err)
   }

   var records []Record
   if err := json.Unmarshal(fileData, &records); err != nil {
      log.Fatalf("cannot parse input JSON: %v", err)
   }

   stemCounts := countStems(records, *outputDir)

   titleToRecord := make(map[string]Record)
   for _, r := range records {
      if r.I == "" || r.T == "" {
         continue
      }
      stem := fileStem(r, stemCounts, *outputDir)
      if stem == "" {
         continue
      }
      titleToRecord[stem] = r
   }

   // ── Delete files not in input ────────────────────────────────────

   entries, err := os.ReadDir(*outputDir)
   if err != nil {
      log.Fatalf("cannot read output directory: %v", err)
   }

   allFiles := make(map[string]string)
   nonEmpty := make(map[string]bool)

   for _, entry := range entries {
      if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") || strings.HasSuffix(entry.Name(), ".ff") || strings.HasSuffix(entry.Name(), ".t") {
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
      switch platformOf(r.I) {
      case platformBandcamp:
         err = downloadBandcamp(r.I, title, *outputDir, *maxETA)
      case platformSoundCloud:
         err = downloadSoundCloud(r.I, title, *outputDir, *maxETA)
      case platformYouTube:
         err = downloadVideo(r.I, title, cfg.VisitorID, *outputDir, *threads, *maxETA)
      }
      if err != nil {
         if errors.Is(err, errVisitorExpired) {
            log.Printf("visitor ID expired, clearing from config: %v", err)
            cfg.VisitorID = ""
            saveConfig(configPath, &cfg)
            return
         }
         log.Printf("error downloading %s: %v", title, err)
      }
   }

   if err := generateM3U(*outputDir, records); err != nil {
      log.Printf("error generating M3U file: %v", err)
   }
}

func saveConfig(configPath string, cfg *Config) {
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

// stemExt returns the extension a record's output file is expected to use,
// which sizes the filename truncation cap. ok is false for unsupported
// platforms.
func stemExt(raw string) (ext string, ok bool) {
   switch platformOf(raw) {
   case platformBandcamp, platformSoundCloud:
      return ".mp3", true
   case platformYouTube:
      return ".opus", true
   }
   return "", false
}

// Config is persisted to os.UserConfigDir()/umber/umber.json.
type Config struct {
   VisitorID string `json:"visitor_id"`
}

// Record represents one entry in the input JSON. I is the item URL, T the
// required title, R the optional author, and A the optional artwork URL.
type Record struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R,omitempty"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

// baseName returns the filename stem for the record: the author (R) when
// present, followed by the title (T).
func (r Record) baseName() string {
   if r.R != "" {
      return r.R + " - " + r.T
   }
   return r.T
}

// main.go marker preserve
