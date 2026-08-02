//go:build windows

package main

import (
   "encoding/binary"
   "errors"
   "fmt"
   "strings"

   "golang.org/x/sys/windows"
)

const (
   attrReadOnly  = 0x01
   attrHidden    = 0x02
   attrSystem    = 0x04
   attrVolumeID  = 0x08
   attrDirectory = 0x10
   attrArchive   = 0x20
   attrLFN       = 0x0F

   endOfDir    = 0x00
   deletedMark = 0xE5
)

// FSCTL_* constants not available in older versions of golang.org/x/sys/windows.
// Derived via CTL_CODE(DeviceType, Function, Method, Access):
//
//   CTL_CODE(0x00000009, Function, METHOD_BUFFERED(0), FILE_ANY_ACCESS(0))
//   = (0x00000009 << 16) | (0 << 14) | (Function << 2) | 0
const (
   fsctlLockVolume     uint32 = 0x00090018
   fsctlUnlockVolume   uint32 = 0x0009001C
   fsctlDismountVolume uint32 = 0x00090020
)

type bootSector struct {
   raw               [512]byte
   bytesPerSector    uint16
   sectorsPerCluster uint8
   reservedSectors   uint16
   numFATs           uint8
   rootEntries       uint16
   totalSectors16    uint16
   fatSize16         uint16
   totalSectors32    uint32
   fatSize32         uint32
   rootCluster       uint32
   fsType            string
}

func parseBootSector(b []byte) (*bootSector, error) {
   if len(b) < 512 {
      return nil, errors.New("boot sector too short")
   }
   bs := &bootSector{}
   copy(bs.raw[:], b[:512])
   bs.bytesPerSector = binary.LittleEndian.Uint16(b[11:13])
   bs.sectorsPerCluster = b[13]
   bs.reservedSectors = binary.LittleEndian.Uint16(b[14:16])
   bs.numFATs = b[16]
   bs.rootEntries = binary.LittleEndian.Uint16(b[17:19])
   bs.totalSectors16 = binary.LittleEndian.Uint16(b[19:21])
   bs.fatSize16 = binary.LittleEndian.Uint16(b[22:24])
   bs.totalSectors32 = binary.LittleEndian.Uint32(b[32:36])
   bs.fatSize32 = binary.LittleEndian.Uint32(b[36:40])
   bs.rootCluster = binary.LittleEndian.Uint32(b[44:48])
   bs.fsType = strings.TrimSpace(strings.TrimRight(string(b[54:62]), "\x00"))
   if bs.bytesPerSector == 0 {
      return nil, errors.New("invalid bytes per sector")
   }
   return bs, nil
}

func (bs *bootSector) bytesPerCluster() uint32 {
   return uint32(bs.bytesPerSector) * uint32(bs.sectorsPerCluster)
}

func (bs *bootSector) fatSize() uint32 {
   if bs.fatSize16 != 0 {
      return uint32(bs.fatSize16)
   }
   return bs.fatSize32
}

func (bs *bootSector) fatType() string {
   if bs.rootEntries != 0 {
      return "FAT16"
   }
   return "FAT32"
}

type fatVolume struct {
   handle            windows.Handle
   bs                *bootSector
   fatOffset         uint64
   fatBytes          uint64
   dataOffset        uint64
   rootOffset        uint64
   rootCluster       uint32
   fat               []byte
   sectorsPerCluster uint8
   bytesPerSector    uint16
   bytesPerCluster   uint32
}

