package fat16

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

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

func fatLongFileNameFromBytes(data []byte) fatLongFileName {
	lfn := fatLongFileName{
		LDIR_Ord:       data[0],
		LDIR_Attr:      data[11],
		LDIR_Type:      data[12],
		LDIR_Chksum:    data[13],
		LDIR_FstClusLO: binary.LittleEndian.Uint16(data[26:28]),
	}
	copy(lfn.LDIR_Name1[:], data[1:11])
	copy(lfn.LDIR_Name2[:], data[14:26])
	copy(lfn.LDIR_Name3[:], data[28:32])

	return lfn
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
	numLfnEntries := fs.numDirectoryEntriesRequired(longName) - 1
	if numLfnEntries < 1 {
		return nil
	}

	runes := utf16.Encode([]rune(longName))
	lfnEntries := make([]fatLongFileName, numLfnEntries)
	limit := numLfnEntries - 1
	for i := limit; i >= 0; i-- {
		start := i * longFileNameUtf16Length
		end := min(start+longFileNameUtf16Length, len(runes))
		chunk := runes[start:end]

		// fill with 0xFFFF
		var lfnBytes [26]byte
		for j := 0; j < len(lfnBytes); j += 2 {
			binary.LittleEndian.PutUint16(lfnBytes[j:j+2], 0xFFFF)
		}

		// copy in chunk
		for j, r := range chunk {
			binary.LittleEndian.PutUint16(lfnBytes[j*2:j*2+2], r)
		}

		// null terminate
		chunkLen := len(chunk)
		if chunkLen < longFileNameUtf16Length {
			binary.LittleEndian.PutUint16(lfnBytes[chunkLen*2:], 0x0000)
		}

		// Calculate sequence number
		seqNum := uint8(i + 1)
		// Set the last entry flag for the last LFN entry
		if i == limit {
			seqNum |= 0x40
		}

		lfnEntry := fatLongFileName{
			LDIR_Ord:       seqNum,
			LDIR_Attr:      0x0F,
			LDIR_Type:      0,
			LDIR_Chksum:    checksum,
			LDIR_FstClusLO: 0,
		}
		copy(lfnEntry.LDIR_Name1[:], lfnBytes[0:10])
		copy(lfnEntry.LDIR_Name2[:], lfnBytes[10:22])
		copy(lfnEntry.LDIR_Name3[:], lfnBytes[22:26])
		lfnEntries[limit-i] = lfnEntry
	}

	return lfnEntries
}
