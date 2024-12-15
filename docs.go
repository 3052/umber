package main

import "os"

func main() {
   err := os.CopyFS("../41.neocities.org/umber", os.DirFS("docs"))
   if err != nil {
      panic(err)
   }
}
