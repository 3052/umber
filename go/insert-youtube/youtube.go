package youtube

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "net/http"
   "net/url"
   "os"
   "path"
   "sort"
   "strconv"
   "strings"
   "time"
)

func (f *flags) do_youtube() error {
   now := strconv.FormatInt(time.Now().Unix(), 36)
   values := url.Values{}
   values.Set("a", now)
   values.Set("p", "y")
   values.Set("b", f.video_id)
   image, err := get_image(f.video_id)
   if err != nil {
      return err
   }
   if image != "" {
      values.Set("c", image)
   }
   ///
   play, err := y.tube.Player()
   if err != nil {
      return nil, err
   }
   var song1 song
   song1.S = play.VideoDetails.Author + " - " + play.VideoDetails.Title
   fmt.Println(play.VideoDetails.ShortDescription)
   values.Set("y", strconv.Itoa(
      play.Microformat.PlayerMicroformatRenderer.PublishDate[0].Year(),
   ))
   song1.Q = values.Encode()
   songs, err := read_songs("umber.json")
   if err != nil {
      panic(err)
   }
   songs = append([]*song{song1}, songs...)
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err := enc.Encode(songs)
   if err != nil {
      panic(err)
   }
   return write_file("umber.json", buf.Bytes())
}
