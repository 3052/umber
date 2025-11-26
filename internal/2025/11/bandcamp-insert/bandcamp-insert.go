package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "html"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "slices"
   "strconv"
   "strings"
   "time"
)

func (f *flag_set) do() error {
   var params report_params
   err := params.fetch(f.address)
   if err != nil {
      return err
   }
   details, err := params.tralbum()
   if err != nil {
      return err
   }
   var song_var song
   song_var.S = details.TralbumArtist + " - " + details.Title
   values := url.Values{}
   values.Set("a", strconv.FormatInt(time.Now().Unix(), 36))
   values.Set("b", strconv.Itoa(params.Iid))
   values.Set("c", strconv.Itoa(details.ArtId))
   values.Set("p", "b")
   values.Set("y", strconv.Itoa(
      time.Unix(details.ReleaseDate, 0).Year(),
   ))
   song_var.Q = values.Encode()
   data, err := os.ReadFile(f.file)
   if err != nil {
      return err
   }
   var songs []song
   err = json.Unmarshal(data, &songs)
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
type tralbum_details struct {
   ArtId         int `json:"art_id"`
   ReleaseDate   int64 `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

func (r *report_params) tralbum() (*tralbum_details, error) {
   req, _ := http.NewRequest("", "http://bandcamp.com", nil)
   req.URL.Path = "/api/mobile/24/tralbum_details"
   req.URL.RawQuery = url.Values{
      "band_id":      {"1"},
      "tralbum_id":   {strconv.Itoa(r.Iid)},
      "tralbum_type": {r.Itype},
   }.Encode()
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   details := &tralbum_details{}
   err = json.NewDecoder(resp.Body).Decode(details)
   if err != nil {
      return nil, err
   }
   return details, nil
}

type report_params struct {
   Aid   int64  `json:"a_id"`
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

func main() {
   var set flag_set
   flag.StringVar(&set.address, "a", "", "address")
   flag.StringVar(&set.file, "f", "umber.json", "file")
   flag.Parse()
   if set.address != "" {
      err := set.do()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

func (r *report_params) fetch(address string) error {
   resp, err := http.Get(address)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return err
   }
   _, value, found := strings.Cut(string(data), `data-tou-report-params="`)
   if !found {
      return errors.New("attribute not found")
   }
   value, _, found = strings.Cut(value, `"`)
   if !found {
      return errors.New("closing quote not found")
   }
   value = html.UnescapeString(value)
   return json.Unmarshal([]byte(value), r)
}

type song struct {
   Q string
   S string
}

type flag_set struct {
   address string
   file    string
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}
