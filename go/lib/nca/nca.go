package nca

const (
	MagicNCA3              uint32 = 0x3341434E
	Pfs0ExefsHashBlockSize uint32 = 0x10000
)

type NcaSectionEntry struct {
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

type NcaFsHeader struct {
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

type NcaHeader struct {
	FixedKeySig    [0x100]byte // RSA-PSS signature over header with fixed key
	NpdmKeySig     [0x100]byte // RSA-PSS signature over header with key in NPDM
	Magic          uint32
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
	SectionEntries [4]NcaSectionEntry // Section entry metadata
	SectionHashes  [4][0x20]byte      // SHA-256 hashes for each section header
	EncryptedKeys  [4][0x10]byte      // Encrypted key area
	_              [0xC0]byte
	FsHeaders      [4]NcaFsHeader // FS section headers
}

func (h *NcaHeader) GetSdkVersionParts() SdkVersion {
	return SdkVersion{
		Revision: uint8(h.SdkVersion),
		Micro:    uint8(h.SdkVersion >> 8),
		Minor:    uint8(h.SdkVersion >> 16),
		Major:    uint8(h.SdkVersion >> 24),
	}
}
