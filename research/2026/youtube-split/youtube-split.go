// Parses a JSON file of YouTube videos ({Author, Title, ...}).
//
// Every item gets the same process, no special cases:
//
//   line = title; if the title does not contain the author
//   (case-insensitive, spaces ignored), line = author + ", " + title.
//   Split line at the space closest to its center:
//   artist = left half, title = right half.
//
// Usage:
//
//   go run main.go videos.json [results.txt]
package main

import (
   "bufio"
   "encoding/json"
   "fmt"
   "log"
   "math"
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

   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "usage: %s <videos.json> [results.txt]\n", os.Args[0])
      os.Exit(1)
   }
   inPath := os.Args[1]
   outPath := "results.txt"
   if len(os.Args) > 2 {
      outPath = os.Args[2]
   }

   raw, err := os.ReadFile(inPath)
   if err != nil {
      log.Fatalf("reading %s: %v", inPath, err)
   }

   var videos []Video
   if err := json.Unmarshal(raw, &videos); err != nil {
      log.Fatalf("parsing JSON: %v", err)
   }

   outFile, err := os.Create(outPath)
   if err != nil {
      log.Fatalf("creating %s: %v", outPath, err)
   }
   defer outFile.Close()

   out := bufio.NewWriter(outFile)

   var count, emptyCount int

   for _, v := range videos {
      // Rows with no data at all (removed videos etc.) - skip.
      if v.Author == "" && v.Title == "" {
         emptyCount++
         continue
      }

      line := v.Title
      if !containsAuthor(v.Title, v.Author) {
         line = v.Author + ", " + v.Title
      }

      var artist, title string
      if left, right, ok := splitAtCenter(line); ok {
         artist, title = left, right
      } else {
         // No usable split point: fall back to author/title as-is.
         artist, title = v.Author, v.Title
      }

      fmt.Fprintf(out, "artist: %s\ntitle: %s\n\n", artist, title)
      count++
   }

   if err := out.Flush(); err != nil {
      log.Fatalf("writing %s: %v", outPath, err)
   }

   fmt.Fprintf(os.Stderr, "%d entries -> %s, %d empty rows skipped\n",
      count, outPath, emptyCount)
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

type Video struct {
   Author string `json:"Author"`
   Title  string `json:"Title"`
}
