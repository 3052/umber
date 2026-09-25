// bandcamp.go marker preserve
package main

import (
   "cmp"
   "encoding/json"
   "errors"
   "fmt"
   "html"
   "io"
   "net/http"
   "net/url"
   "slices"
   "strconv"
   "strings"
   "time"
)

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
   if detail.BandcampURL == "" {
      return errors.New("tralbum_details: bandcamp_url is empty")
   }

   songs, err := read_songs(name)
   if err != nil {
      return err
   }

   if slices.ContainsFunc(songs, func(song *Song) bool { return song.I == detail.BandcampURL }) {
      return fmt.Errorf("duplicate found: '%s' already exists in %s", detail.BandcampURL, name)
   }

   song_data := Song{
      A: "https://f4.bcbits.com/img/a" + strconv.Itoa(detail.ArtId) + "_2",
      D: time.Now().Unix(),
      I: detail.BandcampURL,
      R: detail.TralbumArtist,
      T: detail.Title,
      Y: detail.Time().Year(),
   }

   songs = append(songs, &song_data)
   slices.SortFunc(songs, func(a, b *Song) int {
      return cmp.Compare(b.D, a.D)
   })

   return write_songs(name, songs)
}

type ReportParams struct {
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

func (r *ReportParams) New(address string) error {
   resp, err := http.Get(address)
   if err != nil {
      return err
   }
   defer resp.Body.Close()

   var b strings.Builder
   if _, err := io.Copy(&b, resp.Body); err != nil {
      return err
   }
   data := b.String()

   // Locate the report-account-vm <p> tag.
   _, data, found := strings.Cut(data, `<p id="report-account-vm"`)
   if !found {
      return errors.New("report-account-vm not found")
   }

   // Locate the data-tou-report-params attribute inside that tag.
   _, data, found = strings.Cut(data, `data-tou-report-params="`)
   if !found {
      return errors.New("data-tou-report-params not found")
   }

   // The attribute value ends at the next double quote. The value is
   // HTML-entity-encoded JSON (e.g. &quot; for "), so unescape it.
   value, _, found := strings.Cut(data, `"`)
   if !found {
      return errors.New("data-tou-report-params: closing quote not found")
   }

   return json.Unmarshal([]byte(html.UnescapeString(value)), r)
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
   ArtId         int    `json:"art_id"`
   BandcampURL   string `json:"bandcamp_url"`
   ReleaseDate   int64  `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

func (t *TralbumDetails) Time() time.Time {
   return time.Unix(t.ReleaseDate, 0)
}

// bandcamp.go marker preserve
