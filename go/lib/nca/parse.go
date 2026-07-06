package nca

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/xtsn"
)

const (
	magicNCA3              uint32 = 0x3341434E
	pfs0ExefsHashBlockSize uint32 = 0x10000
	ncaSectionEntrySize           = 0x10
	ncaFsHeaderSize               = 0x200
	mediaSize                     = 0x200
)

type DistributionType uint8

const (
	DistributionTypeDownload DistributionType = 0
	DistributionTypeGamecard DistributionType = 1
)

func (dt DistributionType) String() string {
	switch dt {
	case DistributionTypeDownload:
		return "Download"
	case DistributionTypeGamecard:
		return "Gamecard"
	default:
		return "Unknown"
	}
}

type ContentType uint8

const (
	ContentTypeProgram    ContentType = 0
	ContentTypeMeta       ContentType = 1
	ContentTypeControl    ContentType = 2
	ContentTypeManual     ContentType = 3
	ContentTypeData       ContentType = 4
	ContentTypePublicData ContentType = 5
)

func (dt ContentType) String() string {
	switch dt {
	case ContentTypeProgram:
		return "Program"
	case ContentTypeMeta:
		return "Meta"
	case ContentTypeControl:
		return "Control"
	case ContentTypeManual:
		return "Manual"
	case ContentTypeData:
		return "Data"
	case ContentTypePublicData:
		return "PublicData"
	default:
		return "Unknown"
	}
}

type CryptoType uint8

func (ct CryptoType) String() string {
	switch ct {
	case 0:
		return "1.0.0-2.3.0"
	case 1:
		return "3.0.0"
	case 2:
		return "3.0.1-3.0.2"
	case 3:
		return "4.0.0-4.1.0"
	case 4:
		return "5.0.0-5.1.0"
	case 5:
		return "6.0.0-6.1.0"
	case 6:
		return "6.2.0"
	case 7:
		return "7.0.0-8.0.1"
	case 8:
		return "8.1.0-8.1.1"
	case 9:
		return "9.0.0-9.0.1"
	case 0xA:
		return "9.1.0-"
	default:
		return "Unknown"
	}
}

type ncaSectionEntry struct {
	MediaStartOffset uint32
	MediaEndOffset   uint32
	_                [8]byte
}

type SectionFsType uint8

const (
	SectionFsTypeRomfs SectionFsType = 0
	SectionFsTypePfs0  SectionFsType = 1
)

