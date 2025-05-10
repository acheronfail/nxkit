package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/acheronfail/nxkit/lib/utils"
)

const (
	longFileNameUtf16Length = 13
)

type Entry struct {
	fatDirectoryEntry
	longFileName string
}

// IsArchive implements fs.Stat.
func (d *Entry) IsArchive() bool {
	return d.fatDirectoryEntry.IsArchive()
}

// IsDir implements fs.Stat.
func (d *Entry) IsDir() bool {
	return d.fatDirectoryEntry.IsDir()
}

// IsFile implements fs.Stat.
func (d *Entry) IsFile() bool {
	return !d.fatDirectoryEntry.IsDir()
}

// IsHidden implements fs.Stat.
func (d *Entry) IsHidden() bool {
	return d.fatDirectoryEntry.IsHidden()
}

// IsReadOnly implements fs.Stat.
func (d *Entry) IsReadOnly() bool {
	return d.fatDirectoryEntry.IsReadOnly()
}

// IsSystem implements fs.Stat.
func (d *Entry) IsSystem() bool {
	return d.fatDirectoryEntry.IsSystem()
}

// IsVolumeId implements fs.Stat.
func (d *Entry) IsVolumeId() bool {
	return d.fatDirectoryEntry.IsVolumeId()
}

// Size implements fs.Stat.
func (d *Entry) Size() int64 {
	return int64(d.fatDirectoryEntry.DIR_FileSize)
}

type fatDirectoryEntry struct {
	// Short file name (SFN) of the object.
	DIR_Name [11]byte
	//File attribute in combination of following flags. Upper 2 bits are reserved and must be zero.
	// 0x01: ATTR_READ_ONLY (Read-only)
	// 0x02: ATTR_HIDDEN (Hidden)
	// 0x04: ATTR_SYSTEM (System)
	// 0x08: ATTR_VOLUME_ID (Volume label)
	// 0x10: ATTR_DIRECTORY (Directory)
	// 0x20: ATTR_ARCHIVE (Archive)
	// 0x0F: ATTR_LONG_FILE_NAME (LFN entry)
	DIR_Attr uint8
	// 	Optional flags that indicates case information of the SFN.
	// 0x08: Every alphabet in the body is low-case.
	// 0x10: Every alphabet in the extensiton is low-case.
	DIR_NTRes uint8
	// Optional sub-second information corresponds to DIR_CrtTime.
	// The time resolution of DIR_CrtTime is 2 seconds, so that this field gives a count of sub-second and its valid value
	// range is from 0 to 199 in unit of 10 miliseconds. If not supported, set zero and do not change afterwards.
	DIR_CrtTimeTenth uint8
	// Optional file creation time. If not supported, set zero and do not change afterwards.
	DIR_CrtTime uint16
	// Optional file creation date. If not supported, set zero and do not change afterwards.
	DIR_CrtDate uint16
	// Optional last accesse date. There is no time information about last accesse time, so that the resolution of last
	// accesse time is 1 day. If not supported, set zero and do not change afterwards.
	DIR_LstAccDate uint16
	// Upeer part of cluster number. Always zero on the FAT12/16 volume. See DIR_FstClusLO.
	DIR_FstClusHI uint16
	// Last time when any change is made to the file (typically on closeing).
	DIR_WrtTime uint16
	// Last data when any change is made to the file (typically on closeing).
	DIR_WrtDate uint16
	// Lower part of cluster number. When the file size is zero, no cluster is assigned and this item must be zero. Always an valid value if it is a directory.
	DIR_FstClusLO uint16
	// Size of the file in unit of byte. Not used when it is a directroy and the value must be always zero.
	DIR_FileSize uint32
}

type fatTime struct {
	date  uint16
	time  uint16
	tenth uint8
}

