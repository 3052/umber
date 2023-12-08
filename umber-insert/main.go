package main

import (
   "bytes"
   "embed"
   "encoding/json"
   "os"
)

//go:embed config.json
var fs embed.FS

type config struct {
   Path string
}

type record struct {
   Q string
   S string
}

func (c config) records() ([]*record, error) {
   text, err := os.ReadFile(c.Path)
   if err != nil {
      return nil, err
   }
   var recs []*record
   if err := json.Unmarshal(text, &recs); err != nil {
      return nil, err
   }
   return recs, nil
}

func new_config() (*config, error) {
   text, err := fs.ReadFile("config.json")
   if err != nil {
      return nil, err
   }
   con := new(config)
   if err := json.Unmarshal(text, con); err != nil {
      return nil, err
   }
   return con, nil
}

func main() {
   con, err := new_config()
   if err != nil {
      panic(err)
   }
   recs, err := con.records()
   if err != nil {
      panic(err)
   }
   if len(os.Args) >= 3 {
      arg := os.Args[2:]
      var rec *record
      switch os.Args[1] {
      case "backblaze":
         rec, err = new_backblaze().parse(arg)
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
      if err := enc.Encode(recs); err != nil {
         panic(err)
      }
      if err := os.WriteFile(con.Path, text.Bytes(), 0666); err != nil {
         panic(err)
      }
   } else {
      new_backblaze().Usage()
      new_bandcamp().Usage()
      new_soundcloud().Usage()
      new_youtube().Usage()
   }
}
