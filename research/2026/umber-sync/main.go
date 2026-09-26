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

// countStems counts, for every supported record, how many records sanitize
// to the same filename stem. Stems are compared lowercased (so names
// differing only in case collide, as they do on case-insensitive
// filesystems) and keyed with the expected extension (so a Bandcamp or
// SoundCloud .mp3 and a YouTube .opus sharing a title are not duplicates
// of each other). A record whose URL does not parse is an error: it would
// otherwise silently vanish from the stem map.
func countStems(records []Record, outputDir string) (map[string]int, error) {
   counts := make(map[string]int)
   for _, r := range records {
      if r.I == "" || r.T == "" {
         continue
      }
      p, err := platformOf(r.I)
      if err != nil {
         return nil, fmt.Errorf("record %q: %w", r.T, err)
      }
      ext, ok := stemExt(p)
      if !ok {
         continue
      }
      stem := sanitizeFilename(r.baseName(), ext, outputDir)
      counts[strings.ToLower(stem)+ext]++
   }
   return counts, nil
}

// fileStem returns the filename stem for r: the sanitized base name, plus
// " " + recordID when stemCounts shows another record sharing that stem
// case-insensitively. The suffix lets distinct items whose names differ
// only in case coexist on case-insensitive filesystems; non-duplicates keep
// their exact names. Identical records (same ID) collapse to one stem. A
// return of "" with a nil error means the record's platform is unsupported
// and no file is expected for it.
func fileStem(r *Record, stemCounts map[string]int, outputDir string) (string, error) {
   p, err := platformOf(r.I)
   if err != nil {
      return "", fmt.Errorf("record %q: %w", r.T, err)
   }
   ext, ok := stemExt(p)
   if !ok {
      return "", nil
   }
   stem := sanitizeFilename(r.baseName(), ext, outputDir)
   if stemCounts[strings.ToLower(stem)+ext] < 2 {
      return stem, nil
   }
   id, err := recordID(p, r.I)
   if err != nil {
      return "", fmt.Errorf("record %q: %w", r.T, err)
   }
   if id != "" {
      stem = sanitizeFilename(stem+" "+id, ext, outputDir)
   }
   return stem, nil
}

func generateM3U(outputDir string, records []Record) error {
   stemCounts, err := countStems(records, outputDir)
   if err != nil {
      return err
   }

   var items []*Record
   for i, r := range records {
      if r.I == "" || r.T == "" {
         continue
      }
      p, perr := platformOf(r.I)
      if perr != nil {
         return fmt.Errorf("record %q: %w", r.T, perr)
      }
      switch p {
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
      if isTempFile(name) || strings.HasSuffix(name, ".m3u") {
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

   if _, werr := fmt.Fprintln(out, "#EXTM3U"); werr != nil {
      return fmt.Errorf("write m3u header: %w", werr)
   }

   trackNum := 0
   for _, item := range items {
      stem, serr := fileStem(item, stemCounts, outputDir)
      if serr != nil {
         return serr
      }
      if stem == "" {
         continue
      }
      filename, exists := titleToFile[stem]
      if !exists {
         continue
      }
      trackNum++
      if _, werr := fmt.Fprintf(out, "#EXTINF:0,%s\n", item.baseName()); werr != nil {
         return fmt.Errorf("write m3u entry: %w", werr)
      }
      if _, werr := fmt.Fprintf(out, "%s\n", filename); werr != nil {
         return fmt.Errorf("write m3u entry: %w", werr)
      }
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

   if err := run(*inputFile, *outputDir, *threads, *maxETA); err != nil {
      log.Fatal(err)
   }
}

// run performs the full sync after flag validation: prepares the output
// directory, loads config and input records, removes files no longer in
// the input, downloads missing or empty items, and writes the M3U
// playlist. Conditions that abort the run are returned as errors for
// main to report; per-item failures (file removals, stats, downloads)
// are logged and skipped so one bad item does not abort the run. The
// visitor-expired return ends the run early so the next invocation
// refetches the visitor ID.
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

   // A record whose URL does not parse is fatal: it would silently drop
   // out of titleToRecord below and the sweep would then delete its
   // existing file as unreferenced.
   stemCounts, err := countStems(records, outputDir)
   if err != nil {
      return err
   }

   titleToRecord := make(map[string]Record)
   for _, r := range records {
      if r.I == "" || r.T == "" {
         continue
      }
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

   for title, r := range titleToRecord {
      if nonEmpty[title] {
         continue
      }
      var err error
      p, perr := platformOf(r.I)
      if perr != nil {
         log.Printf("error downloading %s: %v", title, perr)
         continue
      }
      switch p {
      case platformBandcamp:
         err = downloadBandcamp(r.I, title, outputDir, maxETA)
      case platformSoundCloud:
         err = downloadSoundCloud(r.I, title, outputDir, maxETA)
      case platformYouTube:
         err = downloadYouTube(r.I, title, cfg.VisitorID, outputDir, threads, maxETA)
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

// stemExt returns the extension a record's output file is expected to use,
// which sizes the filename truncation cap. ok is false for unsupported
// platforms.
func stemExt(p platform) (ext string, ok bool) {
   switch p {
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
// required title, R the author, and A the optional artwork URL.
type Record struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

// baseName returns the filename stem for the record, translating the
// adder's label() function: a YouTube "... - Topic" author (an auto-
// generated topic channel) has that suffix stripped, and the title alone
// is used when it already contains the author (case-insensitive substring
// match); otherwise the stem is "author - title".
func (r *Record) baseName() string {
   author := r.R
   if strings.HasPrefix(r.I, "https://youtube.com/") && strings.HasSuffix(author, " - Topic") {
      author = strings.TrimSuffix(author, " - Topic")
   }
   if strings.Contains(strings.ToLower(r.T), strings.ToLower(author)) {
      return r.T
   }
   return author + " - " + r.T
}

// main.go marker preserve