func asFatTime(t time.Time) *fatTime {
	return &fatTime{
		// FAT16 date format: Bits 15-9: Year (0 = 1980), Bits 8-5: Month (1-12), Bits 4-0: Day (1-31)
		date: uint16(((t.Year() - 1980) << 9) | (int(t.Month()) << 5) | t.Day()),
		// FAT16 time format: Bits 15-11: Hours (0-23), Bits 10-5: Minutes (0-59), Bits 4-0: Seconds/2 (0-29)
		time: uint16((t.Hour() << 11) | (t.Minute() << 5) | (t.Second() / 2)),
		// FAT16 CrtTimeTenth: Sub-second information in 10ms units (0-199)
		tenth: uint8(t.Nanosecond() / 1e7),
	}
}

func (d *Entry) names() []string {
	names := []string{d.ShortName()}
	if d.longFileName != "" {
		names = append(names, d.longFileName)
	}

	return names
}

func (d *Entry) ShortName() string {
	nameBytes := bytes.Clone(d.DIR_Name[:])

	if d.IsVolumeId() {
		return string(bytes.TrimRight(nameBytes[:11], " "))
	}

	bdyBytes := bytes.TrimRight(nameBytes[:8], " ")
	extBytes := bytes.TrimRight(bdyBytes[8:11], " ")

	if d.DIR_NTRes&0x08 == 0x08 {
		bdyBytes = bytes.ToLower(bdyBytes)
	}

	if len(extBytes) == 0 {
		return string(bdyBytes)
	}

	if d.DIR_NTRes&0x10 == 0x10 {
		extBytes = bytes.ToLower(extBytes)
	}

	return string(bdyBytes) + "." + string(extBytes)
}

func (d *Entry) LongName() string {
	if d.longFileName != "" {
		return d.longFileName
	}

	return d.ShortName()
}

func (d *fatDirectoryEntry) IsReadOnly() bool {
	return d.DIR_Attr&0x01 == 0x01
}

func (d *fatDirectoryEntry) IsHidden() bool {
	return d.DIR_Attr&0x02 == 0x02
}

func (d *fatDirectoryEntry) IsSystem() bool {
	return d.DIR_Attr&0x04 == 0x04
}

func (d *fatDirectoryEntry) IsVolumeId() bool {
	return d.DIR_Attr&0x08 == 0x08
}

func (d *fatDirectoryEntry) IsDir() bool {
	return d.DIR_Attr&0x10 == 0x10
}

func (d *fatDirectoryEntry) IsArchive() bool {
	return d.DIR_Attr&0x20 == 0x20
}

func (lfn *fatLongFileName) toBytes() [32]byte {
	var data [32]byte
	data[0] = lfn.LDIR_Ord
	data[11] = lfn.LDIR_Attr
	data[12] = lfn.LDIR_Type
	data[13] = lfn.LDIR_Chksum
	binary.LittleEndian.PutUint16(data[26:28], lfn.LDIR_FstClusLO)
	copy(data[1:11], lfn.LDIR_Name1[:])
	copy(data[14:26], lfn.LDIR_Name2[:])
	copy(data[28:32], lfn.LDIR_Name3[:])
	return data
}

func (d *fatDirectoryEntry) toBytes() [32]byte {
	var data [32]byte

	copy(data[:], d.DIR_Name[:])
	for i := len(d.DIR_Name); i < 11; i++ {
		data[i] = 0x20
	}
	data[11] = d.DIR_Attr
	data[12] = d.DIR_NTRes
	data[13] = d.DIR_CrtTimeTenth
	binary.LittleEndian.PutUint16(data[14:16], d.DIR_CrtTime)
	binary.LittleEndian.PutUint16(data[16:18], d.DIR_CrtDate)
	binary.LittleEndian.PutUint16(data[18:20], d.DIR_LstAccDate)
	binary.LittleEndian.PutUint16(data[20:22], d.DIR_FstClusHI)
	binary.LittleEndian.PutUint16(data[22:24], d.DIR_WrtTime)
	binary.LittleEndian.PutUint16(data[24:26], d.DIR_WrtDate)
	binary.LittleEndian.PutUint16(data[26:28], d.DIR_FstClusLO)
	binary.LittleEndian.PutUint32(data[28:32], d.DIR_FileSize)

	return data
}

