package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
)

const sep = "\nytcfg.set("

func do() error {
   var req http.Request
   req.Header = http.Header{}
   req.URL = &url.URL{}
   req.URL.Host = "www.youtube.com"
   req.URL.Scheme = "https"
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return err
   }
   var found bool
   _, data, found = bytes.Cut(data, []byte(sep))
   if !found {
      return errors.New(sep)
   }
   var result yt_cfg
   err = json.Unmarshal(data, &result)
   if err != nil {
      return err
   }
   encode := json.NewEncoder(os.Stdout)
   encode.SetIndent("", " ")
   return encode.Encode(result)
}

func main() {
   err := do()
   if err != nil {
      log.Fatal(err)
   }
}

type yt_cfg struct {
   InnertubeClientName    string `json:"INNERTUBE_CLIENT_NAME"`
   InnertubeClientVersion string `json:"INNERTUBE_CLIENT_VERSION"`
   InnertubeContext       struct {
      Client struct {
         VisitorData visitor_data
      }
   } `json:"INNERTUBE_CONTEXT"`
   InnertubeContextClientName    int    `json:"INNERTUBE_CONTEXT_CLIENT_NAME"`
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
