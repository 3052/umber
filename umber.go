package main

import "os"

const umber = "D:/41.neocities.org/umber"

func main() {
   err := os.RemoveAll(umber)
   if err != nil {
      panic(err)
   }
   err = os.CopyFS(umber, os.DirFS("page"))
   if err != nil {
      panic(err)
   }
}
