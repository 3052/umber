// main.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "strings"
)

// do_fix sets the R (artist) and T (title) fields of every youtube.com song
// to the exact values reported by the innertube player API, and writes the
// result back to the same file. Because processed items cannot be told apart
// from unprocessed ones in the songs file itself (R may be missing or just
// truncated), progress is tracked in a separate resume file. If the run is
// interrupted (for example by rate limiting), running the same command again
// picks up where it left off. Each processed item prints one line:
// file index, remaining count, video ID, new artist, new title.
func do_fix(name string) error {
   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   resume := resume_path(name)
   done, err := read_resume(resume)
   if err != nil {
      return err
   }

   type job struct {
      index int
      song  *song
      id    string
   }

   // Parse each URL once, up front. Non-youtube.com links are
   // skipped; anything malformed is an error, not a silent skip.
   var jobs []job
   for i, s := range songs {
      if done[s.I] {
         continue
      }
      id, err := youtube_id(s.I)
      if err != nil {
         return fmt.Errorf("song %d: %w", i, err)
      }
      if id == "" {
         continue // not on youtube.com
      }
      jobs = append(jobs, job{index: i, song: s, id: id})
   }
   if len(jobs) == 0 {
      fmt.Println("nothing to fix:", name)
      return nil
   }

   // A fresh visitor ID is fetched each run, since a cached one may have
   // expired.
   visitorID, err := fetchVisitorID()
   if err != nil {
      return err
   }

   remaining := len(jobs)
   for _, j := range jobs {
      play, err := fetch_player(j.id, visitorID)
      if err != nil {
         if errors.Is(err, errRateLimited) || errors.Is(err, errVisitorExpired) {
            return fmt.Errorf(
               "song %d, %s: %w (%v remaining, run the same command again to resume)",
               j.index, j.id, err, remaining,
            )
         }
         return fmt.Errorf("song %d, %s: %w", j.index, j.id, err)
      }
      if play.PlayabilityStatus.Status != "OK" {
         return fmt.Errorf(
            "song %d, %s: %s — %s",
            j.index, j.id, play.PlayabilityStatus.Status, play.PlayabilityStatus.Reason,
         )
      }
      if play.VideoDetails.Title == "" {
         return fmt.Errorf("song %d, %s: player response has empty title", j.index, j.id)
      }
      if play.VideoDetails.Author == "" {
         return fmt.Errorf("song %d, %s: player response has empty author", j.index, j.id)
      }
      j.song.R = play.VideoDetails.Author
      j.song.T = play.VideoDetails.Title
      if err := write_songs(name, songs); err != nil {
         return err
      }
      done[j.song.I] = true
      if err := write_resume(resume, done); err != nil {
         return err
      }
      fmt.Println(j.index, remaining, j.id, play.VideoDetails.Author, "-", play.VideoDetails.Title)
      remaining--
   }
   return nil
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "", "input JSON file path (required)")
   flag.Parse()
   if *name == "" {
      flag.Usage()
      os.Exit(1)
   }
   if err := do_fix(*name); err != nil {
      log.Fatal(err)
   }
}

func read_resume(name string) (map[string]bool, error) {
   data, err := os.ReadFile(name)
   if errors.Is(err, os.ErrNotExist) {
      return map[string]bool{}, nil
   }
   if err != nil {
      return nil, err
   }
   var done map[string]bool
   if err := json.Unmarshal(data, &done); err != nil {
      return nil, fmt.Errorf("cannot parse %v (delete it to start over): %w", name, err)
   }
   if done == nil {
      done = map[string]bool{}
   }
   return done, nil
}

// resume_path returns the path of the file that tracks which songs have
// already been processed. It sits next to the songs file.
func resume_path(name string) string {
   base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
   return filepath.Join(filepath.Dir(name), base+".resume.json")
}

func write_file(name string, data []byte) error {
   // Write to a temp file first, then rename, so an interrupted run cannot
   // leave a half-written songs file behind.
   temp := name + ".tmp"
   if err := os.WriteFile(temp, data, os.ModePerm); err != nil {
      return err
   }
   return os.Rename(temp, name)
}

func write_resume(name string, done map[string]bool) error {
   // json.Marshal sorts map keys, so the output is stable across runs.
   data, err := json.MarshalIndent(done, "", " ")
   if err != nil {
      return err
   }
   data = append(data, '\n')
   return write_file(name, data)
}

// write_songs writes the songs in the same format as the main script.
func write_songs(name string, songs []*song) error {
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err := enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
}

type song struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   R string `json:"R,omitempty"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

func read_songs(name string) ([]*song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []*song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

// main.go marker preserve
