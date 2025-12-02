package main

import (
   "log"
   "os"
)

const umber = "D:/41.neocities.org/umber"

func main() {
   err := os.RemoveAll(umber)
   if err != nil {
      log.Fatal(err)
   }
   err = os.CopyFS(umber, os.DirFS("page"))
   if err != nil {
      log.Fatal(err)
   }
}
