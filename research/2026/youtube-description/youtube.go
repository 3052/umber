// youtube.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "errors"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "strings"
)

const sep = "\nytcfg.set("

var (
   errRateLimited    = fmt.Errorf("rate limited")
   errVisitorExpired = fmt.Errorf("visitor ID expired")
)

// extractJSON isolates the JSON payload by balancing curly braces.
func extractJSON(content []byte, prefix []byte) ([]byte, error) {
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
   for i, char := range after {
      if escapeNext {
         escapeNext = false
         continue
      }
      if char == '\\' {
         escapeNext = true
         continue
      }
      if char == '"' {
         inString = !inString
         continue
      }
      if !inString {
         if char == '{' {
            openBraces++
         } else if char == '}' {
            openBraces--
            if openBraces == 0 {
               return after[:i+1], nil
            }
         }
      }
   }
   return nil, fmt.Errorf("could not find the matching closing brace for the JSON object")
}

func fetchVisitorID() (string, error) {
   resp, err := http.Get("https://www.youtube.com")
   if err != nil {
      return "", err
   }
   defer resp.Body.Close()
   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return "", err
   }
   data, err = extractJSON(data, []byte(sep))
   if err != nil {
      return "", err
   }
   var result ytCfg
   if err := json.Unmarshal(data, &result); err != nil {
      return "", err
   }
   return string(result.InnertubeContext.Client.VisitorData), nil
}

type player struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   VideoDetails struct {
      Author           string
      ShortDescription string
      Title            string
   }
}

func fetch_player(video_id, visitorID string) (*player, error) {
   data, err := json.Marshal(map[string]any{
      "contentCheckOk": true,
      "context": map[string]any{
         "client": map[string]string{
            "clientName":    "WEB",
            "clientVersion": "2.20231219.04.00",
         },
      },
      "racyCheckOk": true,
      "videoId":     video_id,
   })
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("X-Goog-Visitor-Id", visitorID)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode == http.StatusTooManyRequests {
      return nil, fmt.Errorf("%w: HTTP 429", errRateLimited)
   }
   if resp.StatusCode != http.StatusOK {
      return nil, errors.New(resp.Status)
   }
   result := &player{}
   if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
      return nil, err
   }
   if result.PlayabilityStatus.Status == "LOGIN_REQUIRED" &&
      strings.Contains(result.PlayabilityStatus.Reason, "not a bot") {
      return nil, fmt.Errorf("%w: %s — %s", errVisitorExpired,
         result.PlayabilityStatus.Status, result.PlayabilityStatus.Reason)
   }
   return result, nil
}

type visitorData string

func (v *visitorData) UnmarshalText(data []byte) error {
   visitor, err := url.PathUnescape(string(data))
   if err != nil {
      return err
   }
   *v = visitorData(visitor)
   return nil
}

type ytCfg struct {
   InnertubeContext struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
}

// youtube.go marker preserve
