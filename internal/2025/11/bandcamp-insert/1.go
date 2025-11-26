package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "net/http"
   "net/url"
   "os"
   "slices"
   "strconv"
   "time"
)

func (f *flag_set) do() error {
   var params report_params
   err := params.fetch(f.address)
   if err != nil {
      return err
   }
   tralbum_var, ok := params.tralbum()
   if !ok {
      return errors.New("tralbum")
   }
   detail, err := tralbum_var.tralbum()
   if err != nil {
      return err
   }
   var song_var song
   song_var.S = detail.TralbumArtist + " - " + detail.Title
   song_var.Q = url.Values{
      "a": {strconv.FormatInt(time.Now().Unix(), 36)},
      "b": {strconv.Itoa(params.Iid)},
      "c": {strconv.FormatInt(detail.ArtId, 10)},
      "p": {"b"},
      "y": {
         strconv.Itoa(detail.Time().Year()),
      },
   }.Encode()
   songs, err := read_songs(f.file)
   if err != nil {
      return err
   }
   songs = slices.Insert(songs, 0, song_var)
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", " ")
   err = enc.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(f.file, buf.Bytes())
}

func read_songs(name string) ([]song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

func (t *tralbum_details) Time() time.Time {
   return time.Unix(t.ReleaseDate, 0)
}

func (r *report_params) tralbum() (*tralbum, bool) {
   switch r.Itype {
   case "a":
      return &tralbum{r.Iid, 'a'}, true
   case "t":
      return &tralbum{r.Iid, 't'}, true
   }
   return nil, false
}

func (t *tralbum) tralbum() (*tralbum_details, error) {
   req, _ := http.NewRequest("", "http://bandcamp.com", nil)
   req.URL.Path = "/api/mobile/24/tralbum_details"
   req.URL.RawQuery = url.Values{
      "band_id":      {"1"},
      "tralbum_id":   {strconv.Itoa(t.Id)},
      "tralbum_type": {string(t.Type)},
   }.Encode()
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   details := &tralbum_details{}
   if err := json.NewDecoder(resp.Body).Decode(details); err != nil {
      return nil, err
   }
   return details, nil
}
