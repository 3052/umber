// naming.go marker preserve
package main

import (
   "cmp"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "slices"
   "strings"
   "unicode/utf8"
)

const m3uFileName = "!playlist.m3u"

// maxExtLen is the longest extension this program produces (".opus").
// sanitizeFilename reserves it for every caller, so stems truncate
// identically no matter which platform's file they end up naming —
// callers never need to know the output format.
const maxExtLen = len(".opus")

// astralRune reports whether r is outside the Basic Multilingual Plane.
// HiBy players fail to open files whose names contain such runes (emoji,
// "fancy text" letters), reporting "playback failed file not found".
// BMP-only names are never rewritten.
func astralRune(r rune) bool {
   return r > 0xFFFF
}

// countStems counts, for every record, how many records sanitize to the
// same filename stem. Stems are compared lowercased, so names differing
// only in case collide, as they do on case-insensitive filesystems.
// Extensions play no part — sanitizeFilename reserves a fixed maximum
// extension length, so a Bandcamp or SoundCloud .mp3 and a YouTube .opus
// sharing a stem count as duplicates of each other. Records on
// unsupported platforms are counted too: that can only inflate a count
// and hand a record a harmless ID suffix, never mask a collision.
func countStems(records []Record, outputDir string) map[string]int {
   counts := make(map[string]int)
   for _, r := range records {
      counts[strings.ToLower(sanitizeFilename(r.baseName(), outputDir))]++
   }
   return counts
}

// fileStem returns the filename stem for r: the sanitized base name, plus
// " " + recordID when stemCounts shows another record sharing that stem
// case-insensitively — the count ignores extensions, so a cross-platform
// pair with the same stem disambiguates too. The suffix lets distinct
// items whose names differ only in case coexist on case-insensitive
// filesystems; non-duplicates keep their exact names. Identical records
// (same ID) collapse to one stem. A return of "" with a nil error means
// the record's platform is unsupported and no file is expected for it.
// A record whose URL does not parse is an error: it would otherwise
// silently vanish from the callers' stem maps.
func fileStem(r *Record, stemCounts map[string]int, outputDir string) (string, error) {
   p, err := platformOf(r.I)
   if err != nil {
      return "", fmt.Errorf("record %q: %w", r.T, err)
   }
   switch p {
   case platformBandcamp, platformYouTube, platformSoundCloud:
   default:
      return "", nil
   }
   stem := sanitizeFilename(r.baseName(), outputDir)
   if stemCounts[strings.ToLower(stem)] < 2 {
      return stem, nil
   }
   id, err := recordID(p, r.I)
   if err != nil {
      return "", fmt.Errorf("record %q: %w", r.T, err)
   }
   if id != "" {
      stem = sanitizeFilename(stem+" "+id, outputDir)
   }
   return stem, nil
}

// fixAstralRunes rewrites s (which must contain astral runes) into a
// HiBy-safe string: mathematical letters/digits become ASCII, every other
// astral rune (emoji, flags, ...) is dropped, along with the zero-width
// emoji joiners they leave dangling. All other runes pass through as-is.
func fixAstralRunes(s string) string {
   var b strings.Builder
   b.Grow(len(s))
   for _, r := range s {
      switch {
      case r <= 0xFFFF:
         if r == 0x200D || r == 0xFE0F {
            continue
         }
         b.WriteRune(r)
      default:
         if a, ok := mathAlnumASCII(r); ok {
            b.WriteRune(a)
         }
      }
   }
   return b.String()
}

// generateM3U writes the playlist: one file name per line, ordered by D
// descending. No #EXTINF entries are written — the display info they
// carry (author - title) is already in the file name and the tag
// metadata, so every player reads it from the file itself. The
// #EXTM3U header stays: it is the standard marker identifying the
// file as a playlist. Only records on supported platforms with an
// existing non-empty-name output file are listed.
func generateM3U(outputDir string, records []Record) error {
   stemCounts := countStems(records, outputDir)

   var items []*Record
   for i, r := range records {
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
      if _, werr := fmt.Fprintf(out, "%s\n", filename); werr != nil {
         return fmt.Errorf("write m3u entry: %w", werr)
      }
   }

   log.Printf("M3U file generated: %s (%d tracks)", m3uPath, trackNum)
   return nil
}

// isTempFile reports whether name is one of this program's in-flight
// temp files: <stem>.tmp (raw YouTube stream), <stem>.t (Bandcamp /
// SoundCloud stream), or <stem>.remux.<ext> (FFmpeg remux output). The
// remux temp ends in the real extension so FFmpeg picks the muxer from
// the name; its marker ends in a dot, and since sanitizeFilename
// strips trailing dots from stems, a final file can never land on a
// temp name.
func isTempFile(name string) bool {
   if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".t") {
      return true
   }
   if i := strings.LastIndex(name, "."); i >= 0 {
      return strings.HasSuffix(name[:i], ".remux.")
   }
   return false
}

// mathAlnumASCII maps Mathematical Alphanumeric Symbols (U+1D400–U+1D7FF,
// the 𝗯𝗼𝗹𝗱 / 𝘀𝗰𝗿𝗶𝗽𝘁 / 𝟭𝟮𝟯 code points emitted by fancy-text generators)
// to their ASCII equivalents, matching their Unicode NFKC decompositions.
// ok is false for runes with no ASCII equivalent.
func mathAlnumASCII(r rune) (ascii rune, ok bool) {
   if r >= 0x1D400 && r <= 0x1D6A3 { // Latin letter styles, all 26+26 runs
      off := (r - 0x1D400) % 52
      if off < 26 {
         return 'A' + off, true
      }
      return 'a' + off - 26, true
   }
   if r >= 0x1D7CE && r <= 0x1D7FF { // digit styles
      return '0' + (r-0x1D7CE)%10, true
   }
   return 0, false
}

// sanitizeFilename sanitizes a title for use as a filename, then truncates
// the result so that name+ext fits within both the NTFS component limit
// (255 chars) and the Windows MAX_PATH limit (259 usable chars), with
// maxExtLen reserved for the extension. The output directory determines
// the per-file path cap. BMP-only names pass through unchanged; only
// names containing astral runes are rewritten first.
func sanitizeFilename(s string, outputDir string) string {
   if strings.ContainsFunc(s, astralRune) {
      s = fixAstralRunes(s)
   }
   invalid := `\/:*?"<>|`
   var b strings.Builder
   for _, c := range s {
      if strings.ContainsRune(invalid, c) {
         b.WriteByte('_')
      } else {
         b.WriteRune(c)
      }
   }
   result := strings.TrimRight(b.String(), ". ")

   capComponent := 255 - maxExtLen
   capPath := 259 - len(outputDir) - 1 - maxExtLen
   cap := capComponent
   if capPath < cap {
      cap = capPath
   }
   if cap < 1 {
      cap = 1
   }

   if len(result) > cap {
      result = result[:cap]
      for !utf8.ValidString(result) {
         result = result[:len(result)-1]
      }
      result = strings.TrimRight(result, ". ")
   }
   return result
}

// naming.go marker preserve
