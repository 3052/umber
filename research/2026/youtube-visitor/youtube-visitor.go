package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
)

func do() error {
   req := http.Request{
      URL: &url.URL{
         Scheme: "https",
         Host: "www.youtube.com",
      },
      Header: http.Header{},
   }
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return err
   }
   data, err = extractJSON(data, []byte(sep))
   if err != nil {
      return err
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

// extractJSON isolates the JSON payload by balancing curly braces 
// directly on a byte slice to avoid memory allocations.
func extractJSON(content []byte, prefix[]byte) ([]byte, error) {
   _, after, found := bytes.Cut(content, prefix)
   if !found {
      return nil, fmt.Errorf("prefix %q not found in file", prefix)
   }
   if len(after) == 0 {
      return nil, fmt.Errorf("content ends abruptly after prefix")
   }
   if after[0] != '{' {
      return nil, fmt.Errorf("expected '{' at the start of JSON, got %c", after[0])
   }
   openBraces := 0
   inString := false
   escapeNext := false
   // Parse through the bytes to find the exact end of the JSON object.
   // We can use a clean range loop now that we are looking exclusively at the 'after' slice.
   for i, char := range after {
      // Handle escaped characters (e.g., \")
      if escapeNext {
         escapeNext = false
         continue
      }
      if char == '\\' {
         escapeNext = true
         continue
      }
      // Toggle string state to ignore braces inside strings
      if char == '"' {
         inString = !inString
         continue
      }
      // If we are not inside a string literal, count braces
      if !inString {
         if char == '{' {
            openBraces++
         } else if char == '}' {
            openBraces--
            // When the count goes back to 0, we've found the end of the JSON body
            if openBraces == 0 {
               // Return the exact slice of bytes representing the JSON object
               return after[:i+1], nil
            }
         }
      }
   }
   return nil, fmt.Errorf("could not find the matching closing brace for the JSON object")
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

const sep = "\nytcfg.set("

func main() {
   err := do()
   if err != nil {
      log.Fatal(err)
   }
}
