// main.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
   "strings"
)

// default_output derives the output path from the input path:
// lines.json -> lines.out.json.
func default_output(name string) string {
   return strings.TrimSuffix(name, ".json") + ".out.json"
}

// do_replace reads the input file and replaces the L lines of every
// item with R and T. Items on youtube.com get author and title from
// the player API; every other item uses L[0] as the author and L[1]
// as the title. The output file is written once, after every item has
// been converted; any error aborts the run and nothing is written.
func do_replace(name, output string) error {
   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   // First pass: convert the items that need no request. Malformed
   // links are an error, not a silent skip.
   type job struct {
      index int
      id    string
   }
   var jobs []job
   for i := range songs {
      s := &songs[i]
      id, err := youtube_id(s.I)
      if err != nil {
         return fmt.Errorf("song %d: %w", i, err)
      }
      if id == "" {
         // Not on youtube.com: author is line 0, title is line 1.
         if len(s.L) < 2 {
            return fmt.Errorf("song %d: need two lines in L, got %d", i, len(s.L))
         }
         author, title := s.L[0], s.L[1]
         s.L = nil
         s.R = author
         s.T = title
         continue
      }
      jobs = append(jobs, job{index: i, id: id})
   }

   if len(jobs) > 0 {
      visitorID, err := fetchVisitorID()
      if err != nil {
         return err
      }

      remaining := len(jobs)
      for _, j := range jobs {
         play, err := fetch_player(j.id, visitorID)
         if err != nil {
            return fmt.Errorf("song %d, %s: %w", j.index, j.id, err)
         }
         if play.PlayabilityStatus.Status != "OK" {
            return fmt.Errorf("song %d, %s: %s — %s", j.index, j.id,
               play.PlayabilityStatus.Status, play.PlayabilityStatus.Reason)
         }
         author := play.VideoDetails.Author
         title := play.VideoDetails.Title
         if author == "" || title == "" {
            return fmt.Errorf("song %d, %s: player response has empty author or title",
               j.index, j.id)
         }
         remaining--

         set_artist_title(&songs[j.index], author, title)
         fmt.Println(j.index, remaining, j.id, author, "-", title)
      }
   }

   return write_songs(output, songs)
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "", "input JSON file (required)")
   output := flag.String("o", "", "output JSON file (default: input with .out.json)")
   flag.Parse()
   if *name == "" {
      flag.Usage()
      return
   }
   out := *output
   if out == "" {
      out = default_output(*name)
   }
   if err := do_replace(*name, out); err != nil {
      log.Fatal(err)
   }
}

// set_artist_title replaces the L lines of a youtube.com song with R
// and T from the player response. The title is stored exactly as
// delivered; R is stored only when the title does not already name
// the author.
func set_artist_title(s *Song, author, title string) {
   s.L = nil
   s.R = ""
   s.T = title
   if !containsAuthor(title, author) {
      s.R = author
   }
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, 0644)
}

// write_songs writes the songs as indented JSON (one space per level,
// no HTML escaping), matching the umber file style.
func write_songs(name string, songs []Song) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   if err := enc.Encode(songs); err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
}

// Song is one entry of the JSON file. Input entries carry L, the raw
// lines this script replaces; output entries carry R and T instead. R
// is dropped when the title already contains the author.
type Song struct {
   A string   `json:"A,omitempty"` // image URL
   D int64    `json:"D"`
   I string   `json:"I"`           // source URL
   L []string `json:"L,omitempty"` // raw lines, replaced by R and T
   R string   `json:"R,omitempty"` // author (optional)
   T string   `json:"T,omitempty"` // title
   Y int      `json:"Y"`
}

func read_songs(name string) ([]Song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []Song
   if err := json.Unmarshal(data, &songs); err != nil {
      return nil, err
   }
   return songs, nil
}

// main.go marker preserve
