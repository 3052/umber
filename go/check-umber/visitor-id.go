package main

import (
   "fmt"
   "io"
   "net/http"
   "net/url"
   "strings"
)

func main() {
   var req http.Request
   req.Header = http.Header{}
   req.URL = &url.URL{}
   req.URL.Host = "www.youtube.com"
   req.URL.Scheme = "https"
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      panic(err)
   }
   defer resp.Body.Close()
   data, err := io.ReadAll(resp.Body)
   if err != nil {
      panic(err)
   }
   data1 := string(data)
   i := strings.Index(data1, `Cgt`)
   fmt.Println(data1[i-99:][:999])
}
