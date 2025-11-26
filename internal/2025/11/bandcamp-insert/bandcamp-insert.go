package main

import (
   "bytes"
   "encoding/json"
   "encoding/xml"
   "errors"
   "flag"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "slices"
   "strconv"
   "time"
)

type flag_set struct {
   address string
   file    string
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

type tralbum struct {
   Id   int
   Type byte
}

///

func (f *flag_set) do() error {
   var params ReportParams
   err := params.New(f.address)
   if err != nil {
      return err
   }
   tralbum, ok := params.Tralbum()
   if !ok {
      return errors.New("tralbum")
   }
   detail, err := tralbum.Tralbum()
   if err != nil {
      return err
   }
   var songVar song
   songVar.S = detail.TralbumArtist + " - " + detail.Title
   songVar.Q = url.Values{
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
   songs = slices.Insert(songs, 0, songVar)
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

func (t *TralbumDetails) Time() time.Time {
   return time.Unix(t.ReleaseDate, 0)
}

func cut_before(s, sep []byte) ([]byte, []byte, bool) {
   i := bytes.Index(s, sep)
   if i >= 0 {
      return s[:i], s[i:], true
   }
   return s, nil, false
}

type TralbumDetails struct {
   ArtId         int64 `json:"art_id"`
   ReleaseDate   int64 `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

type song struct {
   Q string
   S string
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

type ReportParams struct {
   Aid   int64  `json:"a_id"`
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

func (r *ReportParams) New(urlVar string) error {
   resp, err := http.Get(urlVar)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return err
   }
   _, data, _ = cut_before(data, []byte(`<p id="report-account-vm"`))
   var p struct {
      DataTouReportParams []byte `xml:"data-tou-report-params,attr"`
   }
   err = xml.Unmarshal(data, &p)
   if err != nil {
      return err
   }
   return json.Unmarshal(p.DataTouReportParams, r)
}

func (r *ReportParams) Tralbum() (*tralbum, bool) {
   switch r.Itype {
   case "a":
      return &tralbum{r.Iid, 'a'}, true
   case "t":
      return &tralbum{r.Iid, 't'}, true
   }
   return nil, false
}

func (t *tralbum) tralbum() (*TralbumDetails, error) {
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
   detail := &TralbumDetails{}
   if err := json.NewDecoder(resp.Body).Decode(detail); err != nil {
      return nil, err
   }
   return detail, nil
}
