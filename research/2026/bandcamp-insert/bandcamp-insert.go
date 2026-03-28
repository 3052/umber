package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
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

func (c *client) do_address() error {
   params, err := fetch_report_params(c.address)
   if err != nil {
      return err
   }
   details, err := params.tralbum()
   if err != nil {
      return err
   }
   var song_data song
   song_data.S = fmt.Sprintf("%v - %v", details.TralbumArtist, details.Title)
   values := url.Values{}
   values.Set("a", strconv.FormatInt(time.Now().Unix(), 36))
   values.Set("b", strconv.Itoa(params.Iid))
   values.Set("c", strconv.Itoa(details.ArtId))
   values.Set("p", "b")
   values.Set("y", strconv.Itoa(
      time.Unix(details.ReleaseDate, 0).Year(),
   ))
   song_data.Q = values.Encode()
   data, err := os.ReadFile(c.file)
   if err != nil {
      return err
   }
   var songs []song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return err
   }
   songs = slices.Insert(songs, 0, song_data)
   var buf bytes.Buffer
   encode := json.NewEncoder(&buf)
   encode.SetEscapeHTML(false)
   encode.SetIndent("", " ")
   err = encode.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(c.file, buf.Bytes())
}

func (r *report_params) tralbum() (*tralbum_details, error) {
   req := http.Request{
      URL: &url.URL{
         Scheme: "http",
         Host: "bandcamp.com",
         Path: "/api/mobile/24/tralbum_details",
         RawQuery: url.Values{
            "band_id":      {"1"},
            "tralbum_id":   {strconv.Itoa(r.Iid)},
            "tralbum_type": {r.Itype},
         }.Encode(),
      },
   }
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   result := &tralbum_details{}
   err = json.NewDecoder(resp.Body).Decode(result)
   if err != nil {
      return nil, err
   }
   return result, nil
}

type report_params struct {
   Aid   int64  `json:"a_id"`
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

type song struct {
   Q string
   S string
}

type client struct {
   address string
   file    string
}

type tralbum_details struct {
   ArtId         int `json:"art_id"`
   ReleaseDate   int64 `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

func main() {
   err := new(client).do()
   if err != nil {
      log.Fatal(err)
   }
}

func (c *client) do() error {
   flag.StringVar(&c.address, "a", "", "address")
   flag.StringVar(&c.file, "f", "umber.json", "file")
   flag.Parse()
   if c.address != "" {
      return c.do_address()
   }
   flag.Usage()
   return nil
}

func fetch_report_params(url_data string) (*report_params, error) {
   resp, err := http.Get(url_data)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   var builder strings.Builder
   _, err = io.Copy(&builder, resp.Body)
   if err != nil {
      return nil, err
   }
   _, data, found := strings.Cut(builder.String(), `data-tou-report-params="`)
   if !found {
      return nil, errors.New("attribute not found")
   }
   data, _, found = strings.Cut(data, `"`)
   if !found {
      return nil, errors.New("closing quote not found")
   }
   data = html.UnescapeString(data)
   result := &report_params{}
   err = json.Unmarshal([]byte(data), result)
   if err != nil {
      return nil, err
   }
   return result, nil
}
