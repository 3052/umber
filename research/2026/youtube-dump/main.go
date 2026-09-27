// main.go marker preserve
package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
   "strings"
)

func main() {
   video := flag.String("v", "", "YouTube video ID")
   flag.Parse()
   if *video == "" {
      flag.Usage()
      os.Exit(2)
   }

   visitorID, err := fetchVisitorID()
   if err != nil {
      log.Fatal(err)
   }

   body, err := playerResponse(*video, visitorID)
   if err != nil {
      log.Fatal(err)
   }

   var pd playerData
   if err := json.Unmarshal(body, &pd); err != nil {
      log.Fatal(err)
   }

   switch {
   case pd.PlayabilityStatus.Status == "LOGIN_REQUIRED" &&
      strings.Contains(pd.PlayabilityStatus.Reason, "not a bot"):
      log.Fatalf("%v: %s — %s", errVisitorExpired,
         pd.PlayabilityStatus.Status, pd.PlayabilityStatus.Reason)
   case pd.PlayabilityStatus.Status == "UNPLAYABLE" &&
      pd.PlayabilityStatus.Reason == visitorExpiredReason:
      log.Fatalf("%v: %s — %s", errVisitorExpired,
         pd.PlayabilityStatus.Status, pd.PlayabilityStatus.Reason)
   }

   af := pd.StreamingData.AdaptiveFormats
   if len(af) == 0 || string(af) == "null" {
      log.Fatalf("no adaptiveFormats in response (playability %s — %s)",
         pd.PlayabilityStatus.Status, pd.PlayabilityStatus.Reason)
   }

   // json.Indent keeps the server's field order; MarshalIndent would
   // sort keys because the top level is a map.
   var out bytes.Buffer
   if err := json.Indent(&out, af, "", "  "); err != nil {
      log.Fatal(err)
   }
   fmt.Println(out.String())
}

// playerData pulls just what we need from the /player response.
// AdaptiveFormats stays a RawMessage so the server's key order and
// formatting survive untouched.
type playerData struct {
   PlayabilityStatus struct {
      Status string
      Reason string
   }
   StreamingData struct {
      AdaptiveFormats json.RawMessage `json:"adaptiveFormats"`
   } `json:"streamingData"`
}

// main.go marker preserve
