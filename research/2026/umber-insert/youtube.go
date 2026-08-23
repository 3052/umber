package main

import (
   "fmt"
   "log"
   "slices"
   "time"
)

func do_video_id(video_id, name, visitorID string) error {
   raw_songs, err := read_songs(name)
   if err != nil {
      return err
   }
   seen := make(map[string]bool)
   var songs []Song
   input_exists := false

   // Iterate through ALL existing records to filter out duplicates
   for _, song := range raw_songs {
      if song.I == "" {
         // Safety fallback: if the record is missing the "I" string, keep it to prevent data loss
         songs = append(songs, song)
         continue
      }
      // Check if the input we are trying to add already exists
      if song.I == video_id {
         input_exists = true
      }
      // If we haven't seen this ID yet in the loop, keep it and mark it as seen
      if !seen[song.I] {
         seen[song.I] = true
         songs = append(songs, song)
      }
   }

   if input_exists {
      // If pre-existing duplicates were found and cleaned from the file, save the clean file before exiting.
      if len(songs) < len(raw_songs) {
         log.Printf("Cleaned up %d pre-existing duplicate(s) in %s\n", len(raw_songs)-len(songs), name)
         _ = write_songs(name, songs)
      }
      return fmt.Errorf("duplicate found: video ID '%s' already exists in %s", video_id, name)
   }

   play, err := fetch_player(video_id, visitorID)
   if err != nil {
      return err
   }
   fmt.Println(play.VideoDetails.ShortDescription)

   image, err := get_image(video_id)
   if err != nil {
      return err
   }

   // Insert native map data
   song_data := Song{
      D: time.Now().Unix(),
      I: video_id,
      T: play.VideoDetails.Author + " - " + play.VideoDetails.Title,
      Y: play.Microformat.PlayerMicroformatRenderer.PublishDate.Year(),
   }
   if image != "" {
      song_data.A = image
   }

   songs = slices.Insert(songs, 0, song_data)

   // Save the newly cleaned and updated list
   return write_songs(name, songs)
}

// youtube-insert.go
