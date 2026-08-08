// kmfbuild.go - Build a Rockbox binary keyremap (.kmf) file from a JSON keymap.
//
// Fetches action.h, hibyr1.h and the R1 button-target.h from the Rockbox GitHub
// mirror at the commit matching your installed build (these contain sequential
// enums whose values shift between builds). Core BUTTON_* event flags are
// stable hex constants and are hardcoded below.
//
// Usage:
//
//   go run kmfbuild.go -rev 20f4f9539a -in keymap.json -out keyremap.kmf
package main

import (
   "encoding/binary"
   "encoding/json"
   "flag"
   "fmt"
   "io"
   "net/http"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

const rawBase = "https://raw.githubusercontent.com/Rockbox/rockbox/%s/%s"

var (
   rev     = flag.String("rev", "master", "git commit/branch matching your build (System > About Rockbox)")
   inFile  = flag.String("in", "", "input JSON keymap file")
   outFile = flag.String("out", "keyremap.kmf", "output .kmf file")
)

// ---------- header parsing ----------

var (
   reEnumBlock = regexp.MustCompile(`(?s)enum\s+\w+\s*\{(.*?)\};`)
   reEnumItem  = regexp.MustCompile(`^\s*([A-Z][A-Z0-9_]*)\s*(?:=\s*(.+?))?\s*$`)
   reDefine    = regexp.MustCompile(`(?m)^#define\s+([A-Z][A-Z0-9_]+)\s+(.+?)\s*(?://.*)?$`)
)

// Hardcoded from firmware/export/button.h - these are universal hex flags
// shared by all targets and have been stable for many years.
var coreButtonFlags = map[string]int32{
   "BUTTON_NONE":   0x00000000,
   "BUTTON_REL":    0x02000000,
   "BUTTON_REPEAT": 0x04000000,
}

// Only headers with build-specific values (sequential enums, TARGET_ID).
var headerFiles = []string{
   "apps/action.h",                                  // ACTION_*, CONTEXT_*, LAST_*_PLACEHOLDER
   "firmware/export/config/hibyr1.h",                // TARGET_ID
   "firmware/target/hosted/hiby/r1/button-target.h", // BUTTON_PLAY, BUTTON_VOL_UP, etc.
}

// ---------- binary emission ----------

func buildBinary(groups []ctxGroup, headerID, version, stop int32) []byte {
   // Offsets are relative to the first entry AFTER the header.
   // Post-header layout: [ctx table][sentinel][group1 entries][sentinel][group2...]
   offset := int32(len(groups) + 1)
   var entries []mapping
   for _, g := range groups {
      entries = append(entries, mapping{g.context, offset, int32(len(g.entries))})
      offset += int32(len(g.entries)) + 1
   }
   entries = append(entries, mapping{stop, 0, 0})
   for _, g := range groups {
      entries = append(entries, g.entries...)
      entries = append(entries, mapping{stop, 0, 0})
   }

   total := int32(len(entries) + 1)
   buf := make([]byte, 0, total*12)
   put := func(m mapping) {
      var b [12]byte
      binary.LittleEndian.PutUint32(b[0:4], uint32(m.action))
      binary.LittleEndian.PutUint32(b[4:8], uint32(m.button))
      binary.LittleEndian.PutUint32(b[8:12], uint32(m.prebtn))
      buf = append(buf, b[:]...)
   }
   put(mapping{version, headerID, total})
   for _, m := range entries {
      put(m)
   }
   return buf
}

func evalExpr(expr string, consts map[string]int32) (int32, error) {
   expr = strings.Trim(expr, "() \t")
   var out int32
   for _, p := range strings.Split(expr, "|") {
      p = strings.Trim(p, "() \t")
      if p == "" {
         continue
      }
      if v, err := strconv.ParseInt(p, 0, 32); err == nil {
         out |= int32(v)
         continue
      }
      if v, ok := consts[p]; ok {
         out |= v
         continue
      }
      return 0, fmt.Errorf("cannot resolve %q", p)
   }
   return out, nil
}

func fatal(format string, args ...interface{}) {
   fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
   os.Exit(1)
}

func fetch(url string) (string, error) {
   client := &http.Client{Timeout: 30 * time.Second}
   resp, err := client.Get(url)
   if err != nil {
      return "", err
   }
   defer resp.Body.Close()
   if resp.StatusCode != 200 {
      return "", fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
   }
   b, err := io.ReadAll(resp.Body)
   return string(b), err
}

func main() {
   flag.Parse()
   if *inFile == "" {
      flag.Usage()
      os.Exit(1)
   }

   consts := map[string]int32{}
   for k, v := range coreButtonFlags {
      consts[k] = v
   }

   for _, path := range headerFiles {
      fmt.Printf("fetching %s @ %s\n", path, *rev)
      body, err := fetch(fmt.Sprintf(rawBase, *rev, path))
      must(err)
      parseEnums(body, consts)
      parseDefines(body, consts)
   }

   targetID, ok := consts["TARGET_ID"]
   if !ok {
      fatal("TARGET_ID not found in hibyr1.h")
   }
   lastAction, ok := consts["LAST_ACTION_PLACEHOLDER"]
   if !ok {
      fatal("LAST_ACTION_PLACEHOLDER not found in action.h")
   }
   remapped, ok := consts["CONTEXT_REMAPPED"]
   if !ok {
      fatal("CONTEXT_REMAPPED not found in action.h")
   }
   stop, ok := consts["CONTEXT_STOPSEARCHING"]
   if !ok {
      fatal("CONTEXT_STOPSEARCHING not found in action.h")
   }
   const keyremapVersion = 1 // KEYREMAP_VERSION from core_keymap.h
   headerID := lastAction | (targetID << 8)

   fmt.Printf("TARGET_ID=%d LAST_ACTION_PLACEHOLDER=%d CONTEXT_REMAPPED=0x%x STOP=0x%x headerID=0x%x\n",
      targetID, lastAction, remapped, stop, headerID)

   groups, err := parseJSONKeymap(*inFile, consts, remapped)
   must(err)

   out := buildBinary(groups, headerID, keyremapVersion, stop)
   must(os.WriteFile(*outFile, out, 0644))
   fmt.Printf("wrote %s (%d bytes)\n", *outFile, len(out))
}

func must(err error) {
   if err != nil {
      fatal("%v", err)
   }
}

func parseDefines(body string, consts map[string]int32) {
   for _, m := range reDefine.FindAllStringSubmatch(body, -1) {
      if v, err := evalExpr(m[2], consts); err == nil {
         consts[m[1]] = v
      }
   }
}

func parseEnums(body string, consts map[string]int32) {
   for _, block := range reEnumBlock.FindAllStringSubmatch(body, -1) {
      val := int32(-1)
      for _, item := range strings.Split(block[1], ",") {
         if i := strings.Index(item, "/*"); i >= 0 {
            item = item[:i]
         }
         if i := strings.Index(item, "//"); i >= 0 {
            item = item[:i]
         }
         item = strings.TrimSpace(item)
         if item == "" {
            continue
         }
         m := reEnumItem.FindStringSubmatch(item)
         if m == nil {
            continue
         }
         if m[2] != "" {
            if v, err := evalExpr(m[2], consts); err == nil {
               val = v
            } else {
               continue
            }
         } else {
            val++
         }
         consts[m[1]] = val
      }
   }
}

// resolveList ORs a list of button names (empty list = BUTTON_NONE = 0).
func resolveList(names []string, consts map[string]int32) (int32, error) {
   var out int32
   for _, n := range names {
      v, ok := consts[n]
      if !ok {
         return 0, fmt.Errorf("unknown button %q", n)
      }
      out |= v
   }
   return out, nil
}

type ctxGroup struct {
   context int32
   entries []mapping
}

func parseJSONKeymap(path string, consts map[string]int32, remappedBit int32) ([]ctxGroup, error) {
   data, err := os.ReadFile(path)
   if err != nil {
      return nil, err
   }
   var km jsonKeymap
   if err := json.Unmarshal(data, &km); err != nil {
      return nil, fmt.Errorf("parsing %s: %v", path, err)
   }

   var groups []ctxGroup
   for ci, c := range km.Contexts {
      ctxVal, ok := consts[c.Name]
      if !ok {
         return nil, fmt.Errorf("contexts[%d]: unknown context %q", ci, c.Name)
      }
      g := ctxGroup{context: ctxVal | remappedBit}
      for ei, e := range c.Entries {
         where := fmt.Sprintf("contexts[%d].entries[%d] (%s)", ci, ei, e.Action)
         act, ok := consts[e.Action]
         if !ok {
            return nil, fmt.Errorf("%s: unknown action %q", where, e.Action)
         }
         btn, err := resolveList(e.Button, consts)
         if err != nil {
            return nil, fmt.Errorf("%s: button: %v", where, err)
         }
         pre, err := resolveList(e.Prebtn, consts)
         if err != nil {
            return nil, fmt.Errorf("%s: prebtn: %v", where, err)
         }
         g.entries = append(g.entries, mapping{int32(act), btn, pre})
      }
      groups = append(groups, g)
   }
   if len(groups) == 0 {
      return nil, fmt.Errorf("no contexts found in %s", path)
   }
   return groups, nil
}

// ---------- JSON keymap parsing ----------

// Input schema:
// {
//   "contexts": [
//     {
//       "name": "CONTEXT_WPS",
//       "entries": [
//         {
//           "comment": "optional - ignored, for humans",
//           "action":  "ACTION_WPS_PLAY",
//           "button":  ["BUTTON_VOL_UP", "BUTTON_REPEAT"],
//           "prebtn":  ["BUTTON_VOL_UP"]
//         }
//       ]
//     }
//   ]
// }

type jsonKeymap struct {
   Contexts []struct {
      Name    string `json:"name"`
      Entries []struct {
         Comment string   `json:"comment,omitempty"`
         Action  string   `json:"action"`
         Button  []string `json:"button"`
         Prebtn  []string `json:"prebtn"`
      } `json:"entries"`
   } `json:"contexts"`
}

type mapping struct {
   action int32
   button int32
   prebtn int32
}
