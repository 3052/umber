package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "regexp"
   "strconv"
   "sync"
)

const sep = "\nytcfg.set("

var (
   stsCache = map[string]int{}
   stsMu    sync.Mutex
   stsRe    = regexp.MustCompile(`["']?signatureTimestamp["']?\s*[:=]\s*(\d+)`)
)

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
   targetUrl := &url.URL{Scheme: "https", Host: "www.youtube.com"}
   req := &http.Request{
      Method: http.MethodGet,
      URL:    targetUrl,
   }
   log.Println("fetching visitor ID from", req.URL)
   resp, err := http.DefaultClient.Do(req)
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

// signatureTimestamp extracts the signature timestamp (sts) from the player
// base.js. Current YouTube clients must send it in
// playbackContext.contentPlaybackContext, or /player returns UNPLAYABLE.
// Results are cached: sts only changes when the player build changes.
func signatureTimestamp(playerJSPath string) (int, error) {
   stsMu.Lock()
   defer stsMu.Unlock()

   if sts, ok := stsCache[playerJSPath]; ok {
      return sts, nil
   }

   jsURL := playerJSPath
   if jsURL[0] == '/' {
      jsURL = "https://www.youtube.com" + jsURL
   }
   resp, err := http.Get(jsURL)
   if err != nil {
      return 0, err
   }
   defer resp.Body.Close()

   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return 0, err
   }

   m := stsRe.FindSubmatch(data)
   if m == nil {
      return 0, fmt.Errorf("signatureTimestamp not found in %s", playerJSPath)
   }
   sts, err := strconv.Atoi(string(m[1]))
   if err != nil {
      return 0, fmt.Errorf("parse signatureTimestamp: %w", err)
   }
   stsCache[playerJSPath] = sts
   return sts, nil
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

type watchConfig struct {
   PlayerJSURL string      `json:"PLAYER_JS_URL"`
   VisitorData visitorData `json:"VISITOR_DATA"`
}

// fetchWatchConfig fetches the regular watch page. Its ytcfg provides a fresh
// visitor ID and PLAYER_JS_URL, which points at the player base.js build used
// to extract the signature timestamp.
func fetchWatchConfig(videoID string) (*watchConfig, error) {
   resp, err := http.Get("https://www.youtube.com/watch?v=" + videoID)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()

   data, err := io.ReadAll(resp.Body)
   if err != nil {
      return nil, err
   }

   data, err = extractJSON(data, []byte(sep))
   if err != nil {
      return nil, err
   }

   var cfg watchConfig
   if err := json.Unmarshal(data, &cfg); err != nil {
      return nil, err
   }
   if cfg.PlayerJSURL == "" {
      return nil, fmt.Errorf("no PLAYER_JS_URL in watch config")
   }
   return &cfg, nil
}

type ytCfg struct {
   InnertubeClientName    string `json:"INNERTUBE_CLIENT_NAME"`
   InnertubeClientVersion string `json:"INNERTUBE_CLIENT_VERSION"`
   InnertubeContext       struct {
      Client struct {
         VisitorData visitorData
      }
   } `json:"INNERTUBE_CONTEXT"`
   InnertubeContextClientName    int    `json:"INNERTUBE_CONTEXT_CLIENT_NAME"`
   InnertubeContextClientVersion string `json:"INNERTUBE_CONTEXT_CLIENT_VERSION"`
}

// visitor.go