func fatDirectoryEntryFromBytes(data []byte) fatDirectoryEntry {
	fatEntry := fatDirectoryEntry{
		DIR_Attr:         data[11],
		DIR_NTRes:        data[12],
		DIR_CrtTimeTenth: data[13],
		DIR_CrtTime:      binary.LittleEndian.Uint16(data[14:16]),
		DIR_CrtDate:      binary.LittleEndian.Uint16(data[16:18]),
		DIR_LstAccDate:   binary.LittleEndian.Uint16(data[18:20]),
		DIR_FstClusHI:    binary.LittleEndian.Uint16(data[20:22]),
		DIR_WrtTime:      binary.LittleEndian.Uint16(data[22:24]),
		DIR_WrtDate:      binary.LittleEndian.Uint16(data[24:26]),
		DIR_FstClusLO:    binary.LittleEndian.Uint16(data[26:28]),
		DIR_FileSize:     binary.LittleEndian.Uint32(data[28:32]),
	}
	copy(fatEntry.DIR_Name[:], data[:11])

	return fatEntry
}

func (d *fatDirectoryEntry) clone() fatDirectoryEntry {
	data := d.toBytes()
	return fatDirectoryEntryFromBytes(data[:])
}

func (d *Entry) clone() Entry {
	return Entry{
		fatDirectoryEntry: d.fatDirectoryEntry.clone(),
		longFileName:      d.longFileName,
	}
}

func (d *Entry) clusterNumber() uint16 {
	// d.DIR_FstClusHI unused in FAT16 and always zero
	return d.DIR_FstClusLO
}

func (d *Entry) setCluster(cluster uint16) {
	d.DIR_FstClusLO = cluster
}

// http://elm-chan.org/docs/fat_e.html#name_conversion
// returns (shortFileName, shortFileNameBytes, NTRes value)
func (fs *FileSystem) createShortName(desiredName string, siblingEntries []Entry) (string, [11]byte, uint8) {
	lossy := false

	// 1. convert to upper
	name := strings.ToUpper(desiredName)

	// 2. remove any space
	if strings.Contains(name, " ") {
		lossy = true
		name = strings.ReplaceAll(name, " ", "")
	}

	// 3. remove leading dots
	for strings.HasPrefix(name, ".") {
		lossy = true
		name = strings.TrimPrefix(name, ".")
	}

	// 4. remove all but last dot
	dotCount := strings.Count(name, ".")
	if dotCount > 1 {
		lossy = true
		lastDot := strings.LastIndex(name, ".")
		first := name[:lastDot]
		last := name[lastDot:]
		name = strings.ReplaceAll(first, ".", "") + last
	}

	// 5. remove disallowed ascii chars
	name = strings.Map(func(r rune) rune {
		if r < 0x20 {
			lossy = true
			return '_'
		}

		switch r {
		case '"', '*', '+', ',', '/', ':', ';', '<', '=', '>', '?', '[', '\\', ']', '|':
			lossy = true
			return '_'
		}

		return r
	}, name)

	// 6. convert unicode to ansi/oem code (and replace with random 4-digit hex if empty after that)
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7E {
			lossy = true
			return -1
		}
		return r
	}, name)
	if len(name) == 0 || strings.HasPrefix(name, ".") {
		name = fmt.Sprintf("%04X%s", (*fs.randIntn)(0x10000), name)
	}

	// 7. truncate body and extension to 8 and 3 bytes (if truncation occurs, set lossy)
	var body, ext string
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		body = parts[0]
		ext = parts[1]
		if len(body) > 8 {
			lossy = true
			body = body[:8]
		}
		if len(ext) > 3 {
			lossy = true
			ext = ext[:3]
		}
	} else {
		body = name
		if len(body) > 8 {
			lossy = true
			body = body[:8]
		}
	}

	var finalName string
	if ext == "" {
		finalName = body
	} else {
		finalName = body + "." + ext
	}

	make83 := func(body, ext string) [11]byte {
		value := [11]byte{}
		copy(value[:8], body)
		copy(value[8:11], ext)
		for i := len(body); i < 8; i++ {
			value[i] = 0x20
		}
		for i := len(ext); i < 3; i++ {
			value[i+8] = 0x20
		}

		return value
	}

	existingNames := utils.MapSlice(siblingEntries, func(entry Entry) string { return entry.ShortName() })
	if !lossy && !slices.Contains(existingNames, finalName) {
		parts := strings.SplitN(desiredName, ".", 2)
		ntRes := uint8(0)
		if parts[0] == strings.ToLower(body) {
			ntRes |= 0x08
		}
		if ext != "" && parts[1] == strings.ToLower(ext) {
			ntRes |= 0x10
		}

		return finalName, make83(body, ext), ntRes
	}

	var asBytes [11]byte
	n := 1
	l := len(body)
	for {
		digitCount := int(math.Log10(float64(n))) + 1
		newBody := fmt.Sprintf("%s~%d", body[:min(l, 8-digitCount-1)], n)
		asBytes = make83(newBody, ext)

		if ext == "" {
			finalName = newBody
		} else {
			finalName = fmt.Sprintf("%s.%s", newBody, ext)
		}
		if !slices.Contains(existingNames, finalName) {
			break
		}

		n++
	}

	return finalName, asBytes, 0
}