func openVolume(drive string) (*fatVolume, error) {
   if len(drive) >= 2 && drive[1] == ':' {
      drive = drive[:2]
   }
   path := `\\.\` + drive
   h, err := windows.CreateFile(
      windows.StringToUTF16Ptr(path),
      windows.GENERIC_READ|windows.GENERIC_WRITE,
      windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
      nil,
      windows.OPEN_EXISTING,
      windows.FILE_ATTRIBUTE_NORMAL,
      0,
   )
   if err != nil {
      return nil, fmt.Errorf("CreateFile %s: %w (are you running as Administrator?)", path, err)
   }
   v := &fatVolume{handle: h}
   if err := v.ioctl(fsctlLockVolume); err != nil {
      v.close()
      return nil, fmt.Errorf("FSCTL_LOCK_VOLUME: %w", err)
   }
   if err := v.ioctl(fsctlDismountVolume); err != nil {
      v.unlock()
      v.close()
      return nil, fmt.Errorf("FSCTL_DISMOUNT_VOLUME: %w", err)
   }
   boot := make([]byte, 512)
   if _, err := v.readAt(boot, 0); err != nil {
      v.unlock()
      v.close()
      return nil, fmt.Errorf("read boot sector: %w", err)
   }
   bs, err := parseBootSector(boot)
   if err != nil {
      v.unlock()
      v.close()
      return nil, err
   }
   v.bs = bs
   v.bytesPerSector = bs.bytesPerSector
   v.sectorsPerCluster = bs.sectorsPerCluster
   v.bytesPerCluster = bs.bytesPerCluster()
   reserved := uint64(bs.reservedSectors) * uint64(bs.bytesPerSector)
   v.fatOffset = reserved
   v.fatBytes = uint64(bs.fatSize()) * uint64(bs.bytesPerSector)
   v.fat = make([]byte, v.fatBytes)
   if _, err := v.readAt(v.fat, int64(v.fatOffset)); err != nil {
      v.unlock()
      v.close()
      return nil, fmt.Errorf("read FAT: %w", err)
   }
   fatSectors := uint64(bs.fatSize()) * uint64(bs.numFATs)
   if bs.fatType() == "FAT16" {
      rootEntries := uint64(bs.rootEntries)
      v.rootOffset = reserved + fatSectors*uint64(bs.bytesPerSector)
      v.dataOffset = v.rootOffset + rootEntries*32
   } else {
      v.rootCluster = bs.rootCluster
      v.dataOffset = reserved + fatSectors*uint64(bs.bytesPerSector)
   }
   fmt.Printf("Detected %s  bytes/sector=%d  sectors/cluster=%d  rootCluster=%d\n",
      bs.fatType(), bs.bytesPerSector, bs.sectorsPerCluster, v.rootCluster)
   return v, nil
}

func (v *fatVolume) close() { windows.CloseHandle(v.handle) }

func (v *fatVolume) clusterByteOffset(c uint32) uint64 {
   return v.dataOffset + uint64(c-2)*uint64(v.bytesPerCluster)
}

func (v *fatVolume) ioctl(code uint32) error {
   var bytesReturned uint32
   return windows.DeviceIoControl(v.handle, code, nil, 0, nil, 0, &bytesReturned, nil)
}

func (v *fatVolume) nextCluster(c uint32) (uint32, bool) {
   switch v.bs.fatType() {
   case "FAT16":
      off := c * 2
      if int(off+2) > len(v.fat) {
         return 0, false
      }
      n := binary.LittleEndian.Uint16(v.fat[off:])
      if n >= 0xFFF8 {
         return 0, false
      }
      if n == 0xFFF7 {
         return 0, false
      }
      return uint32(n), true
   case "FAT32":
      off := c * 4
      if int(off+4) > len(v.fat) {
         return 0, false
      }
      n := binary.LittleEndian.Uint32(v.fat[off:]) & 0x0FFFFFFF
      if n >= 0x0FFFFFF8 {
         return 0, false
      }
      if n == 0x0FFFFFF7 {
         return 0, false
      }
      return n, true
   }
   return 0, false
}

func (v *fatVolume) readAt(p []byte, off int64) (int, error) {
   ov := windows.Overlapped{Offset: uint32(off), OffsetHigh: uint32(off >> 32)}
   var n uint32
   err := windows.ReadFile(v.handle, p, &n, &ov)
   if err != nil {
      return int(n), err
   }
   if int(n) != len(p) {
      return int(n), fmt.Errorf("short read: got %d want %d", n, len(p))
   }
   return int(n), nil
}
func (v *fatVolume) unlock() { _ = v.ioctl(fsctlUnlockVolume) }

func (v *fatVolume) writeAt(p []byte, off int64) (int, error) {
   ov := windows.Overlapped{Offset: uint32(off), OffsetHigh: uint32(off >> 32)}
   var n uint32
   err := windows.WriteFile(v.handle, p, &n, &ov)
   if err != nil {
      return int(n), err
   }
   if int(n) != len(p) {
      return int(n), fmt.Errorf("short write: got %d want %d", n, len(p))
   }
   return int(n), nil
}

// volume.go
