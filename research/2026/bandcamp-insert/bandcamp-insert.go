package main

import (
   "bytes"
   "encoding/json"
   "encoding/xml"
   "errors"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "slices"
   "strconv"
   "time"
)

func cut_before(s, sep []byte) ([]byte, []byte, bool) {
   i := bytes.Index(s, sep)
   if i >= 0 {
      return s[:i], s[i:], true
   }
   return s, nil, false
}

func do_bandcamp(address, name string) error {
   var params ReportParams
   err := params.New(address)
   if err != nil {
      return err
   }
   tralbum, ok := params.Tralbum()
   if !ok {
      return errors.New("Tralbum")
   }
   detail, err := tralbum.Tralbum()
   if err != nil {
      return err
   }

   tralbum_id := strconv.Itoa(tralbum.Id)

   raw_songs, err := read_songs(name)
   if err != nil {
      return err
   }

   seen := make(map[string]bool)
   var songs []Song
   input_exists := false

   for _, song := range raw_songs {
      if song.I == tralbum_id {
         input_exists = true
      }
      if !seen[song.I] {
         seen[song.I] = true
         songs = append(songs, song)
      }
   }

   if input_exists {
      if len(songs) < len(raw_songs) {
         log.Printf("Cleaned up %d pre-existing duplicate(s) in %s\n", len(raw_songs)-len(songs), name)
         _ = write_songs(name, songs)
      }
      return fmt.Errorf("duplicate found: tralbum ID '%s' already exists in %s", tralbum_id, name)
   }

   song_data := Song{
      A: strconv.FormatInt(detail.ArtId, 10),
      D: time.Now().Unix(),
      I: tralbum_id,
      P: "bandcamp",
      T: detail.TralbumArtist + " - " + detail.Title,
      Y: detail.Time().Year(),
   }

   songs = slices.Insert(songs, 0, song_data)

   return write_songs(name, songs)
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "umber.json", "name")
   address := flag.String("a", "", "address")
   flag.Parse()

   if *address != "" {
      err := do_bandcamp(*address, *name)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

func write_songs(name string, songs []Song) error {
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

type ReportParams struct {
   Aid   int64  `json:"a_id"`
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

func (r *ReportParams) New(url2 string) error {
   resp, err := http.Get(url2)
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

func (r *ReportParams) Tralbum() (*Tralbum, bool) {
   switch r.Itype {
   case "a":
      return &Tralbum{r.Iid, 'a'}, true
   case "t":
      return &Tralbum{r.Iid, 't'}, true
   }
   return nil, false
}

type Song struct {
   A string `json:"A,omitempty"`
   D int64  `json:"D"`
   I string `json:"I"`
   P string `json:"P,omitempty"`
   T string `json:"T"`
   Y int    `json:"Y"`
}

func read_songs(name string) ([]Song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []Song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

type Tralbum struct {
   Id   int
   Type byte
}

func (t *Tralbum) Tralbum() (*TralbumDetails, error) {
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

type TralbumDetails struct {
   ArtId         int64 `json:"art_id"`
   ReleaseDate   int64 `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

func (t *TralbumDetails) Time() time.Time {
   return time.Unix(t.ReleaseDate, 0)
}

// bandcamp-insert.go