func (fs *FileSystem) createShortNameBytes(desiredName string, siblingEntries []Entry) ([11]byte, uint8) {
	_, sfnBytes, ntRes := fs.createShortName(desiredName, siblingEntries)
	return sfnBytes, ntRes
}

type readDirResult struct {
	entries []Entry
	bytes   []byte
	cluster *uint16
}

func (fs *FileSystem) readDir(path string, mkdir bool) (*readDirResult, error) {
	parts, err := fs.splitPath(path)
	if err != nil {
		return nil, err
	}

	var currentEntryCluster *uint16 = nil
	currentBytes, err := fs.getRootDirectoryBytes()
	if err != nil {
		return nil, fmt.Errorf("could not read root directory bytes: %w", err)
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		currentEntries, err := readDirectoryEntries(currentBytes)
		if err != nil {
			return nil, fmt.Errorf("could not read directory entries: %w", err)
		}

		found := false
		for _, entry := range currentEntries {
			if !slices.ContainsFunc(entry.names(), func(name string) bool { return strings.EqualFold(name, part) }) {
				continue
			}

			if entry.IsDir() {
				clusterNumber := entry.clusterNumber()
				currentBytes, err = fs.getClusterChainBytes(clusterNumber)
				if err != nil {
					return nil, fmt.Errorf("could not read directory bytes: %w", err)
				}

				currentEntryCluster = &clusterNumber
				found = true
				break
			} else {
				return nil, fmt.Errorf("%s is not a directory", path)
			}
		}

		if !found {
			if mkdir {
				nRequired := fs.numDirectoryEntriesRequired(part)
				freeIndex, newDirectoryBytes, err := fs.getAvailableDirectoryEntry(nRequired, currentEntryCluster, currentBytes)
				if err != nil {
					return nil, err
				}

				newDirectoryCluster, err := fs.allocateClusterChain(directoryEntrySize * 2)
				if err != nil {
					return nil, err
				}

				newDirectoryBytes, err = fs.writeDirectoryEntry(part, newDirectoryCluster, newDirectoryBytes, currentEntries, currentEntryCluster, freeIndex)
				if err != nil {
					return nil, fmt.Errorf("could not write directory entry: %w", err)
				}

				currentBytes = newDirectoryBytes
				currentEntryCluster = &newDirectoryCluster
			} else {
				return nil, fmt.Errorf("no such file or directory %s", path)
			}
		}
	}

	entries, err := readDirectoryEntries(currentBytes)
	if err != nil {
		return nil, fmt.Errorf("could not read directory entries: %w", err)
	}

	return &readDirResult{
		entries: entries,
		bytes:   currentBytes,
		cluster: currentEntryCluster,
	}, nil
}
