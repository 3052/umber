// main.go marker preserve
package main

import (
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "strings"
   "time"
)

// validExts are the audio file extensions this program produces.
var validExts = map[string]bool{
   ".opus": true,
   ".m4a":  true,
   ".mp3":  true,
}

// cleanupTmpFiles removes leftover in-flight temp files from interrupted
// runs: every file matching isTempFile in outputDir. A leftover remux
// temp is not just clutter — ffmpeg runs without -y and would refuse to
// overwrite it — so cleanup failure is an error, not a log line. All
// removal failures are joined so one bad file does not mask another.
func cleanupTmpFiles(outputDir string) error {
   entries, err := os.ReadDir(outputDir)
   if err != nil {
      return fmt.Errorf("read output dir: %w", err)
   }
   var errs []error
   for _, entry := range entries {
      if entry.IsDir() {
         continue
      }
      name := entry.Name()
      if !isTempFile(name) {
         continue
      }
      path := filepath.Join(outputDir, name)
      if err := os.Remove(path); err != nil {
         errs = append(errs, fmt.Errorf("remove %s: %w", path, err))
      } else {
         log.Printf("removed tmp file %s", path)
      }
   }
   return errors.Join(errs...)
}

func main() {
   log.SetFlags(log.Ltime)

   inputFile := flag.String("input", "", "input JSON file path (required)")
   outputDir := flag.String("output", "", "output directory (required)")
   threads := flag.Int("threads", 2, "number of download threads per item")
   maxETA := flag.Duration("max-eta", time.Minute, "maximum ETA for YouTube downloads; YouTube items exceeding this are skipped (0 disables)")
   flag.Parse()

   if *threads < 1 || *outputDir == "" || *inputFile == "" {
      flag.Usage()
      return
   }

   if err := run(*inputFile, *outputDir, *threads, *maxETA); err != nil {
      log.Fatal(err)
   }
}

