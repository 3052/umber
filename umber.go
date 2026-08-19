package main

import (
   "log"
   "os"
   "path/filepath"
)

const umber = "D:/41.neocities.org/umber"

var files = []string{
   "miller-display-italic.woff2",
   "overpass.woff2",
   "umber.css",
   "umber.png",
   "index.html",
   "platform.js",
   "umber.js",
   "umber.json",
}

func main() {
   err := os.RemoveAll(umber)
   if err != nil {
      log.Fatal(err)
   }
   err = os.MkdirAll(umber, 0755)
   if err != nil {
      log.Fatal(err)
   }
   for _, f := range files {
      data, err := os.ReadFile(f)
      if err != nil {
         log.Fatal(err)
      }
      dst := filepath.Join(umber, f)
      err = os.WriteFile(dst, data, 0644)
      if err != nil {
         log.Fatal(err)
      }
   }
}
