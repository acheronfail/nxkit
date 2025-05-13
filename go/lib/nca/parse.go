package nca

import (
	"encoding/binary"
	"fmt"
)

const (
	magicNCA3              uint32 = 0x3341434E
	pfs0ExefsHashBlockSize uint32 = 0x10000
	ncaSectionEntrySize           = 16
	ncaFsHeaderSize               = 512
)

type ncaSectionEntry struct {
	MediaStartOffset uint32
	MediaEndOffset   uint32
	_                [8]byte
}

type SectionFsType uint8

const (
	FsTypeRomfs SectionFsType = 0
	FsTypePfs0  SectionFsType = 1
)

type SectionHashType uint8

const (
	HashTypePfs0  SectionHashType = 2
	HashTypeRomfs SectionHashType = 3
)

type SectionCryptType uint8

const (
	CryptNone SectionCryptType = 1
	CryptXts  SectionCryptType = 2
	CryptCtr  SectionCryptType = 3
	CryptBktr SectionCryptType = 4
)

type ncaFsHeader struct {
	Version    uint16
	FsType     uint8 // SectionFsType
	HashType   uint8 // SectionHashType
	CryptType  uint8 // SectionCryptType
	_          [3]byte
	Superblock [0x138]byte // FS-specific superblock, size = 0x138
	SectionCtr [8]byte
	_          [0xB8]byte
}

type SdkVersion struct {
	Revision uint8
	Micro    uint8
	Minor    uint8
	Major    uint8
}

type ncaHeader struct {
	FixedKeySig    [0x100]byte // RSA-PSS signature over header with fixed key
	NpdmKeySig     [0x100]byte // RSA-PSS signature over header with key in NPDM
	magic          uint32
	Distribution   uint8 // System vs gamecard
	ContentType    uint8
	CryptoType     uint8  // Which keyblob (field 1)
	KaekInd        uint8  // Which kaek index?
	NcaSize        uint64 // Entire archive size
	TitleID        uint64
	_              [4]byte
	SdkVersion     uint32 // This could also be accessed as SdkVersionParts
	CryptoType2    uint8  // Which keyblob (field 2)
	_              [0xF]byte
	RightsID       [0x10]byte         // Rights ID (for titlekey crypto)
	SectionEntries [4]ncaSectionEntry // Section entry metadata
	SectionHashes  [4][0x20]byte      // SHA-256 hashes for each section header
	EncryptedKeys  [4][0x10]byte      // Encrypted key area
	_              [0xC0]byte
	FsHeaders      [4]ncaFsHeader // FS section headers
}

func (h *ncaHeader) GetSdkVersionParts() SdkVersion {
	return SdkVersion{
		Revision: uint8(h.SdkVersion),
		Micro:    uint8(h.SdkVersion >> 8),
		Minor:    uint8(h.SdkVersion >> 16),
		Major:    uint8(h.SdkVersion >> 24),
	}
}

type Nca struct {
	header ncaHeader
}

func NewNcaFromBytes(data []byte) (*Nca, error) {
	// TODO: need to decrypt here before parsing

	header := ncaHeader{
		magic:          binary.LittleEndian.Uint32(data[0x200:0x204]),
		Distribution:   data[0x204],
		ContentType:    data[0x205],
		CryptoType:     data[0x206],
		KaekInd:        data[0x207],
		NcaSize:        binary.LittleEndian.Uint64(data[0x208:0x210]),
		TitleID:        binary.LittleEndian.Uint64(data[0x210:0x218]),
		SdkVersion:     binary.LittleEndian.Uint32(data[0x21b:0x220]),
		CryptoType2:    data[0x220],
		SectionEntries: parseSectionEntries(data[0x240:0x280]),
		FsHeaders:      parseFsHeaders(data[0x400:0xc00]),
	}
	copy(header.FixedKeySig[:], data[0:0x100])
	copy(header.NpdmKeySig[:], data[0x100:0x200])
	copy(header.RightsID[:], data[0x230:0x240])
	copy(header.SectionHashes[0][:], data[0x280:0x2a0])
	copy(header.SectionHashes[1][:], data[0x2a0:0x2c0])
	copy(header.SectionHashes[2][:], data[0x2c0:0x2e0])
	copy(header.SectionHashes[3][:], data[0x2e0:0x300])
	copy(header.EncryptedKeys[0][:], data[0x300:0x310])
	copy(header.EncryptedKeys[1][:], data[0x310:0x320])
	copy(header.EncryptedKeys[2][:], data[0x320:0x330])
	copy(header.EncryptedKeys[3][:], data[0x330:0x340])

	if header.magic != magicNCA3 {
		return nil, fmt.Errorf("unrecognised magic value; expected %x got %x", magicNCA3, header.magic)
	}

	return &Nca{header: header}, nil
}

func parseSectionEntries(data []byte) [4]ncaSectionEntry {
	var sectionEntries [4]ncaSectionEntry
	for i := range 4 {
		start := i * ncaSectionEntrySize
		sectionEntries[i] = ncaSectionEntry{
			MediaStartOffset: binary.LittleEndian.Uint32(data[start : start+4]),
			MediaEndOffset:   binary.LittleEndian.Uint32(data[start+4 : start+8]),
		}
	}

	return sectionEntries
}

func parseFsHeaders(data []byte) [4]ncaFsHeader {
	var fsHeaders [4]ncaFsHeader
	for i := range 4 {
		start := i * ncaFsHeaderSize
		fsHeaders[i] = ncaFsHeader{
			Version:   binary.LittleEndian.Uint16(data[start : start+2]),
			FsType:    data[start+2],
			HashType:  data[start+3],
			CryptType: data[start+4],
		}
		copy(fsHeaders[i].Superblock[:], data[start+8:start+0x140])
		copy(fsHeaders[i].SectionCtr[:], data[start+0x140:start+0x148])
	}

	return fsHeaders
}
