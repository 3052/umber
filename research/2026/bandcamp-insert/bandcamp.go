// bandcamp.go
package main

import (
   "bytes"
   "encoding/json"
   "encoding/xml"
   "errors"
   "flag"
   "io"
   "net/http"
   "net/url"
   "os"
   "strconv"
   "time"
)

type song struct {
   Q string
   S string
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

func main() {
   flag.Parse()
   songs, err := read_songs("umber.json")
   if err != nil {
      panic(err)
   }
   if len(os.Args) >= 3 {
      args := os.Args[2:]
      var songVar *song
      songVar, err = new_bandcamp().parse(args)
      if err != nil {
         panic(err)
      }
      songs = append([]*song{songVar}, songs...)
      var buf bytes.Buffer
      enc := json.NewEncoder(&buf)
      enc.SetEscapeHTML(false)
      enc.SetIndent("", " ")
      err := enc.Encode(songs)
      if err != nil {
         panic(err)
      }
      err = os.WriteFile("umber.json", buf.Bytes(), os.ModePerm)
      if err != nil {
         panic(err)
      }
   } else {
      new_bandcamp().f.Usage()
   }
}

type bandcamp_set struct {
   address string
   f       *flag.FlagSet
}

func new_bandcamp() *bandcamp_set {
   var set bandcamp_set
   set.f = flag.NewFlagSet("bandcamp", flag.ExitOnError)
   set.f.StringVar(&set.address, "a", "", "address")
   return &set
}

func (b *bandcamp_set) parse(args []string) (*song, error) {
   b.f.Parse(args)
   var params ReportParams
   err := params.New(b.address)
   if err != nil {
      return nil, err
   }
   tralbum, ok := params.Tralbum()
   if !ok {
      return nil, errors.New("Tralbum")
   }
   detail, err := tralbum.Tralbum()
   if err != nil {
      return nil, err
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
   return &songVar, nil
}
func cut_before(s, sep []byte) ([]byte, []byte, bool) {
   i := bytes.Index(s, sep)
   if i >= 0 {
      return s[:i], s[i:], true
   }
   return s, nil, false
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

type Tralbum struct {
   Id int
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

func (t *TralbumDetails) Time() time.Time {
   return time.Unix(t.ReleaseDate, 0)
}

type TralbumDetails struct {
   ArtId         int64 `json:"art_id"`
   ReleaseDate   int64  `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}
