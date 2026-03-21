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

const sep = "\nytcfg.set("

func main() {
   err := do()
   if err != nil {
      log.Fatal(err)
   }
}

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
func extractJSON(content []byte, prefix []byte) ([]byte, error) {
   startIdx := bytes.Index(content, prefix)
   if startIdx == -1 {
      return nil, fmt.Errorf("prefix %q not found in file", prefix)
   }
   // Move the index forward to where the JSON object actually begins
   jsonStart := startIdx + len(prefix)
   // Make sure we haven't run out of bytes
   if jsonStart >= len(content) {
      return nil, fmt.Errorf("content ends abruptly after prefix")
   }
   // Make sure we are actually starting at a curly brace
   if content[jsonStart] != '{' {
      return nil, fmt.Errorf("expected '{' at the start of JSON, got %c", content[jsonStart])
   }
   openBraces := 0
   inString := false
   escapeNext := false
   // Parse through the bytes to find the exact end of the JSON object
   for i := jsonStart; i < len(content); i++ {
      char := content[i]
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
               return content[jsonStart : i+1], nil
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
