package fat16

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

type DirectoryEntry struct {
	fatDirectoryEntry
	longFileName string
}

// IsArchive implements fs.Stat.
func (d *DirectoryEntry) IsArchive() bool {
	return d.fatDirectoryEntry.IsArchive()
}

// IsDir implements fs.Stat.
func (d *DirectoryEntry) IsDir() bool {
	return d.fatDirectoryEntry.IsDir()
}

// IsFile implements fs.Stat.
func (d *DirectoryEntry) IsFile() bool {
	return !d.fatDirectoryEntry.IsDir()
}

// IsHidden implements fs.Stat.
func (d *DirectoryEntry) IsHidden() bool {
	return d.fatDirectoryEntry.IsHidden()
}

// IsReadOnly implements fs.Stat.
func (d *DirectoryEntry) IsReadOnly() bool {
	return d.fatDirectoryEntry.IsReadOnly()
}

// IsSystem implements fs.Stat.
func (d *DirectoryEntry) IsSystem() bool {
	return d.fatDirectoryEntry.IsSystem()
}

// IsVolumeId implements fs.Stat.
func (d *DirectoryEntry) IsVolumeId() bool {
	return d.fatDirectoryEntry.IsVolumeId()
}

// Size implements fs.Stat.
func (d *DirectoryEntry) Size() int64 {
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

type fatLongFileName struct {
	// Sequence number (1-20) to identify where this entry is in the sequence of LFN entries to compose an LFN.
	// One indicates the top part of the LFN and any value with LAST_LONG_ENTRY flag (0x40) indicates the last part of the LFN.
	LDIR_Ord uint8
	// Part of LFN from 1st character to 5th character.
	LDIR_Name1 [10]byte
	// LFN attribute. Always ATTR_LONG_NAME and it indicates this is an LFN entry.
	LDIR_Attr uint8
	// Must be zero.
	LDIR_Type uint8
	// Checksum of the SFN entry associated with this entry.
	LDIR_Chksum uint8
	// Part of LFN from 6th character to 11th character.
	LDIR_Name2 [12]byte
	// Must be zero to avoid any wrong repair by old disk utility.
	LDIR_FstClusLO uint16
	// Part of LFN from 12th character to 13th character.
	LDIR_Name3 [4]byte
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

func (d *DirectoryEntry) names() []string {
	names := []string{d.ShortName()}
	if d.longFileName != "" {
		names = append(names, d.longFileName)
	}

	return names
}

func (d *DirectoryEntry) ShortName() string {
	nameBytes := bytes.Clone(d.DIR_Name[:])

	if d.IsVolumeId() {
		return string(bytes.TrimRight(nameBytes[:11], " "))
	}

	sfnBytes := bytes.TrimRight(nameBytes[:8], " ")
	extBytes := bytes.TrimRight(sfnBytes[8:11], " ")

	if d.DIR_NTRes&0x08 == 0x08 {
		sfnBytes = bytes.ToLower(sfnBytes)
	}

	if len(extBytes) == 0 {
		return string(sfnBytes)
	}

	if d.DIR_NTRes&0x10 == 0x10 {
		extBytes = bytes.ToLower(extBytes)
	}

	return string(sfnBytes) + "." + string(extBytes)
}

func (d *DirectoryEntry) LongName() string {
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

func (d *DirectoryEntry) clone() DirectoryEntry {
	return DirectoryEntry{
		fatDirectoryEntry: d.fatDirectoryEntry.clone(),
		longFileName:      d.longFileName,
	}
}

func (d *DirectoryEntry) clusterNumber() uint16 {
	// d.DIR_FstClusHI unused in FAT16 and always zero
	return d.DIR_FstClusLO
}

func (d *DirectoryEntry) setCluster(cluster uint16) {
	d.DIR_FstClusLO = cluster
}

func (fs *FileSystem) getRootDirectoryBytes() ([]byte, error) {
	start := fs.rootDirectorySectorStart * fs.bytesPerSector
	rootDirSize := fs.bootSector.BPB_RootEntCnt * directoryEntrySize
	b := make([]byte, rootDirSize)
	_, err := fs.file.ReadAt(b, int64(start))
	if err != nil {
		return nil, fmt.Errorf("could not read root directory bytes: %w", err)
	}

	return b, nil
}

func (fs *FileSystem) getClusterChain(startCluster uint16) ([]uint16, error) {
	var clusters []uint16
	currentCluster := startCluster
	for {
		clusters = append(clusters, currentCluster)
		nextCluster := fs.table.clusters[currentCluster]

		if nextCluster >= eoc {
			break
		}
		if nextCluster < 2 {
			return nil, fmt.Errorf("invalid cluster number: %d", nextCluster)
		}
		if nextCluster > fs.table.maxCluster {
			return nil, fmt.Errorf("cluster number out of range: %d", nextCluster)
		}

		currentCluster = nextCluster
	}

	return clusters, nil
}

func (fs *FileSystem) getClusterChainBytes(startCluster uint16) ([]byte, error) {
	clusterChain, err := fs.getClusterChain(startCluster)
	if err != nil {
		return nil, err
	}

	var bytes []byte
	for _, cluster := range clusterChain {
		clusterBytes := make([]byte, fs.bytesPerCluster)
		fileOffset := int64(fs.clusterToSector(cluster) * fs.bytesPerSector)
		_, err := fs.file.ReadAt(clusterBytes, fileOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to read cluster data: %w", err)
		}

		bytes = append(bytes, clusterBytes...)
	}

	return bytes, nil
}

// http://elm-chan.org/docs/fat_e.html#name_conversion
func (fs *FileSystem) createShortName(desiredName string, siblingEntries []DirectoryEntry) (string, [11]byte, error) {
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

	existingNames := utils.MapSlice(siblingEntries, func(entry DirectoryEntry) string { return entry.ShortName() })
	if !lossy && !slices.Contains(existingNames, finalName) {
		return finalName, make83(body, ext), nil
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

	return finalName, asBytes, nil
}

func (fs *FileSystem) createShortNameBytes(desiredName string, siblingEntries []DirectoryEntry) ([11]byte, error) {
	_, sfnBytes, err := fs.createShortName(desiredName, siblingEntries)
	if err != nil {
		return [11]byte{}, err
	}

	return sfnBytes, nil
}

func (fs *FileSystem) numDirectoryEntriesRequired(dirName string) int {
	if len(dirName) <= 11 {
		return 1
	}

	return (len(dirName) / longFileNameUtf16Length) + 1
}

// TODO: improve this - with tests - to:
//
//	(1) stop scanning when 0x00 is found, and
//	(2) handle issues when near end of `dirBytes`
func (fs *FileSystem) findAvailableDirectoryEntry(dirBytes []byte, numEntries int) (int, bool) {
	count := 0
	for i := 0; i < len(dirBytes); i += directoryEntrySize {
		// 0x00 == free, 0xE5 == deleted
		if dirBytes[i] == 0x00 || dirBytes[i] == 0xE5 {
			count++
		} else {
			count = 0
		}

		if count == numEntries {
			return i - (numEntries-1)*directoryEntrySize, true
		}
	}

	return 0, false
}

func readDirectoryEntries(dirBytes []byte) ([]DirectoryEntry, error) {
	entries := make([]DirectoryEntry, 0)

	var longFileNameSum *uint8
	longFileName := ""
	for i := 0; i < len(dirBytes); i += directoryEntrySize {
		if dirBytes[i] == 0x00 {
			break
		}

		if dirBytes[i] == 0xE5 {
			continue
		}

		if dirBytes[i+11] == 0x0F {
			lfn := fatLongFileName{
				LDIR_Ord:       dirBytes[i],
				LDIR_Attr:      dirBytes[i+11],
				LDIR_Type:      dirBytes[i+12],
				LDIR_Chksum:    dirBytes[i+13],
				LDIR_FstClusLO: binary.LittleEndian.Uint16(dirBytes[i+26 : i+28]),
			}
			copy(lfn.LDIR_Name1[:], dirBytes[i+1:i+11])
			copy(lfn.LDIR_Name2[:], dirBytes[i+14:i+26])
			copy(lfn.LDIR_Name3[:], dirBytes[i+28:i+32])

			// Extract the actual name parts and prepend them (since LFN entries are stored in reverse order)
			namePart := lfn.extractNamePart()

			// this was the first lfn entry, reset lfn
			if lfn.LDIR_Ord&0x40 != 0 {
				longFileName = ""
				longFileNameSum = nil
			}

			// if the checksums didn't match, then discard them
			if longFileNameSum != nil && *longFileNameSum != lfn.LDIR_Chksum {
				continue
			}

			longFileName = namePart + longFileName
			longFileNameSum = &lfn.LDIR_Chksum
			continue
		}

		fatEntry := fatDirectoryEntryFromBytes(dirBytes[i : i+32])
		entry := DirectoryEntry{fatDirectoryEntry: fatEntry}

		// check lfn checksum before applying it
		if longFileNameSum != nil && calculateShortNameChecksum(fatEntry.DIR_Name[:]) == *longFileNameSum {
			entry.longFileName = longFileName
		}

		// reset lfn
		longFileName = ""
		longFileNameSum = nil

		entries = append(entries, entry)
	}

	return entries, nil
}

func (fs *FileSystem) findIndexInParentBytes(ent *DirectoryEntry, parentDirCluster *uint16) (int, []byte, error) {
	parentDirBytes, err := fs.getClusterChainBytes(*parentDirCluster)
	if err != nil {
		return -1, nil, err
	}

	for i := 0; i < len(parentDirBytes); i += directoryEntrySize {
		if parentDirBytes[i] == 0x00 {
			break
		}

		// skip deleted and lfn entries
		if parentDirBytes[i] == 0xE5 || parentDirBytes[i+11] == 0x0F {
			continue
		}

		// check if this is the entry we are looking for
		if bytes.Equal(parentDirBytes[i:i+11], ent.DIR_Name[:]) {
			return i, parentDirBytes, nil
		}
	}

	return -1, nil, fmt.Errorf("entry not found in parent directory")
}

func (lfn *fatLongFileName) extractNamePart() string {
	var name strings.Builder

	lfnBytes := make([]byte, longFileNameUtf16Length*2)
	copy(lfnBytes[0:10], lfn.LDIR_Name1[:])
	copy(lfnBytes[10:22], lfn.LDIR_Name2[:])
	copy(lfnBytes[22:26], lfn.LDIR_Name3[:])

	// Convert the UTF-16 bytes to runes
	for i := 0; i < len(lfnBytes); i += 2 {
		if lfnBytes[i] == 0x00 && lfnBytes[i+1] == 0x00 {
			break
		}
		name.WriteRune(rune(binary.LittleEndian.Uint16(lfnBytes[i : i+2])))
	}

	return name.String()
}

func calculateShortNameChecksum(shortName []byte) uint8 {
	var sum uint8 = 0
	for i := range 11 {
		sum = ((sum & 1) << 7) + (sum >> 1) + uint8(shortName[i])
	}
	return sum
}

func (fs *FileSystem) createLongFileNameEntries(longName string, checksum uint8) []fatLongFileName {
	numEntries := fs.numDirectoryEntriesRequired(longName)
	if numEntries <= 1 {
		return nil
	}

	lfnEntries := make([]fatLongFileName, numEntries)
	max := numEntries - 1
	for i := max; i >= 0; i-- {
		start := i * longFileNameUtf16Length
		end := min((i+1)*longFileNameUtf16Length, len(longName))
		chunk := longName[start:end]

		lfnBytes := make([]byte, longFileNameUtf16Length*2)
		for j := range lfnBytes {
			lfnBytes[j] = 0xFF
		}

		for j, char := range chunk {
			binary.LittleEndian.PutUint16(lfnBytes[j*2:j*2+2], uint16(char))
		}

		// null terminate
		chunkLen := len(chunk)
		if chunkLen < longFileNameUtf16Length {
			binary.LittleEndian.PutUint16(lfnBytes[chunkLen*2:chunkLen*2+2], 0x0000)
		}

		// Calculate sequence number
		seqNum := uint8(i + 1)
		// Set the last entry flag for the last LFN entry
		if i == max {
			seqNum |= 0x40
		}
		lfnEntry := fatLongFileName{
			LDIR_Ord:       seqNum,
			LDIR_Attr:      0x0F,
			LDIR_Type:      0x00,
			LDIR_Chksum:    checksum,
			LDIR_FstClusLO: 0,
		}
		copy(lfnEntry.LDIR_Name1[:], lfnBytes[:10])
		copy(lfnEntry.LDIR_Name2[:], lfnBytes[10:22])
		copy(lfnEntry.LDIR_Name3[:], lfnBytes[22:26])
		lfnEntries[max-i] = lfnEntry
	}

	return lfnEntries
}

func (fs *FileSystem) createNewEntry(
	newEntName string,
	newEntTime *fatTime,
	newEntAttr uint8,
	newEntCluster uint16,
	parentDirEntries []DirectoryEntry,
) (*DirectoryEntry, error) {
	shortNameBytes, err := fs.createShortNameBytes(newEntName, parentDirEntries)
	if err != nil {
		return nil, err
	}

	newFatDirEntry := fatDirectoryEntry{
		DIR_Name:         shortNameBytes,
		DIR_Attr:         newEntAttr,
		DIR_NTRes:        0x00,
		DIR_CrtTimeTenth: newEntTime.tenth,
		DIR_CrtTime:      newEntTime.time,
		DIR_CrtDate:      newEntTime.date,
		DIR_LstAccDate:   newEntTime.date,
		DIR_FstClusHI:    0,
		DIR_WrtTime:      newEntTime.time,
		DIR_WrtDate:      newEntTime.date,
		DIR_FstClusLO:    newEntCluster,
		DIR_FileSize:     0,
	}

	return &DirectoryEntry{fatDirectoryEntry: newFatDirEntry, longFileName: newEntName}, nil
}

func (fs *FileSystem) writeEntryWithLfnToParent(
	entLongName string,
	ent *DirectoryEntry,
	parentDirCluster *uint16,
	parentDirBytes []byte,
	atParentByteIndex int,
) error {
	items := []to32Bytes{}
	for _, lfnEntry := range fs.createLongFileNameEntries(entLongName, calculateShortNameChecksum(ent.fatDirectoryEntry.DIR_Name[:])) {
		items = append(items, &lfnEntry)
	}
	items = append(items, &ent.fatDirectoryEntry)

	return fs.writeEntriesToParent(items, parentDirCluster, parentDirBytes, atParentByteIndex)
}

type to32Bytes interface {
	toBytes() [32]byte
}

func (fs *FileSystem) writeEntriesToParent(
	items []to32Bytes,
	parentDirCluster *uint16,
	parentDirBytes []byte,
	atParentByteIndex int,
) error {
	// write new directory into parent's directory bytes
	for i, item := range items {
		start := atParentByteIndex + (directoryEntrySize * i)
		end := start + directoryEntrySize
		bytesToWrite := item.toBytes()
		copy(parentDirBytes[start:end], bytesToWrite[:])
	}

	// write back to disk
	if parentDirCluster == nil {
		// if root, just write it all back since it's in the dedicated root directory area
		_, err := fs.file.WriteAt(parentDirBytes, int64(fs.rootDirectorySectorStart*fs.bytesPerSector))
		if err != nil {
			return err
		}
	} else {
		// if not root, then write the entire parent directory back to disk, along its cluster chain
		err := fs.writeClusterChain(*parentDirCluster, parentDirBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) writeDirectoryEntry(
	newDirName string,
	newDirCluster uint16,
	parentDirBytes []byte,
	parentDirEntries []DirectoryEntry,
	parentDirCluster *uint16,
	atParentByteIndex int,
) ([]byte, error) {
	t := asFatTime(time.Now())
	newDirEntry, err := fs.createNewEntry(newDirName, t, 0x10, newDirCluster, parentDirEntries)
	if err != nil {
		return nil, err
	}

	err = fs.writeEntryWithLfnToParent(newDirName, newDirEntry, parentDirCluster, parentDirBytes, atParentByteIndex)
	if err != nil {
		return nil, err
	}

	// create special directory entries
	dotDirEntry := fatDirectoryEntry{
		DIR_Name:         [11]byte{'.', 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20},
		DIR_Attr:         0x10,
		DIR_CrtTimeTenth: t.tenth,
		DIR_CrtTime:      t.time,
		DIR_CrtDate:      t.date,
		DIR_LstAccDate:   t.date,
		DIR_FstClusHI:    0,
		DIR_WrtTime:      t.time,
		DIR_WrtDate:      t.date,
		DIR_FstClusLO:    newDirCluster,
		DIR_FileSize:     0,
	}
	dotDotDirEntry := fatDirectoryEntry{
		DIR_Name:         [11]byte{'.', '.', 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20},
		DIR_Attr:         0x10,
		DIR_CrtTimeTenth: t.tenth,
		DIR_CrtTime:      t.time,
		DIR_CrtDate:      t.date,
		DIR_LstAccDate:   t.date,
		DIR_FstClusHI:    0,
		DIR_WrtTime:      t.time,
		DIR_WrtDate:      t.date,
		DIR_FstClusLO:    0,
		DIR_FileSize:     0,
	}
	if parentDirCluster != nil {
		dotDotDirEntry.DIR_FstClusLO = *parentDirCluster
	}

	// create directory bytes for the new directory
	newDirectoryDataBytes := make([]byte, fs.bytesPerCluster)
	dotBytes := dotDirEntry.toBytes()
	dotDotBytes := dotDotDirEntry.toBytes()
	copy(newDirectoryDataBytes[:directoryEntrySize], dotBytes[:])
	copy(newDirectoryDataBytes[directoryEntrySize:], dotDotBytes[:])

	// write . and .. into new cluster in data region
	_, err = fs.file.WriteAt(newDirectoryDataBytes, int64(fs.clusterToSector(newDirCluster)*fs.bytesPerSector))
	if err != nil {
		return nil, err
	}

	return newDirectoryDataBytes, nil
}
