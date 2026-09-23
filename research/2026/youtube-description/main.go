// main.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "log"
   "net/url"
   "os"
)

func main() {
   log.SetFlags(log.Ltime)
   input := flag.String("i", "", "input JSON file (required)")
   output := flag.String("o", "descriptions.json", "output JSON file")
   flag.Parse()
   if *input == "" {
      flag.Usage()
      os.Exit(1)
   }

   data, err := os.ReadFile(*input)
   if err != nil {
      log.Fatal(err)
   }
   var songs []inputSong
   if err := json.Unmarshal(data, &songs); err != nil {
      log.Fatal(err)
   }

   progressPath := *output + ".progress"

   // ── Resume: load progress file ────────────────────────────────

   var entries []doneEntry
   if pdata, err := os.ReadFile(progressPath); err == nil {
      if err := json.Unmarshal(pdata, &entries); err != nil {
         log.Fatalf("cannot parse progress file %s: %v", progressPath, err)
      }
   }
   done := make(map[string]bool, len(entries))
   for _, e := range entries {
      done[e.URL] = true
   }
   already := len(done)
   if already > 0 {
      log.Printf("resuming: %d item(s) already fetched", already)
   }

   // Collect the YouTube items that still need fetching.
   type job struct{ url, id string }
   var jobs []job
   for _, song := range songs {
      u, err := url.Parse(song.I)
      if err != nil {
         log.Printf("skip %q: %v", song.I, err)
         continue
      }
      if u.Host != "youtube.com" {
         continue // not a YouTube item
      }
      id := u.Query().Get("v")
      if id == "" {
         log.Printf("skip %q: no 'v' parameter", song.I)
         continue
      }
      if done[song.I] {
         continue // already in the progress file
      }
      jobs = append(jobs, job{song.I, id})
   }

   if len(jobs) == 0 {
      if err := saveAll(progressPath, *output, entries); err != nil {
         log.Fatal(err)
      }
      log.Println("nothing to do")
      return
   }

   visitorID, err := fetchVisitorID()
   if err != nil {
      log.Fatalf("cannot fetch visitor ID: %v", err)
   }

   // ── Fetch loop ────────────────────────────────────────────────

   rateLimited := false
   for _, j := range jobs {
      play, err := fetch_player(j.id, visitorID)
      if err != nil {
         if errors.Is(err, errRateLimited) || errors.Is(err, errVisitorExpired) {
            rateLimited = true
            log.Printf("stopping at %q: %v", j.url, err)
            break
         }
         log.Printf("skip %q: %v", j.url, err)
         continue
      }

      entries = append(entries, doneEntry{
         URL: j.url,
         Info: VideoInfo{
            Author:           play.VideoDetails.Author,
            Title:            play.VideoDetails.Title,
            ShortDescription: play.VideoDetails.ShortDescription,
         },
      })
      log.Printf("%d/%d %s — %s", len(entries)-already, len(jobs),
         play.VideoDetails.Author, play.VideoDetails.Title)

      // Save after every item so a rate limit or crash never loses work.
      if err := saveAll(progressPath, *output, entries); err != nil {
         log.Fatal(err)
      }
   }

   if err := saveAll(progressPath, *output, entries); err != nil {
      log.Fatal(err)
   }
   if rateLimited {
      log.Fatalf("rate limited: %d of %d fetched, saved to %s — rerun the same command to resume",
         len(entries)-already, len(jobs), progressPath)
   }
   log.Printf("done: %d item(s) written to %s", len(entries), *output)
}

// saveAll writes the progress file and rebuilds the output file from it.
func saveAll(progressPath, output string, entries []doneEntry) error {
   pdata, err := json.MarshalIndent(entries, "", " ")
   if err != nil {
      return err
   }
   if err := os.WriteFile(progressPath, append(pdata, '\n'), 0644); err != nil {
      return err
   }

   infos := make([]VideoInfo, len(entries))
   for i, e := range entries {
      infos[i] = e.Info
   }
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   if err := enc.Encode(infos); err != nil {
      return err
   }
   return os.WriteFile(output, buf.Bytes(), 0644)
}

type VideoInfo struct {
   Author           string `json:"Author"`
   Title            string `json:"Title"`
   ShortDescription string `json:"ShortDescription"`
}

// doneEntry is one completed item in the progress file. The URL is the
// resume key; Info rides along so the output file can always be rebuilt
// from the progress file alone.
type doneEntry struct {
   URL  string    `json:"url"`
   Info VideoInfo `json:"info"`
}

type inputSong struct {
   I string `json:"I"`
}

// main.go marker preserve
