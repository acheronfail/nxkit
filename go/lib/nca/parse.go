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
	sb.WriteString(fmt.Sprintf("Magic: %s\n", string(m)))

	sb.WriteString(fmt.Sprintf("Fixed-Key Index: %x\n", n.header.fixedKeyGeneration))
	sb.WriteString(fmt.Sprintf("Fixed-Key Signature: %x\n", n.header.fixedKeySig))
	sb.WriteString(fmt.Sprintf("NPDM Signature: %x\n", n.header.npdmKeySig))
	sb.WriteString(fmt.Sprintf("Content Size: 0x%x\n", n.header.ncaSize))
	sb.WriteString(fmt.Sprintf("Title Id: 0x%x\n", n.header.titleID))
	sdk := n.header.GetSdkVersionParts()
	sb.WriteString(fmt.Sprintf("SDK Version: %d.%d.%d.%d\n", sdk.Major, sdk.Minor, sdk.Micro, sdk.Revision))
	// FIXME determine this
	sb.WriteString(fmt.Sprintf("Encryption Type: %s\n", "??"))
	sb.WriteString(fmt.Sprintf("Key Area Encryption Key: %d\n", n.header.keyAreaKeyIndex))
	sb.WriteString("Key Area (Encrypted):\n")
	sb.WriteString(fmt.Sprintf("    Key 0 (Encrypted): %x\n", n.header.encryptedKeys[0]))
	sb.WriteString(fmt.Sprintf("    Key 1 (Encrypted): %x\n", n.header.encryptedKeys[1]))
	sb.WriteString(fmt.Sprintf("    Key 2 (Encrypted): %x\n", n.header.encryptedKeys[2]))
	sb.WriteString(fmt.Sprintf("    Key 3 (Encrypted): %x\n", n.header.encryptedKeys[3]))
	sb.WriteString("Key Area (Decrypted):\n")
	sb.WriteString(fmt.Sprintf("    Key 0 (Decrypted): %x\n", n.decryptedKeys[0]))
	sb.WriteString(fmt.Sprintf("    Key 1 (Decrypted): %x\n", n.decryptedKeys[1]))
	sb.WriteString(fmt.Sprintf("    Key 2 (Decrypted): %x\n", n.decryptedKeys[2]))
	sb.WriteString(fmt.Sprintf("    Key 3 (Decrypted): %x\n", n.decryptedKeys[3]))
	sb.WriteString("Sections:\n")
	// FIXME
	sb.WriteString("    ??\n")

	return sb.String()
}

// TODO: make keys struct so I can select which one
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

		_, err = c.Decrypt(plain[0x000:0xc00], 0)
		if err != nil {
			return nil, err
		}

		if binary.LittleEndian.Uint32(plain[0x200:0x204]) != magicNCA3 {
			return nil, fmt.Errorf("failed to decrypt NCA header")
		}
	} else {
		plain = data
	}

	// parse header
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
