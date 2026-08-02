package main

import (
   "bytes"
   "encoding/binary"
   "fmt"
   "sort"
   "strings"
   "unicode"
   "unicode/utf16"
)

func naturalLess(a, b string) bool {
   la, lb := []rune(a), []rune(b)
   i, j := 0, 0
   for i < len(la) && j < len(lb) {
      ra, rb := la[i], lb[j]
      if unicode.IsDigit(ra) && unicode.IsDigit(rb) {
         si, sj := i, j
         for i < len(la) && unicode.IsDigit(la[i]) {
            i++
         }
         for j < len(lb) && unicode.IsDigit(lb[j]) {
            j++
         }
         na := strings.TrimLeft(string(la[si:i]), "0")
         nb := strings.TrimLeft(string(lb[sj:j]), "0")
         if len(na) != len(nb) {
            return len(na) < len(nb)
         }
         if na != nb {
            return na < nb
         }
         continue
      }
      if ra != rb {
         return ra < rb
      }
      i++
      j++
   }
   return len(la) < len(lb)
}

func parseLFN(b []byte) string {
   if len(b) != 32 {
      return ""
   }
   var u16 []uint16
   for i := 1; i <= 10; i += 2 {
      u16 = append(u16, binary.LittleEndian.Uint16(b[i:i+2]))
   }
   for i := 14; i <= 24; i += 2 {
      u16 = append(u16, binary.LittleEndian.Uint16(b[i:i+2]))
   }
   for i := 28; i <= 30; i += 2 {
      u16 = append(u16, binary.LittleEndian.Uint16(b[i:i+2]))
   }
   for i, c := range u16 {
      if c == 0x0000 || c == 0xFFFF {
         u16 = u16[:i]
         break
      }
   }
   return string(utf16.Decode(u16))
}

func parseSFN(b []byte) string {
   if len(b) != 32 {
      return ""
   }
   if b[0] == 0x05 {
      b = append([]byte{0xE5}, b[1:]...)
   }
   name := strings.TrimRight(string(b[0:8]), " ")
   ext := strings.TrimRight(string(b[8:11]), " ")
   if ext != "" {
      return name + "." + ext
   }
   return name
}

type dirEntry struct {
   raw     []byte
   name    string
   isDir   bool
   isLFN   bool
   isFree  bool
   isLast  bool
   cluster uint32
}

type group struct {
   entries []dirEntry
   name    string
   isDir   bool
}

func groupEntries(entries []dirEntry) []group {
   var groups []group
   i := 0
   for i < len(entries) {
      e := entries[i]
      if e.isLast || e.isFree {
         groups = append(groups, group{entries: []dirEntry{e}})
         i++
         continue
      }
      if e.isLFN {
         start := i
         for i < len(entries) && entries[i].isLFN {
            i++
         }
         if i < len(entries) && !entries[i].isLFN && !entries[i].isFree && !entries[i].isLast {
            sfn := entries[i]
            lfnEntries := entries[start:i]
            var nameParts []string
            for j := len(lfnEntries) - 1; j >= 0; j-- {
               nameParts = append(nameParts, lfnEntries[j].name)
            }
            name := strings.Join(nameParts, "")
            groups = append(groups, group{
               entries: entries[start : i+1],
               name:    name,
               isDir:   sfn.isDir,
            })
            i++
            continue
         }
         groups = append(groups, group{entries: entries[start:i]})
         continue
      }
      groups = append(groups, group{
         entries: []dirEntry{e},
         name:    e.name,
         isDir:   e.isDir,
      })
      i++
   }
   return groups
}

func sortGroups(groups []group, opt sortOptions) []group {
   var sortable []group
   var trailing []group
   for _, g := range groups {
      if len(g.entries) == 1 && (g.entries[0].isLast || g.entries[0].isFree) {
         trailing = append(trailing, g)
         continue
      }
      sortable = append(sortable, g)
   }
   less := func(x, y group) bool {
      if !opt.filesFirst {
         if x.isDir != y.isDir {
            return x.isDir
         }
      } else {
         if x.isDir != y.isDir {
            return !x.isDir
         }
      }
      ax := strings.ToLower(x.name)
      ay := strings.ToLower(y.name)
      if opt.natural {
         return naturalLess(ax, ay)
      }
      return ax < ay
   }
   sort.SliceStable(sortable, func(i, j int) bool { return less(sortable[i], sortable[j]) })
   if opt.reverse {
      for i, j := 0, len(sortable)-1; i < j; i, j = i+1, j-1 {
         sortable[i], sortable[j] = sortable[j], sortable[i]
      }
   }
   return append(sortable, trailing...)
}

type sortOptions struct {
   reverse    bool
   filesFirst bool
   natural    bool
}

func (v *fatVolume) readDir(c uint32) ([]dirEntry, error) {
   var raw []byte
   if c == 0 && v.bs.fatType() == "FAT16" {
      size := uint64(v.bs.rootEntries) * 32
      raw = make([]byte, size)
      if _, err := v.readAt(raw, int64(v.rootOffset)); err != nil {
         return nil, err
      }
   } else {
      for {
         clusterBuf := make([]byte, v.bytesPerCluster)
         off := int64(v.clusterByteOffset(c))
         if _, err := v.readAt(clusterBuf, off); err != nil {
            return nil, err
         }
         raw = append(raw, clusterBuf...)
         next, ok := v.nextCluster(c)
         if !ok {
            break
         }
         c = next
      }
   }
   var entries []dirEntry
   for i := 0; i+32 <= len(raw); i += 32 {
      e := dirEntry{raw: raw[i : i+32]}
      if e.raw[0] == endOfDir {
         e.isLast = true
         entries = append(entries, e)
         break
      }
      if e.raw[0] == deletedMark {
         e.isFree = true
         entries = append(entries, e)
         continue
      }
      attr := e.raw[11]
      if attr&attrLFN == attrLFN {
         e.isLFN = true
         e.name = parseLFN(e.raw)
         entries = append(entries, e)
         continue
      }
      e.name = parseSFN(e.raw)
      e.isDir = attr&attrDirectory != 0
      e.cluster = uint32(e.raw[26]) | uint32(e.raw[27])<<8 |
         uint32(e.raw[20])<<16 | uint32(e.raw[21])<<24
      entries = append(entries, e)
   }
   return entries, nil
}

func (v *fatVolume) writeDir(c uint32, entries []dirEntry) error {
   var buf bytes.Buffer
   for _, e := range entries {
      buf.Write(e.raw)
   }
   data := buf.Bytes()
   if c == 0 && v.bs.fatType() == "FAT16" {
      if uint64(len(data)) != uint64(v.bs.rootEntries)*32 {
         return fmt.Errorf("root dir size mismatch: got %d want %d",
            len(data), uint64(v.bs.rootEntries)*32)
      }
      _, err := v.writeAt(data, int64(v.rootOffset))
      return err
   }
   off := 0
   cur := c
   for off < len(data) {
      clusterBuf := data[off:]
      if uint32(len(clusterBuf)) > v.bytesPerCluster {
         clusterBuf = clusterBuf[:v.bytesPerCluster]
      }
      byteOff := int64(v.clusterByteOffset(cur))
      if _, err := v.writeAt(clusterBuf, byteOff); err != nil {
         return err
      }
      off += len(clusterBuf)
      next, ok := v.nextCluster(cur)
      if !ok {
         break
      }
      cur = next
   }
   return nil
}

// dirent.go
