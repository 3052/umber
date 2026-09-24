// main.go marker preserve
package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
   "time"
)

func do_check(name string, start int) error {
   file, err := os.Open(name)
   if err != nil {
      return err
   }
   defer file.Close()

   var songs []song
   if err := json.NewDecoder(file).Decode(&songs); err != nil {
      return err
   }

   type entry struct {
      index  int
      id     string
      artist string
      title  string
   }

   // Parse each URL once, up front. Non-youtube.com links are
   // skipped; anything malformed is an error, not a silent skip.
   var entries []entry
   for i, s := range songs {
      if i < start {
         continue
      }
      id, err := youtube_id(s.I)
      if err != nil {
         return fmt.Errorf("song %d: %w", i, err)
      }
      if id == "" {
         continue // not on youtube.com
      }
      entries = append(entries, entry{index: i, id: id, artist: s.R, title: s.T})
   }
   if len(entries) == 0 {
      return nil
   }

   visitorID, err := fetchVisitorID()
   if err != nil {
      return err
   }

   remaining := len(entries)
   for _, e := range entries {
      play, err := fetch_player(e.id, visitorID)
      if err != nil {
         return fmt.Errorf("%s: %w", e.id, err)
      }

      fmt.Println(e.index, remaining, e.id, e.artist, "-", e.title)
      remaining--

      if play.PlayabilityStatus.Status != "OK" {
         fmt.Printf("%+v\n", play.PlayabilityStatus)
         break
      }
      time.Sleep(99 * time.Millisecond)
   }
   return nil
}

func main() {
   name := flag.String("n", "umber.json", "name")
   start := flag.Int("s", -1, "start")
   flag.Parse()
   if *start >= 0 {
      err := do_check(*name, *start)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

// song is one entry of the umber JSON file.
type song struct {
   A string `json:"A,omitempty"` // image URL
   D int64  `json:"D"`
   I string `json:"I"` // source URL
   R string `json:"R"` // artist
   T string `json:"T"` // title
   Y int    `json:"Y"`
}

// main.go marker preserve
