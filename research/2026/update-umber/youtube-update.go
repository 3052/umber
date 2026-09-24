// Updates umber.json entries from descriptions.json, writing a new file.
//
// For each umber entry: if the host of "I" is youtube.com, find the entry in
// descriptions with the same URL (exact string match) - fatal error if not
// found, or if its Author or Title is empty. Then build artist/title the
// same way youtube-split.go does, and use them to update "R" and "T".
//
// Usage:
//
//   go run umber-update.go umber.json descriptions.json [umber-new.json]
package main

import (
   "encoding/json"
   "fmt"
   "log"
   "math"
   "net/url"
   "os"
   "strings"
)

// containsAuthor reports whether the title contains the author,
// ignoring case and spaces.
func containsAuthor(title, author string) bool {
   fold := func(s string) string {
      return strings.ToLower(strings.ReplaceAll(s, " ", ""))
   }
   return strings.Contains(fold(title), fold(author))
}

func main() {
   log.SetFlags(0)

   if len(os.Args) < 3 {
      fmt.Fprintf(os.Stderr, "usage: %s <umber.json> <descriptions.json> [umber-new.json]\n", os.Args[0])
      os.Exit(1)
   }
   umberPath, descPath := os.Args[1], os.Args[2]
   outPath := "umber-new.json"
   if len(os.Args) > 3 {
      outPath = os.Args[3]
   }

   umberRaw, err := os.ReadFile(umberPath)
   if err != nil {
      log.Fatalf("reading %s: %v", umberPath, err)
   }
   var umber []UmberEntry
   if err := json.Unmarshal(umberRaw, &umber); err != nil {
      log.Fatalf("parsing %s: %v", umberPath, err)
   }

   descRaw, err := os.ReadFile(descPath)
   if err != nil {
      log.Fatalf("reading %s: %v", descPath, err)
   }
   var descs []Description
   if err := json.Unmarshal(descRaw, &descs); err != nil {
      log.Fatalf("parsing %s: %v", descPath, err)
   }

   byURL := make(map[string]Description, len(descs))
   for _, d := range descs {
      byURL[d.URL] = d
   }

   updated := 0

   for i := range umber {
      e := &umber[i]

      u, err := url.Parse(e.I)
      if err != nil || u.Hostname() != "youtube.com" {
         continue
      }

      d, ok := byURL[e.I]
      if !ok {
         log.Fatalf("error: %s not found in %s", e.I, descPath)
      }
      if d.Author == "" || d.Title == "" {
         log.Fatalf("error: %s has an empty Author or Title", e.I)
      }

      line := d.Title
      if !containsAuthor(d.Title, d.Author) {
         line = d.Author + ", " + d.Title
      }

      var artist, title string
      if left, right, ok := splitAtCenter(line); ok {
         artist, title = left, right
      } else {
         // No usable split point: fall back to author/title as-is.
         artist, title = d.Author, d.Title
      }

      e.R = artist
      e.T = title
      updated++
   }

   out, err := json.MarshalIndent(umber, "", " ")
   if err != nil {
      log.Fatalf("encoding JSON: %v", err)
   }
   out = append(out, '\n')
   if err := os.WriteFile(outPath, out, 0o644); err != nil {
      log.Fatalf("writing %s: %v", outPath, err)
   }

   fmt.Fprintf(os.Stderr, "%d entries -> %s (%d updated)\n", len(umber), outPath, updated)
}

// splitAtCenter splits s on the space closest to the middle of the string and
// returns the trimmed halves. It works on runes, not bytes, so titles with
// non-ASCII characters split in the right place. Ties go to the leftmost
// space. ok is false when there is no usable split point.
func splitAtCenter(s string) (left, right string, ok bool) {
   runes := []rune(s)
   center := float64(len(runes)) / 2

   best, bestDist := -1, math.MaxFloat64
   for i, r := range runes {
      if r != ' ' {
         continue
      }
      if d := math.Abs(float64(i) - center); d < bestDist {
         best, bestDist = i, d
      }
   }
   if best < 0 {
      return "", "", false
   }

   left = strings.TrimSpace(string(runes[:best]))
   right = strings.TrimSpace(string(runes[best+1:]))
   if left == "" || right == "" {
      return "", "", false
   }
   return left, right, true
}

type Description struct {
   Author string `json:"Author"`
   Title  string `json:"Title"`
   URL    string `json:"URL"`
}

type UmberEntry struct {
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R"`
   T string `json:"T"`
   Y int    `json:"Y"`
}
