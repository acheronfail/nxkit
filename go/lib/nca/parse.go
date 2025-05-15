package nca

import (
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/xtsn"
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

func (t SectionFsType) String() string {
	switch t {
	case FsTypeRomfs:
		return "ROMFS"
	case FsTypePfs0:
		return "PFS0"
	}

	return "<UNKNOWN>"
}

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
	version    uint16
	fsType     uint8 // SectionFsType
	hashType   uint8 // SectionHashType
	cryptType  uint8 // SectionCryptType
	_          [3]byte
	superblock [0x138]byte // FS-specific superblock
	sectionCtr [8]byte
	_          [0xB8]byte
}

func (fsHeader *ncaFsHeader) SectionCtr(mediaStartOffset uint32) [0x10]byte {
	var ctr [0x10]byte
	offset := uint64(mediaStartOffset*0x200) >> 4
	for i := range 8 {
		ctr[i] = fsHeader.sectionCtr[0x8-i-1]
		ctr[0x10-i-1] = byte(offset & 0xff)
		offset >>= 8
	}

	return ctr
}

type pfs0Superblock struct {
	masterHash      [0x20]byte
	blockSize       uint32
	always2         uint32
	hashTableOffset uint64
	hashTableSize   uint64
	pfs0Offset      uint64
	pfs0Size        uint64
	_               [0xf0]byte
}

func pfs0SuperblockFromBytes(data [0x138]byte) pfs0Superblock {
	sb := pfs0Superblock{
		blockSize:       binary.LittleEndian.Uint32(data[0x20:0x24]),
		always2:         binary.LittleEndian.Uint32(data[0x24:0x28]),
		hashTableOffset: binary.LittleEndian.Uint64(data[0x28:0x30]),
		hashTableSize:   binary.LittleEndian.Uint64(data[0x30:0x38]),
		pfs0Offset:      binary.LittleEndian.Uint64(data[0x38:0x40]),
		pfs0Size:        binary.LittleEndian.Uint64(data[0x40:0x48]),
	}
	copy(sb.masterHash[:], data[:0x20])
	return sb
}

func (fsHeader *ncaFsHeader) FsType() SectionFsType {
	return SectionFsType(fsHeader.fsType)
}

type SdkVersion struct {
	Revision uint8
	Micro    uint8
	Minor    uint8
	Major    uint8
}

type ncaHeader struct {
	fixedKeySig        [0x100]byte // RSA-PSS signature over header with fixed key
	npdmKeySig         [0x100]byte // RSA-PSS signature over header with key in NPDM
	magic              uint32
	distribution       uint8 // System vs gamecard
	contentType        uint8
	cryptoType         uint8  // Which keyblob (field 1)
	keyAreaKeyIndex    uint8  // Which kaek index?
	ncaSize            uint64 // Entire archive size
	titleID            uint64
	_                  [4]byte
	sdkVersion         uint32 // This could also be accessed as SdkVersionParts
	cryptoType2        uint8  // Which keyblob (field 2)
	fixedKeyGeneration uint8
	_                  [0xF]byte
	rightsID           [0x10]byte         // Rights ID (for titlekey crypto)
	sectionEntries     [4]ncaSectionEntry // Section entry metadata
	sectionHashes      [4][0x20]byte      // SHA-256 hashes for each section header
	encryptedKeys      [4][0x10]byte      // Encrypted key area
	_                  [0xC0]byte
	fsHeaders          [4]ncaFsHeader // FS section headers
}

func (h *ncaHeader) GetSdkVersionParts() SdkVersion {
	return SdkVersion{
		Revision: uint8(h.sdkVersion),
		Micro:    uint8(h.sdkVersion >> 8),
		Minor:    uint8(h.sdkVersion >> 16),
		Major:    uint8(h.sdkVersion >> 24),
	}
}

type Nca struct {
	header        ncaHeader
	decryptedKeys [4][0x10]byte
}

