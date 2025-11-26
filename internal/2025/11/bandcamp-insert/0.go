package main

import (
   "encoding/json"
   "errors"
   "flag"
   "html"
   "io"
   "log"
   "net/http"
   "os"
   "strings"
)

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

type tralbum_details struct {
   ArtId         int64 `json:"art_id"`
   ReleaseDate   int64 `json:"release_date"`
   Title         string
   TralbumArtist string `json:"tralbum_artist"`
}

type report_params struct {
   Aid   int64  `json:"a_id"`
   Iid   int    `json:"i_id"`
   Itype string `json:"i_type"`
}

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
