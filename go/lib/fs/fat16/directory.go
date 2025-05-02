package fat16

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	longFileNameUtf16Length = 13
)

type DirectoryEntry struct {
	fatDirectoryEntry
	longFileName string
}

type fatDirectoryEntry struct {
	// Short file name (SFN) of the object.
	DIR_Name string
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

func asFatTime(t time.Time) (uint16, uint16, uint8) {
	// FAT16 time format: Bits 15-11: Hours (0-23), Bits 10-5: Minutes (0-59), Bits 4-0: Seconds/2 (0-29)
	fatTime := uint16((t.Hour() << 11) | (t.Minute() << 5) | (t.Second() / 2))

	// FAT16 date format: Bits 15-9: Year (0 = 1980), Bits 8-5: Month (1-12), Bits 4-0: Day (1-31)
	fatDate := uint16(((t.Year() - 1980) << 9) | (int(t.Month()) << 5) | t.Day())

	// FAT16 CrtTimeTenth: Sub-second information in 10ms units (0-199)
	fatCrtTimeTenth := uint8(t.Nanosecond() / 1e7) // Convert nanoseconds to 10ms units

	return fatTime, fatDate, fatCrtTimeTenth
}

func (d *DirectoryEntry) names() []string {
	names := []string{d.ShortName()}
	if d.longFileName != "" {
		names = append(names, d.longFileName)
	}

	return names
}

func (d *DirectoryEntry) ShortName() string {
	if d.IsVolumeId() {
		return strings.TrimRight(d.DIR_Name[:11], " ")
	}

	sfn := strings.TrimRight(d.DIR_Name[:8], " ")
	ext := strings.TrimRight(d.DIR_Name[8:11], " ")

	if d.DIR_NTRes&0x08 == 0x08 {
		sfn = strings.ToLower(sfn)
	}
	if d.DIR_NTRes&0x10 == 0x10 {
		ext = strings.ToLower(ext)
	}

	if len(ext) == 0 {
		return sfn
	}

	return sfn + "." + ext
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

func (d *fatDirectoryEntry) toBytes() []byte {
	data := make([]byte, 32)

	copy(data, d.DIR_Name)
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

func (lfn *fatLongFileName) toBytes() []byte {
	data := make([]byte, 32)

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

func (d *DirectoryEntry) clusterNumber() uint16 {
	// d.DIR_FstClusHI unused in FAT16 and always zero
	return d.DIR_FstClusLO
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

func (fs *FileSystem) getDirectoryClusterChain(startCluster uint16) ([]uint16, error) {
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

func (fs *FileSystem) getDirectoryBytes(startCluster uint16) ([]byte, error) {
	clusterChain, err := fs.getDirectoryClusterChain(startCluster)
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

func (fs *FileSystem) findAvailableDirectoryEntryPos(dirBytes []byte, desiredCount int) (int, bool) {
	freeEntry := byte(0x00)
	deletedEntry := byte(0xE5)

	freeSlotCount := 0
	for i := 0; i < len(dirBytes); i += directoryEntrySize {
		if dirBytes[i] == freeEntry || dirBytes[i] == deletedEntry {
			freeSlotCount++
		} else {
			freeSlotCount = 0
		}

		if freeSlotCount == desiredCount {
			return i, true
		}
	}

	return 0, false
}

func (fs *FileSystem) readDirectoryEntries(dirBytes []byte) ([]DirectoryEntry, error) {
	entries := make([]DirectoryEntry, 0)

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
			fmt.Println("parse", lfn, namePart)
			if lfn.LDIR_Ord&0x40 != 0 {
				longFileName = ""
			}
			longFileName = namePart + longFileName
			continue
		}

		fatEntry := fatDirectoryEntry{
			DIR_Name:         string(dirBytes[i : i+11]),
			DIR_Attr:         dirBytes[i+11],
			DIR_NTRes:        dirBytes[i+12],
			DIR_CrtTimeTenth: dirBytes[i+13],
			DIR_CrtTime:      binary.LittleEndian.Uint16(dirBytes[i+14 : i+16]),
			DIR_CrtDate:      binary.LittleEndian.Uint16(dirBytes[i+16 : i+18]),
			DIR_LstAccDate:   binary.LittleEndian.Uint16(dirBytes[i+18 : i+20]),
			DIR_FstClusHI:    binary.LittleEndian.Uint16(dirBytes[i+20 : i+22]),
			DIR_WrtTime:      binary.LittleEndian.Uint16(dirBytes[i+22 : i+24]),
			DIR_WrtDate:      binary.LittleEndian.Uint16(dirBytes[i+24 : i+26]),
			DIR_FstClusLO:    binary.LittleEndian.Uint16(dirBytes[i+26 : i+28]),
			DIR_FileSize:     binary.LittleEndian.Uint32(dirBytes[i+28 : i+32]),
		}

		entries = append(entries, DirectoryEntry{fatDirectoryEntry: fatEntry, longFileName: longFileName})
		longFileName = ""
	}

	return entries, nil
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

func (fs *FileSystem) dirEntrySlotsRequired(name string) int {
	length := len(name)
	if length <= 8 {
		return 1
	}

	return 1 + ((length + longFileNameUtf16Length - 1) / longFileNameUtf16Length)
}

func calculateShortNameChecksum(shortName string) uint8 {
	var sum uint8 = 0
	for i := 0; i < 11; i++ {
		sum = ((sum & 1) << 7) + (sum >> 1) + uint8(shortName[i])
	}
	return sum
}

func (fs *FileSystem) createLongFileNameEntries(longName string, checksum uint8) []fatLongFileName {
	if len(longName) <= 8 {
		return nil
	}

	nameRunes := []rune(longName)
	numEntries := (len(nameRunes) + longFileNameUtf16Length - 1) / longFileNameUtf16Length
	lfnEntries := make([]fatLongFileName, numEntries)

	// Process the name parts in reverse order (last part first)
	for i := 0; i < numEntries; i++ {
		// Calculate the start and end indices for this part
		startIdx := len(nameRunes) - ((i + 1) * longFileNameUtf16Length)
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := len(nameRunes) - (i * longFileNameUtf16Length)

		// Get the name part for this entry
		namePart := nameRunes[startIdx:endIdx]

		// Create UTF-16 bytes for this part
		lfnBytes := make([]byte, longFileNameUtf16Length*2)

		// Fill with 0xFF as padding
		for j := 0; j < len(lfnBytes); j++ {
			lfnBytes[j] = 0xFF
		}

		// Convert name part to UTF-16
		for j, char := range namePart {
			binary.LittleEndian.PutUint16(lfnBytes[j*2:j*2+2], uint16(char))
		}

		// If this is the last part of the name (might be shorter), null terminate
		if i == 0 {
			if len(namePart) < longFileNameUtf16Length {
				binary.LittleEndian.PutUint16(lfnBytes[len(namePart)*2:len(namePart)*2+2], 0x0000)
			}
		}

		// Calculate sequence number
		seqNum := uint8(i + 1)
		if i == numEntries-1 {
			seqNum |= 0x40 // Set LAST_LONG_ENTRY flag for the first entry
		}

		lfnEntry := fatLongFileName{
			LDIR_Ord:       seqNum,
			LDIR_Attr:      0x0F,
			LDIR_Type:      0x00,
			LDIR_Chksum:    checksum,
			LDIR_FstClusLO: 0,
		}

		// Copy the name parts into the entry
		copy(lfnEntry.LDIR_Name1[:], lfnBytes[:10])
		copy(lfnEntry.LDIR_Name2[:], lfnBytes[10:22])
		copy(lfnEntry.LDIR_Name3[:], lfnBytes[22:26])

		lfnEntries[numEntries-1-i] = lfnEntry
	}

	return lfnEntries
}

// TODO: support writing long file names
func (fs *FileSystem) writeDirectoryEntry(
	newDirName string,
	newDirCluster uint16,
	parentDirBytes []byte,
	parentDirCluster *uint16,
	atParentDirByteIndex int,
) ([]byte, error) {
	time, date, tenth := asFatTime(time.Now())

	// create new directory entry
	// TODO: create short name without conflicts
	sfn := make([]byte, 11)
	copy(sfn, newDirName)
	newFatDirEntry := fatDirectoryEntry{
		DIR_Name:         string(sfn),
		DIR_Attr:         0x10,
		DIR_NTRes:        0x00,
		DIR_CrtTimeTenth: tenth,
		DIR_CrtTime:      time,
		DIR_CrtDate:      date,
		DIR_LstAccDate:   date,
		DIR_FstClusHI:    0,
		DIR_WrtTime:      time,
		DIR_WrtDate:      date,
		DIR_FstClusLO:    newDirCluster,
		DIR_FileSize:     0,
	}

	// collect all the bytes for the new directory entry, including long file name entries
	newDirEntriesBytes := make([][]byte, 0)
	for _, lfnEntry := range fs.createLongFileNameEntries(newDirName, calculateShortNameChecksum(string(sfn))) {
		newDirEntriesBytes = append(newDirEntriesBytes, lfnEntry.toBytes())
	}
	newDirEntriesBytes = append(newDirEntriesBytes, newFatDirEntry.toBytes())

	// write new directory into parent's directory bytes
	fmt.Println(newDirName)
	for i, entryBytes := range newDirEntriesBytes {
		start := atParentDirByteIndex + (directoryEntrySize * i)
		end := start + directoryEntrySize
		fmt.Println("writing", atParentDirByteIndex, len(entryBytes), start, end)
		copy(parentDirBytes[start:end], entryBytes)
		fmt.Println("wrote", parentDirBytes[start:end], parentDirBytes[start+11])
	}

	// write back to disk
	if parentDirCluster == nil {
		// if root, just write it all back since it's in the dedicated root directory area
		_, err := fs.file.WriteAt(parentDirBytes, int64(fs.rootDirectorySectorStart*fs.bytesPerSector))
		if err != nil {
			return nil, err
		}
	} else {
		// if not root, then write the entire parent directory back to disk, along its cluster chain
		clusterChain, err := fs.getDirectoryClusterChain(*parentDirCluster)
		if err != nil {
			return nil, err
		}

		// TODONICE: likely just need to write the last cluster rather than re-writing all of them?
		for i, cluster := range clusterChain {
			toWrite := make([]byte, fs.bytesPerCluster)
			copy(toWrite, parentDirBytes[int64(i)*fs.bytesPerCluster:int64(i+1)*fs.bytesPerCluster])
			_, err := fs.file.WriteAt(toWrite, int64(fs.clusterToSector(cluster)*fs.bytesPerSector))
			if err != nil {
				return nil, err
			}
		}
	}

	// create special directory entries
	dotDirEntry := fatDirectoryEntry{
		DIR_Name:         ".",
		DIR_Attr:         0x10,
		DIR_CrtTimeTenth: tenth,
		DIR_CrtTime:      time,
		DIR_CrtDate:      date,
		DIR_LstAccDate:   date,
		DIR_FstClusHI:    0,
		DIR_WrtTime:      time,
		DIR_WrtDate:      date,
		DIR_FstClusLO:    newDirCluster,
		DIR_FileSize:     0,
	}
	dotDotDirEntry := fatDirectoryEntry{
		DIR_Name:         "..",
		DIR_Attr:         0x10,
		DIR_CrtTimeTenth: tenth,
		DIR_CrtTime:      time,
		DIR_CrtDate:      date,
		DIR_LstAccDate:   date,
		DIR_FstClusHI:    0,
		DIR_WrtTime:      time,
		DIR_WrtDate:      date,
		DIR_FstClusLO:    0,
		DIR_FileSize:     0,
	}
	if parentDirCluster != nil {
		dotDotDirEntry.DIR_FstClusLO = *parentDirCluster
	}

	// create directory bytes for the new directory
	newDirectoryDataBytes := make([]byte, fs.bytesPerCluster)
	copy(newDirectoryDataBytes[:directoryEntrySize], dotDirEntry.toBytes())
	copy(newDirectoryDataBytes[directoryEntrySize:], dotDotDirEntry.toBytes())

	// write . and .. into new cluster in data region
	_, err := fs.file.WriteAt(newDirectoryDataBytes, int64(fs.clusterToSector(newDirCluster)*fs.bytesPerSector))
	if err != nil {
		return nil, err
	}

	return newDirectoryDataBytes, nil
}
