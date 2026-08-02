//go:build windows

package main

import (
   "flag"
   "fmt"
   "os"
   "path/filepath"
   "strings"
)

func main() {
   var (
      reverse    = flag.Bool("r", false, "reverse sort order")
      filesFirst = flag.Bool("files-first", false, "sort files before directories")
      natural    = flag.Bool("natural", false, "natural sort (file2 < file10)")
      dryRun     = flag.Bool("n", false, "dry run: don't write, just print what would happen")
   )
   flag.Usage = func() {
      fmt.Fprintf(os.Stderr, "Usage: %s X: [flags]\n", filepath.Base(os.Args[0]))
      flag.PrintDefaults()
   }
   flag.Parse()
   if flag.NArg() != 1 {
      flag.Usage()
      os.Exit(2)
   }
   drive := flag.Arg(0)
   opt := sortOptions{
      reverse:    *reverse,
      filesFirst: *filesFirst,
      natural:    *natural,
   }
   v, err := openVolume(drive)
   if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
   }
   defer func() {
      v.unlock()
      v.close()
   }()
   fmt.Printf("Sorting %s (%s)...\n", drive, v.bs.fatType())
   var rootCluster uint32
   if v.bs.fatType() == "FAT32" {
      rootCluster = v.rootCluster
   }
   if *dryRun {
      fmt.Println("(dry run — not implemented for listing; just run without -n)")
      return
   }
   if err := v.sortDirRecursive(rootCluster, 0, opt); err != nil {
      fmt.Fprintln(os.Stderr, "error:", err)
      os.Exit(1)
   }
   fmt.Println("Done.")
}

func (v *fatVolume) sortDirRecursive(c uint32, depth int, opt sortOptions) error {
   indent := strings.Repeat("  ", depth)
   entries, err := v.readDir(c)
   if err != nil {
      return err
   }
   skipCount := 0
   if c != 0 || v.bs.fatType() == "FAT32" {
      if len(entries) >= 2 &&
         !entries[0].isLFN && !entries[0].isFree && !entries[0].isLast &&
         entries[0].name == "." &&
         !entries[1].isLFN && !entries[1].isFree && !entries[1].isLast &&
         entries[1].name == ".." {
         skipCount = 2
      }
   }
   head := entries[:skipCount]
   rest := entries[skipCount:]
   groups := groupEntries(rest)
   sorted := sortGroups(groups, opt)
   var out []dirEntry
   out = append(out, head...)
   for _, g := range sorted {
      out = append(out, g.entries...)
   }
   hadEnd := false
   for _, e := range entries {
      if e.isLast {
         hadEnd = true
         break
      }
   }
   if !hadEnd {
      zero := make([]byte, 32)
      out = append(out, dirEntry{raw: zero, isLast: true})
   }
   if len(out)*32 != len(entries)*32 {
      targetBytes := len(entries) * 32
      if len(out)*32 > targetBytes {
         out = out[:targetBytes/32]
      } else {
         pad := (targetBytes - len(out)*32) / 32
         for i := 0; i < pad; i++ {
            out = append(out, dirEntry{raw: make([]byte, 32), isLast: true})
         }
      }
   }
   if err := v.writeDir(c, out); err != nil {
      return err
   }
   for _, g := range sorted {
      if len(g.entries) == 0 {
         continue
      }
      sfn := g.entries[len(g.entries)-1]
      if sfn.isLFN || sfn.isFree || sfn.isLast {
         continue
      }
      if !sfn.isDir {
         continue
      }
      if sfn.name == "." || sfn.name == ".." {
         continue
      }
      childCluster := sfn.cluster
      if childCluster < 2 {
         continue
      }
      fmt.Printf("%s%s/\n", indent, g.name)
      if err := v.sortDirRecursive(childCluster, depth+1, opt); err != nil {
         fmt.Fprintf(os.Stderr, "  error in %s: %v\n", g.name, err)
      }
   }
   return nil
}

// main.go
