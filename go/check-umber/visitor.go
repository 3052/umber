package main

import (
   "41.neocities.org/protobuf"
   "encoding/base64"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "strings"
)

const sep = "\nytcfg.set("

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
   _, data1, found := strings.Cut(string(data), sep)
   if !found {
      panic(sep)
   }
   var value struct {
      InnertubeContext struct {
         Client struct {
            VisitorData string
         }
      } `json:"INNERTUBE_CONTEXT"`
   }
   err = json.NewDecoder(strings.NewReader(data1)).Decode(&value)
   if err != nil {
      panic(err)
   }
   visitor, err := url.PathUnescape(value.InnertubeContext.Client.VisitorData)
   if err != nil {
      panic(err)
   }
   data, err = base64.StdEncoding.DecodeString(visitor)
   if err != nil {
      panic(err)
   }
   var message protobuf.Message
   err = message.Unmarshal(data)
   if err != nil {
      panic(err)
   }
   fmt.Printf("%#v\n", message)
}