func (n Nca) String() string {
	var sb strings.Builder
	sb.WriteString("NCA:\n")

	m := make([]byte, 4)
	binary.LittleEndian.PutUint32(m, n.header.magic)
	sdk := n.header.GetSdkVersionParts()

	type field struct {
		label string
		value string
	}
	writeFields := func(fields []field) {
		maxLen := 0
		for _, f := range fields {
			if len(f.label) > maxLen {
				maxLen = len(f.label)
			}
		}

		// account for ":"
		maxLen++

		for _, f := range fields {
			label := f.label + ":"
			value := f.value
			if len(value) <= 64 {
				sb.WriteString(fmt.Sprintf("%-*s %s\n", maxLen, label, f.value))
				continue
			}

			sb.WriteString(fmt.Sprintf("%-*s %s\n", maxLen, label, value[:min(len(value), 64)]))
			for {
				value = value[min(len(value), 64):]
				if len(value) == 0 {
					break
				}

				sb.WriteString(fmt.Sprintf("%-*s %s\n", maxLen, "", value[:min(len(value), 64)]))
			}
		}
	}

	allFields := []field{
		{"Magic", string(m)},
		{"Fixed-Key Index", fmt.Sprintf("0x%x", n.header.fixedKeyGeneration)},
		{"Fixed-Key Signature", fmt.Sprintf("%x", n.header.fixedKeySig)},
		{"NPDM Signature", fmt.Sprintf("%x", n.header.npdmKeySig)},
		{"Content Size", fmt.Sprintf("0x%016x", n.header.ncaSize)},
		{"Title Id", fmt.Sprintf("0x%016x", n.header.titleID)},
		{"SDK Version", fmt.Sprintf("%d.%d.%d.%d", sdk.Major, sdk.Minor, sdk.Micro, sdk.Revision)},
		// FIXME
		{"Encryption Type", "??"},
		{"Key Area Encryption Key", fmt.Sprintf("%d", n.header.keyAreaKeyIndex)},
		{"Key Area (Encrypted)", ""},
		{"    Key %d (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[0])},
		{"    Key %d (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[1])},
		{"    Key %d (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[2])},
		{"    Key %d (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[3])},
		{"Key Area (Decrypted)", ""},
		{"    Key %d (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[0])},
		{"    Key %d (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[1])},
		{"    Key %d (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[2])},
		{"    Key %d (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[3])},
		{"Sections", ""},
	}

	for i, section := range n.header.sectionEntries {
		if section.MediaStartOffset == 0 {
			continue
		}

		sectionHeader := n.header.fsHeaders[i]
		start := section.MediaStartOffset * 0x200
		end := section.MediaEndOffset * 0x200
		fsType := sectionHeader.FsType()
		superblock := pfs0SuperblockFromBytes(sectionHeader.superblock)

		allFields = append(allFields,
			field{fmt.Sprintf("    Section %d", i), ""},
			field{"        Offset", fmt.Sprintf("0x%016x", start)},
			field{"        Size", fmt.Sprintf("0x%016x", end-start)},
			field{"        Partition Type", fsType.String()},
		)

		switch fsType {
		case FsTypePfs0:
			allFields = append(allFields,
				field{"        Section CTR", fmt.Sprintf("%x", sectionHeader.SectionCtr(section.MediaStartOffset))},
				field{"        Superblock Hash", fmt.Sprintf("%x", superblock.masterHash)},
				field{"        Hash Table", ""},
				field{"            Offset", fmt.Sprintf("%016x", superblock.hashTableOffset)},
				field{"            Size", fmt.Sprintf("%016x", superblock.hashTableSize)},
				field{"            Block Size", fmt.Sprintf("0x%x", superblock.blockSize)},
				field{"        PFS0 Offset", fmt.Sprintf("%016x", superblock.pfs0Offset)},
				field{"        PFS0 Size", fmt.Sprintf("%016x", superblock.pfs0Size)},
			)
		case FsTypeRomfs:
			allFields = append(allFields, field{"        TODO ROMFS", ""})
		default:
			allFields = append(allFields, field{"        TODO", ""})
		}

	}

	writeFields(allFields)

	return sb.String()

}

func NewNcaFromBytes(data []byte, keys keys.Keys) (*Nca, error) {
	encrypted := binary.LittleEndian.Uint32(data[0x200:0x204]) != magicNCA3

	// decrypt header
	var plain []byte
	if encrypted {
		plain = make([]byte, len(data))
		copy(plain, data)

		c, err := xtsn.NewXtsnCipher(keys.HeaderKey[16:], keys.HeaderKey[:16], 0x200)
		if err != nil {
			return nil, err
		}

		c.Decrypt(plain[:0xc00], 0)

		if binary.LittleEndian.Uint32(plain[0x200:0x204]) != magicNCA3 {
			return nil, fmt.Errorf("failed to decrypt NCA header")
		}
	} else {
		plain = data
	}

	// parse header
	header := ncaHeaderFromBytes(plain[:0xc00])

	// decrypt the `encryptedKeys` in the header
	encryptedKeysBytes := make([]byte, 0x40)
	copy(encryptedKeysBytes, plain[0x300:0x340])
	key, err := keys.GetKeyAreaKey(int(header.cryptoType), int(header.keyAreaKeyIndex))
	if err != nil {
		return nil, err
	}

	err = aesEcbDecrypt(key, encryptedKeysBytes)
	if err != nil {
		return nil, err
	}

	var decryptedKeys [4][0x10]byte
	copy(decryptedKeys[0][:], encryptedKeysBytes[0x00:0x10])
	copy(decryptedKeys[1][:], encryptedKeysBytes[0x10:0x20])
	copy(decryptedKeys[2][:], encryptedKeysBytes[0x20:0x30])
	copy(decryptedKeys[3][:], encryptedKeysBytes[0x30:0x40])

	return &Nca{
		header:        header,
		decryptedKeys: decryptedKeys,
	}, nil
}

// assumes decrypted bytes
func ncaHeaderFromBytes(plain []byte) ncaHeader {
	header := ncaHeader{
		magic:              binary.LittleEndian.Uint32(plain[0x200:0x204]),
		distribution:       plain[0x204],
		contentType:        plain[0x205],
		cryptoType:         plain[0x206],
		keyAreaKeyIndex:    plain[0x207],
		ncaSize:            binary.LittleEndian.Uint64(plain[0x208:0x210]),
		titleID:            binary.LittleEndian.Uint64(plain[0x210:0x218]),
		sdkVersion:         binary.LittleEndian.Uint32(plain[0x21c:0x220]),
		cryptoType2:        plain[0x220],
		fixedKeyGeneration: plain[0x221],
		sectionEntries:     parseSectionEntries(plain[0x240:0x280]),
		fsHeaders:          parseFsHeaders(plain[0x400:0xc00]),
	}
	copy(header.fixedKeySig[:], plain[0:0x100])
	copy(header.npdmKeySig[:], plain[0x100:0x200])
	copy(header.rightsID[:], plain[0x230:0x240])
	copy(header.sectionHashes[0][:], plain[0x280:0x2a0])
	copy(header.sectionHashes[1][:], plain[0x2a0:0x2c0])
	copy(header.sectionHashes[2][:], plain[0x2c0:0x2e0])
	copy(header.sectionHashes[3][:], plain[0x2e0:0x300])
	copy(header.encryptedKeys[0][:], plain[0x300:0x310])
	copy(header.encryptedKeys[1][:], plain[0x310:0x320])
	copy(header.encryptedKeys[2][:], plain[0x320:0x330])
	copy(header.encryptedKeys[3][:], plain[0x330:0x340])
	return header
}

// TODO: move this somewhere more relevant
func aesEcbDecrypt(key, data []byte) error {
	if len(data)%aes.BlockSize != 0 {
		return fmt.Errorf("invalid ECB input length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	for bs, be := 0, aes.BlockSize; bs < len(data); bs, be = bs+aes.BlockSize, be+aes.BlockSize {
		block.Decrypt(data[bs:be], data[bs:be])
	}
	return nil
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
			version:   binary.LittleEndian.Uint16(data[start : start+2]),
			fsType:    data[start+2],
			hashType:  data[start+3],
			cryptType: data[start+4],
		}
		copy(fsHeaders[i].superblock[:], data[start+8:start+0x140])
		copy(fsHeaders[i].sectionCtr[:], data[start+0x140:start+0x148])
	}

	return fsHeaders
}
