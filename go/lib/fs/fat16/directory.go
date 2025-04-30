package fat16

import (
	"encoding/binary"
	"fmt"
	"strings"
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

func (d *DirectoryEntry) names() []string {
	names := []string{d.ShortName()}
	if d.longFileName != "" {
		names = append(names, d.longFileName)
	}

	return names
}

// TODO: casing on short names doesn't seem correct - may need to read flags
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

func (d *DirectoryEntry) clusterNumber() uint16 {
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

func (fs *FileSystem) getDirectoryBytes(startCluster uint16) ([]byte, error) {
	bytesPerCluster := int64(fs.bytesPerSector * fs.sectorsPerCluster)

	var bytes []byte
	currentCluster := startCluster
	for {
		clusterSector := (fs.dataSectorStart + uint32(currentCluster-2)*fs.sectorsPerCluster)
		clusterBytes := make([]byte, bytesPerCluster)
		_, err := fs.file.ReadAt(clusterBytes, int64(clusterSector*fs.bytesPerSector))
		if err != nil {
			return nil, fmt.Errorf("failed to read cluster data: %w", err)
		}

		bytes = append(bytes, clusterBytes...)
		nextCluster := fs.table.clusters[currentCluster]

		if nextCluster >= eoc {
			break
		}

		currentCluster = nextCluster
	}

	return bytes, nil
}

func (fs *FileSystem) readDirectoryEntries(b []byte) ([]DirectoryEntry, error) {
	entries := make([]DirectoryEntry, 0)

	longFileName := ""
	for i := 0; i < len(b); i += directoryEntrySize {
		if b[i] == 0x00 {
			break
		}

		if b[i] == 0xE5 {
			continue
		}

		if b[i+11] == 0x0F {
			lfn := fatLongFileName{
				LDIR_Ord:       b[i],
				LDIR_Attr:      b[i+11],
				LDIR_Type:      b[i+12],
				LDIR_Chksum:    b[i+13],
				LDIR_FstClusLO: binary.LittleEndian.Uint16(b[i+26 : i+28]),
			}
			copy(lfn.LDIR_Name1[:], b[i+1:i+11])
			copy(lfn.LDIR_Name2[:], b[i+14:i+26])
			copy(lfn.LDIR_Name3[:], b[i+28:i+32])

			// Extract the actual name parts and prepend them (since LFN entries are stored in reverse order)
			namePart := lfn.extractNamePart()
			if lfn.LDIR_Ord&0x40 != 0 { // Last entry
				longFileName = ""
			}
			longFileName = namePart + longFileName
			continue
		}

		fatEntry := fatDirectoryEntry{
			DIR_Name:         string(b[i : i+11]),
			DIR_Attr:         b[i+11],
			DIR_NTRes:        b[i+12],
			DIR_CrtTimeTenth: b[i+13],
			DIR_CrtTime:      binary.LittleEndian.Uint16(b[i+14 : i+16]),
			DIR_CrtDate:      binary.LittleEndian.Uint16(b[i+16 : i+18]),
			DIR_LstAccDate:   binary.LittleEndian.Uint16(b[i+18 : i+20]),
			DIR_FstClusHI:    binary.LittleEndian.Uint16(b[i+20 : i+22]),
			DIR_WrtTime:      binary.LittleEndian.Uint16(b[i+22 : i+24]),
			DIR_WrtDate:      binary.LittleEndian.Uint16(b[i+24 : i+26]),
			DIR_FstClusLO:    binary.LittleEndian.Uint16(b[i+26 : i+28]),
			DIR_FileSize:     binary.LittleEndian.Uint32(b[i+28 : i+32]),
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
		runeValue := binary.LittleEndian.Uint16(lfnBytes[i : i+2])
		if runeValue == 0x0000 {
			break
		}
		name.WriteRune(rune(runeValue))
	}

	return name.String()
}
