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

// astralRune reports whether r is outside the Basic Multilingual Plane.
// HiBy players fail to open files whose names contain such runes (emoji,
// "fancy text" letters), reporting "playback failed file not found".
// BMP-only names are never rewritten.
func astralRune(r rune) bool {
   return r > 0xFFFF
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
// (255 chars) and the Windows MAX_PATH limit (259 usable chars). The
// extension and output directory determine the per-file cap. BMP-only
// names pass through unchanged; only names containing astral runes are
// rewritten first.
func sanitizeFilename(s string, ext string, outputDir string) string {
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

   capComponent := 255 - len(ext)
   capPath := 259 - len(outputDir) - 1 - len(ext)
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

// naming.go marker preserve
