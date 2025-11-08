package main

import (
   "encoding/json"
   "io"
   "net/http"
   "net/url"
   "os"
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
   var cfg yt_cfg
   err = json.NewDecoder(strings.NewReader(data1)).Decode(&cfg)
   if err != nil {
      panic(err)
   }
   encode := json.NewEncoder(os.Stdout)
   encode.SetIndent("", " ")
   err = encode.Encode(cfg)
   if err != nil {
      panic(err)
   }
}

type yt_cfg struct {
   InnertubeClientName string `json:"INNERTUBE_CLIENT_NAME"`
   InnertubeClientVersion string `json:"INNERTUBE_CLIENT_VERSION"`
   InnertubeContext struct {
      Client struct {
         VisitorData visitor_data
      }
   } `json:"INNERTUBE_CONTEXT"`
   InnertubeContextClientName int `json:"INNERTUBE_CONTEXT_CLIENT_NAME"`
   InnertubeContextClientVersion string `json:"INNERTUBE_CONTEXT_CLIENT_VERSION"`
}

type visitor_data string

func (v *visitor_data) UnmarshalText(data []byte) error {
   visitor, err := url.PathUnescape(string(data))
   if err != nil {
      return err
   }
   *v = visitor_data(visitor)
   return nil
}