func (t SectionFsType) String() string {
	switch t {
	case SectionFsTypeRomfs:
		return "ROMFS"
	case SectionFsTypePfs0:
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

func (sc SectionCryptType) String() string {
	switch sc {
	case CryptNone:
		return "None"
	case CryptXts:
		return "XTS"
	case CryptCtr:
		return "CTR"
	case CryptBktr:
		return "BKTR"
	default:
		return "Unknown"
	}
}

type ncaFsHeader struct {
	version    uint16
	fsType     SectionFsType
	hashType   SectionHashType
	cryptType  SectionCryptType
	_          [3]byte
	superblock [0x138]byte // FS-specific superblock
	sectionCtr [8]byte
	_          [0xB8]byte
}

func (fsHeader *ncaFsHeader) SectionCtr(mediaStartOffset uint32) [0x10]byte {
	var ctr [0x10]byte
	offset := uint64(mediaStartOffset*mediaSize) >> 4
	for i := range 8 {
		ctr[i] = fsHeader.sectionCtr[0x8-i-1]
		ctr[0x10-i-1] = byte(offset & 0xff)
		offset >>= 8
	}

	return ctr
}

func (fsHeader *ncaFsHeader) FsType() SectionFsType {
	return SectionFsType(fsHeader.fsType)
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

const ivfcMaxLevel uint32 = 6

type ivfcLevelHeader struct {
	logicalOffset uint64
	hashDataSize  uint64
	blockSize     uint32
	_             uint32
}

func (header *ivfcLevelHeader) HashBlockSize() uint32 {
	return 1 << header.blockSize
}

type ivfcHeader struct {
	magic          uint32
	id             uint32
	masterHashSize uint32
	numLevels      uint32
	levelHeaders   [ivfcMaxLevel]ivfcLevelHeader
	_              [0x20]byte
	masterHash     [0x20]byte
}

type romfsSuperblock struct {
	ivfcHeader ivfcHeader
	_          [0x58]byte
}

func romfsSuperblockFromBytes(data [0x138]byte) romfsSuperblock {
	ivfcHeader := ivfcHeader{
		magic:          binary.LittleEndian.Uint32(data[:0x4]),
		id:             binary.LittleEndian.Uint32(data[0x4:0x8]),
		masterHashSize: binary.LittleEndian.Uint32(data[0x8:0xc]),
		numLevels:      binary.LittleEndian.Uint32(data[0xc:0x10]),
	}
	copy(ivfcHeader.masterHash[:], data[0xc0:0xe0])

	ivfcStart := uint32(0x10)
	for i := range ivfcMaxLevel {
		off := ivfcStart + (i * 24)
		ivfcHeader.levelHeaders[i] = ivfcLevelHeader{
			logicalOffset: binary.LittleEndian.Uint64(data[off : off+0x8]),
			hashDataSize:  binary.LittleEndian.Uint64(data[off+0x8 : off+0x10]),
			blockSize:     binary.LittleEndian.Uint32(data[off+0x10 : off+0x14]),
		}
	}

	return romfsSuperblock{ivfcHeader: ivfcHeader}
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
	distribution       DistributionType
	contentType        ContentType
	cryptoType         CryptoType
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

func (h *ncaHeader) MasterKeyIndex() int {
	revision := int(max(uint8(h.cryptoType), h.cryptoType2))
	if revision <= 0 {
		return 0
	}
	return revision - 1
}

func (h *ncaHeader) GetSdkVersionParts() SdkVersion {
	return SdkVersion{
		Revision: uint8(h.sdkVersion),
		Micro:    uint8(h.sdkVersion >> 8),
		Minor:    uint8(h.sdkVersion >> 16),
		Major:    uint8(h.sdkVersion >> 24),
	}
}

func (h *ncaHeader) HasRightsId() bool {
	for _, b := range h.rightsID {
		if b != 0x0 {
			return true
		}
	}

	return false
}

func (h *ncaHeader) needsKeyAreaKey() bool {
	if h.HasRightsId() {
		return false
	}
	for _, section := range h.fsHeaders {
		if section.cryptType == CryptCtr || section.cryptType == CryptBktr || section.cryptType == CryptXts {
			return true
		}
	}
	return false
}

func (h *ncaHeader) EncryptionType() string {
	if h.HasRightsId() {
		return "Titlekey crypto"
	}

	return "Standard crypto"
}

type NcaSection struct {
	sectionOffset int64
	size          int64
	cryptType     SectionCryptType
	key           []byte
	ctr           [0x10]byte
	sectionHeader ncaFsHeader
	nca           *Nca
}

type CtrReader struct {
	nca           *Nca
	key           []byte
	ctr           [0x10]byte
	sectionOffset int64
	dataSize      int64
	readOffset    int64
}

func (r *CtrReader) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.readOffset)
	r.readOffset += int64(n)
	return n, err
}

func (r *CtrReader) ctrForOffset(offset int64) []byte {
	ctr := bytes.Clone(r.ctr[:])
	binary.BigEndian.PutUint64(ctr[8:], uint64(r.sectionOffset+offset)>>4)
	return ctr
}

func (r *CtrReader) ReadAt(p []byte, off int64) (int, error) {
	block, err := aes.NewCipher(r.key[:])
	if err != nil {
		return 0, err
	}

	before := off % 0x10
	alignedOffset := off - before

	// if offset is aligned, just read directly
	if before == 0 {
		ctrReader := cipher.StreamReader{
			S: cipher.NewCTR(block, r.ctrForOffset(alignedOffset)),
			R: io.NewSectionReader(r.nca.reader, r.sectionOffset+off, r.dataSize),
		}
		return ctrReader.Read(p)
	}

	// if the read isn't aligned, then read the block that the offset is in...
	var prefix [16]byte
	prefixReader := cipher.StreamReader{
		S: cipher.NewCTR(block, r.ctrForOffset(alignedOffset)),
		R: io.NewSectionReader(r.nca.reader, r.sectionOffset+alignedOffset, 16),
	}
	if _, err := io.ReadFull(prefixReader, prefix[:]); err != nil {
		return 0, err
	}

	// ... copy the part the caller wanted into p ...
	n := copy(p, prefix[before:])

	// ... and finally perform the rest of the read
	ctrReader := cipher.StreamReader{
		S: cipher.NewCTR(block, r.ctrForOffset(alignedOffset+16)),
		R: io.NewSectionReader(r.nca.reader, r.sectionOffset+alignedOffset+16, r.dataSize),
	}
	m, err := ctrReader.Read(p[n:])
	return n + m, err
}

func (s *NcaSection) Open() (NcaReader, error) {
	var dataOffset, dataSize int64
	switch s.sectionHeader.FsType() {
	case SectionFsTypePfs0:
		superblock := pfs0SuperblockFromBytes(s.sectionHeader.superblock)
		dataOffset = int64(superblock.pfs0Offset)
		dataSize = int64(superblock.pfs0Size)
	case SectionFsTypeRomfs:
		superblock := romfsSuperblockFromBytes(s.sectionHeader.superblock)
		lvl := superblock.ivfcHeader.levelHeaders[ivfcMaxLevel-1]
		dataOffset = int64(lvl.logicalOffset)
		dataSize = int64(lvl.hashDataSize)
	default:
		return nil, fmt.Errorf("currently unsupported fs type %s", s.sectionHeader.FsType())
	}

	switch s.cryptType {
	case CryptNone:
		return io.NewSectionReader(s.nca.reader, s.sectionOffset+dataOffset, dataSize), nil
	case CryptCtr:
		return &CtrReader{
			nca:           s.nca,
			key:           s.key,
			ctr:           s.ctr,
			dataSize:      dataSize,
			sectionOffset: s.sectionOffset + dataOffset,
		}, nil
	// TODO other encryption types (XTS with xtsn, etc)
	default:
		return nil, fmt.Errorf("unimplemented decryption for sections encrypted with %s", s.cryptType)
	}
}

func (s *NcaSection) FsType() SectionFsType {
	return s.sectionHeader.FsType()
}

func (s *NcaSection) Offset() int64 {
	return s.sectionOffset
}

func (s *NcaSection) Size() int64 {
	return s.size
}

func (s *NcaSection) CryptType() SectionCryptType {
	return s.cryptType
}

func (s *NcaSection) Key() []byte {
	return bytes.Clone(s.key)
}

func (s *NcaSection) Counter() [0x10]byte {
	return s.ctr
}

type Nca struct {
	reader        NcaReader
	header        ncaHeader
	decryptedKeys [4][0x10]byte
	titleKey      []byte
}

func (n *Nca) HasRightsID() bool {
	return n.header.HasRightsId()
}

func (n *Nca) RightsID() [0x10]byte {
	return n.header.rightsID
}

func (n *Nca) HasTitleKey() bool {
	return len(n.titleKey) == 0x10
}

func (n *Nca) Sections() []NcaSection {
	sections := make([]NcaSection, 0)
	for i, section := range n.header.sectionEntries {
		if section.MediaStartOffset == 0 {
			continue
		}

		sectionHeader := n.header.fsHeaders[i]
		start := section.MediaStartOffset * mediaSize
		end := section.MediaEndOffset * mediaSize

		var key []byte
		if n.header.HasRightsId() {
			key = bytes.Clone(n.titleKey)
			if key == nil {
				key = make([]byte, 0x10)
			}
		} else {
			if sectionHeader.cryptType == CryptCtr || sectionHeader.cryptType == CryptBktr {
				key = n.decryptedKeys[2][:]
			} else if sectionHeader.cryptType == CryptXts {
				key = bytes.Join([][]byte{n.decryptedKeys[0][:], n.decryptedKeys[1][:]}, []byte{})
			}
		}

		sections = append(sections, NcaSection{
			sectionOffset: int64(start),
			size:          int64(end - start),
			key:           key,
			ctr:           sectionHeader.SectionCtr(section.MediaStartOffset),
			sectionHeader: sectionHeader,
			cryptType:     sectionHeader.cryptType,
			nca:           n,
		})
	}

	return sections
}

func (n Nca) String() string {
	var sb strings.Builder
	sb.WriteString("NCA:\n")

	nToString := func(n uint32) string {
		m := make([]byte, 4)
		binary.LittleEndian.PutUint32(m, n)
		return string(m)
	}

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

	sdk := n.header.GetSdkVersionParts()
	allFields := []field{
		{"Magic", nToString(n.header.magic)},
		{"Fixed-Key Index", fmt.Sprintf("0x%x", n.header.fixedKeyGeneration)},
		{"Fixed-Key Signature", fmt.Sprintf("%x", n.header.fixedKeySig)},
		{"NPDM Signature", fmt.Sprintf("%x", n.header.npdmKeySig)},
		{"Content Size", fmt.Sprintf("0x%016x", n.header.ncaSize)},
		{"Title Id", fmt.Sprintf("0x%016x", n.header.titleID)},
		{"SDK Version", fmt.Sprintf("%d.%d.%d.%d", sdk.Major, sdk.Minor, sdk.Micro, sdk.Revision)},
		{"Distribution Type", n.header.distribution.String()},
		{"Content Type", n.header.contentType.String()},
		{"Master Key Revision", fmt.Sprintf("0x%x (%s)", uint8(n.header.cryptoType), n.header.cryptoType.String())},
		{"Encryption Type", n.header.EncryptionType()},
		{"Key Area Encryption Key", fmt.Sprintf("%d", n.header.keyAreaKeyIndex)},
		{"Key Area (Encrypted)", ""},
		{"    Key 0 (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[0])},
		{"    Key 1 (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[1])},
		{"    Key 2 (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[2])},
		{"    Key 3 (Encrypted)", fmt.Sprintf("%x", n.header.encryptedKeys[3])},
		{"Key Area (Decrypted)", ""},
		{"    Key 0 (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[0])},
		{"    Key 1 (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[1])},
		{"    Key 2 (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[2])},
		{"    Key 3 (Decrypted)", fmt.Sprintf("%x", n.decryptedKeys[3])},
		{"Sections", ""},
	}

	for i, section := range n.header.sectionEntries {
		if section.MediaStartOffset == 0 {
			continue
		}

		sectionHeader := n.header.fsHeaders[i]
		start := section.MediaStartOffset * mediaSize
		end := section.MediaEndOffset * mediaSize
		fsType := sectionHeader.FsType()

		allFields = append(allFields,
			field{fmt.Sprintf("    Section %d", i), ""},
			field{"        Offset", fmt.Sprintf("0x%016x", start)},
			field{"        Size", fmt.Sprintf("0x%016x", end-start)},
			field{"        Partition Type", fsType.String()},
			field{"        Section CTR", fmt.Sprintf("%x", sectionHeader.SectionCtr(section.MediaStartOffset))},
		)

		switch fsType {
		case SectionFsTypePfs0:
			superblock := pfs0SuperblockFromBytes(sectionHeader.superblock)
			allFields = append(allFields,
				field{"        Superblock Hash", fmt.Sprintf("%x", superblock.masterHash)},
				field{"        Hash Table", ""},
				field{"            Offset", fmt.Sprintf("%016x", superblock.hashTableOffset)},
				field{"            Size", fmt.Sprintf("%016x", superblock.hashTableSize)},
				field{"            Block Size", fmt.Sprintf("0x%x", superblock.blockSize)},
				field{"        PFS0 Offset", fmt.Sprintf("%016x", superblock.pfs0Offset)},
				field{"        PFS0 Size", fmt.Sprintf("%016x", superblock.pfs0Size)},
			)
		case SectionFsTypeRomfs:
			superblock := romfsSuperblockFromBytes(sectionHeader.superblock)
			allFields = append(allFields,
				field{"        Superblock Hash", fmt.Sprintf("%x", superblock.ivfcHeader.masterHash)},
				field{"        Magic", nToString(superblock.ivfcHeader.magic)},
				field{"        ID", fmt.Sprintf("%08x", superblock.ivfcHeader.id)},
			)

			for i := range superblock.ivfcHeader.numLevels - 1 {
				allFields = append(
					allFields,
					field{fmt.Sprintf("        Level %d", i), ""},
					field{"            Data Offset", fmt.Sprintf("0x%012x", superblock.ivfcHeader.levelHeaders[i].logicalOffset)},
					field{"            Data Size", fmt.Sprintf("0x%012x", superblock.ivfcHeader.levelHeaders[i].hashDataSize)},
				)
				if i > 0 {
					allFields = append(allFields, field{"            Hash Offset", fmt.Sprintf("0x%012x", superblock.ivfcHeader.levelHeaders[i-1].logicalOffset)})
				}
				allFields = append(allFields, field{"            Hash Block Size", fmt.Sprintf("0x%08x", superblock.ivfcHeader.levelHeaders[i].HashBlockSize())})
			}
		// TODO: NCA0_ROMFS
		// TODO: BKTR
		// TODO: INVALID
		default:
			allFields = append(allFields, field{"        Unknown/invalid superblock", ""})
		}

	}

	writeFields(allFields)

	return sb.String()

}

type NcaReader interface {
	io.Reader
	io.ReaderAt
}

func NewNca(reader NcaReader, keys keys.Keys) (*Nca, error) {
	return NewNcaWithTitleKeys(reader, keys, nil)
}

func NewNcaWithTitleKeys(reader NcaReader, keys keys.Keys, titleKeys map[[0x10]byte][]byte) (*Nca, error) {
	data := make([]byte, 0xc00)
	_, err := reader.ReadAt(data, 0)
	if err != nil {
		return nil, err
	}

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
	nca := &Nca{reader: reader, header: header}

	if header.HasRightsId() {
		titleKey := titleKeys[header.rightsID]
		if len(titleKey) != 0 {
			if len(titleKey) != 0x10 {
				return nil, fmt.Errorf("invalid title key length for rights ID %x", header.rightsID)
			}
			nca.titleKey = bytes.Clone(titleKey)
		}
	} else if header.needsKeyAreaKey() {
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

		copy(nca.decryptedKeys[0][:], encryptedKeysBytes[0x00:0x10])
		copy(nca.decryptedKeys[1][:], encryptedKeysBytes[0x10:0x20])
		copy(nca.decryptedKeys[2][:], encryptedKeysBytes[0x20:0x30])
		copy(nca.decryptedKeys[3][:], encryptedKeysBytes[0x30:0x40])
	}

	return nca, nil
}

// assumes decrypted bytes
func ncaHeaderFromBytes(plain []byte) ncaHeader {
	header := ncaHeader{
		magic:              binary.LittleEndian.Uint32(plain[0x200:0x204]),
		distribution:       DistributionType(plain[0x204]),
		contentType:        ContentType(plain[0x205]),
		cryptoType:         CryptoType(plain[0x206]),
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
			fsType:    SectionFsType(data[start+2]),
			hashType:  SectionHashType(data[start+3]),
			cryptType: SectionCryptType(data[start+4]),
		}
		copy(fsHeaders[i].superblock[:], data[start+8:start+0x140])
		copy(fsHeaders[i].sectionCtr[:], data[start+0x140:start+0x148])
	}

	return fsHeaders
}
