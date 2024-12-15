package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "os"
)

type record struct {
   Q string
   S string
}

func main() {
   flag.Parse()
   recs, err := records("umber.json")
   if err != nil {
      panic(err)
   }
   if len(os.Args) >= 3 {
      arg := os.Args[2:]
      var rec *record
      switch os.Args[1] {
      case "http":
         rec, err = new_http().parse(arg)
      case "bandcamp":
         rec, err = new_bandcamp().parse(arg)
      case "soundcloud":
         rec, err = new_soundcloud().parse(arg)
      case "youtube":
         rec, err = new_youtube().parse(arg)
      }
      if err != nil {
         panic(err)
      }
      recs = append([]*record{rec}, recs...)
      var text bytes.Buffer
      enc := json.NewEncoder(&text)
      enc.SetEscapeHTML(false)
      enc.SetIndent("", " ")
      err := enc.Encode(recs)
      if err != nil {
         panic(err)
      }
      err = os.WriteFile("umber.json", text.Bytes(), os.ModePerm)
      if err != nil {
         panic(err)
      }
   } else {
      new_http().f.Usage()
      new_bandcamp().f.Usage()
      new_soundcloud().f.Usage()
      new_youtube().f.Usage()
   }
}

func records(config string) ([]*record, error) {
   text, err := os.ReadFile(config)
   if err != nil {
      return nil, err
   }
   var recs []*record
   err = json.Unmarshal(text, &recs)
   if err != nil {
      return nil, err
   }
   return recs, nil
}
