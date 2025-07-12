package main

import (
   "41.neocities.org/platform/soundcloud"
   "bytes"
   "encoding/json"
   "flag"
   "log"
   "net/url"
   "os"
   "path"
   "slices"
   "strconv"
   "strings"
   "time"
)

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

type song struct {
   Q string
   S string
}

func read_songs(name string) ([]song, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return nil, err
   }
   var songs []song
   err = json.Unmarshal(data, &songs)
   if err != nil {
      return nil, err
   }
   return songs, nil
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "umber.json", "name")
   address := flag.String("a", "", "address")
   flag.Parse()
   if *address != "" {
      err := do_address(*address, *name)
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

func do_address(address, name string) error {
   // 1 resolve
   var resolve soundcloud.Resolve
   err := resolve.New(address)
   if err != nil {
      return err
   }
   // 2 values
   values := url.Values{}
   values.Set("a", strconv.FormatInt(time.Now().Unix(), 36))
   values.Set("b", strconv.FormatInt(resolve.Id, 10))
   values.Set("c",
      path.Base(strings.Replace(resolve.Artwork(), "large", "t500x500", 1)),
   )
   values.Set("p", "s")
   values.Set("y",
      strconv.Itoa(resolve.DisplayDate.Year()),
   )
   // 3 song
   var songVar song
   songVar.Q = values.Encode()
   songVar.S = resolve.Title
   // 4 songs
   songs, err := read_songs(name)
   if err != nil {
      return err
   }
   songs = slices.Insert(songs, 0, songVar)
   var buf bytes.Buffer
   encode := json.NewEncoder(&buf)
   encode.SetEscapeHTML(false)
   encode.SetIndent("", " ")
   err = encode.Encode(songs)
   if err != nil {
      return err
   }
   return write_file(name, buf.Bytes())
}