// run performs the full sync after flag validation: prepares the output
// directory, loads config and input records, removes files no longer in
// the input, downloads missing or empty items (the max-eta limit applies
// to YouTube downloads only), and writes the M3U playlist. Conditions
// that abort the run are returned as errors for main to report; per-item
// failures (file removals, stats, downloads) are logged and skipped so
// one bad item does not abort the run. The visitor-expired return ends
// the run early so the next invocation refetches the visitor ID. Each
// download is announced with a log line counting the items still queued
// after it; the line carries only the count, since the item's filename
// follows immediately in the download's own progress lines.
func run(inputFile, outputDir string, threads int, maxETA time.Duration) error {
   if err := os.MkdirAll(outputDir, 0755); err != nil {
      return fmt.Errorf("cannot create output dir: %w", err)
   }

   if err := cleanupTmpFiles(outputDir); err != nil {
      return fmt.Errorf("cleanup tmp files: %w", err)
   }

   configDir, err := os.UserConfigDir()
   if err != nil {
      return fmt.Errorf("cannot determine config dir: %w", err)
   }
   configPath := filepath.Join(configDir, "umber", "umber.json")

   var cfg Config
   if data, rerr := os.ReadFile(configPath); rerr != nil {
      if !errors.Is(rerr, os.ErrNotExist) {
         return fmt.Errorf("cannot read config: %w", rerr)
      }
      // No config file yet: proceed with an empty one.
   } else if uerr := json.Unmarshal(data, &cfg); uerr != nil {
      return fmt.Errorf("cannot parse config: %w", uerr)
   }

   if cfg.VisitorID == "" {
      visitorId, err := fetchVisitorID()
      if err != nil {
         return fmt.Errorf("cannot fetch visitor ID: %w", err)
      }
      cfg.VisitorID = visitorId
      log.Printf("visitor ID fetched")
      saveConfig(configPath, &cfg)
   }

   fileData, err := os.ReadFile(inputFile)
   if err != nil {
      return fmt.Errorf("cannot read input file: %w", err)
   }

   var records []Record
   if err := json.Unmarshal(fileData, &records); err != nil {
      return fmt.Errorf("cannot parse input JSON: %w", err)
   }

   stemCounts := countStems(records, outputDir)

   // A record whose URL does not parse is fatal: it would silently drop
   // out of titleToRecord below and the sweep would then delete its
   // existing file as unreferenced. fileStem returns that error.
   titleToRecord := make(map[string]Record)
   for _, r := range records {
      stem, serr := fileStem(&r, stemCounts, outputDir)
      if serr != nil {
         return serr
      }
      if stem == "" {
         continue
      }
      titleToRecord[stem] = r
   }

   // ── Delete files not in input ────────────────────────────────────

   entries, err := os.ReadDir(outputDir)
   if err != nil {
      return fmt.Errorf("cannot read output directory: %w", err)
   }

   allFiles := make(map[string]string)
   nonEmpty := make(map[string]bool)

   for _, entry := range entries {
      if entry.IsDir() || isTempFile(entry.Name()) {
         continue
      }
      name := entry.Name()
      if name == m3uFileName {
         continue
      }
      ext := strings.ToLower(filepath.Ext(name))
      if !validExts[ext] {
         path := filepath.Join(outputDir, name)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove non-audio file %s: %v", path, err)
         } else {
            log.Printf("removed non-audio file %s", path)
         }
         continue
      }
      base := strings.TrimSuffix(name, filepath.Ext(name))
      allFiles[base] = name
      info, ierr := entry.Info()
      if ierr != nil {
         log.Printf("cannot stat %s: %v", name, ierr)
      } else if info.Size() > 0 {
         nonEmpty[base] = true
      }
   }

   for title, filename := range allFiles {
      if _, exists := titleToRecord[title]; !exists {
         path := filepath.Join(outputDir, filename)
         if err := os.Remove(path); err != nil {
            log.Printf("cannot remove %s: %v", path, err)
         } else {
            log.Printf("removed %s", path)
         }
      }
   }

   // ── Download missing / empty files ────────────────────────────────

   // remaining is the number of items this run still has to fetch. It
   // counts only missing or empty files — up-to-date items never
   // download — and is logged as each download starts, so progress
   // through the queue stays visible during long transfers.
   remaining := 0
   for title := range titleToRecord {
      if !nonEmpty[title] {
         remaining++
      }
   }

   for title, r := range titleToRecord {
      if nonEmpty[title] {
         continue
      }
      remaining--
      log.Printf("downloading (%d remaining)", remaining)
      var err error
      p, perr := platformOf(r.I)
      if perr != nil {
         log.Printf("error downloading %s: %v", title, perr)
         continue
      }
      switch p {
      case platformBandcamp:
         err = downloadBandcamp(&r, title, outputDir)
      case platformSoundCloud:
         err = downloadSoundCloud(&r, title, outputDir)
      case platformYouTube:
         err = downloadYouTube(&r, title, cfg.VisitorID, outputDir, threads, maxETA)
      }
      if err != nil {
         if errors.Is(err, errVisitorExpired) {
            log.Printf("visitor ID expired, clearing from config: %v", err)
            cfg.VisitorID = ""
            saveConfig(configPath, &cfg)
            return nil
         }
         log.Printf("error downloading %s: %v", title, err)
      }
   }

   if err := generateM3U(outputDir, records); err != nil {
      log.Printf("error generating M3U file: %v", err)
   }
   return nil
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

// Config is persisted to os.UserConfigDir()/umber/umber.json.
type Config struct {
   VisitorID string `json:"visitor_id"`
}

// Record represents one entry in the input JSON. I is the item URL, T the
// required title, R the author, and A the optional artwork URL.
type Record struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

// author returns the record's R with a YouTube topic-channel " - Topic"
// suffix stripped — the artist the auto-generated channel stands for.
// It is the single source of the strip: baseName's filename stem and
// remuxTagged's artist metadata both draw from it, so a topic-channel
// record is named and tagged with the same artist. The URL prefix check
// matches baseName's exactly; non-YouTube authors, even ones ending in
// " - Topic", pass through untouched.
func (r *Record) author() string {
   if strings.HasPrefix(r.I, "https://youtube.com/") && strings.HasSuffix(r.R, " - Topic") {
      return strings.TrimSuffix(r.R, " - Topic")
   }
   return r.R
}

// baseName returns the filename stem for the record, translating the
// adder's label() function: a YouTube "... - Topic" author (an auto-
// generated topic channel) has that suffix stripped via author, and the
// title alone is used when it already contains the author
// (case-insensitive substring match); otherwise the stem is
// "author - title".
func (r *Record) baseName() string {
   author := r.author()
   if strings.Contains(strings.ToLower(r.T), strings.ToLower(author)) {
      return r.T
   }
   return author + " - " + r.T
}

// main.go marker preserve
