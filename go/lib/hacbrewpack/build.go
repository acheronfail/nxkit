package hacbrewpack

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acheronfail/nxkit/lib/cnmt"
	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/nacp"
	"github.com/acheronfail/nxkit/lib/npdm"
	"github.com/acheronfail/nxkit/lib/pfs0"
	"github.com/acheronfail/nxkit/lib/romfs"
	"github.com/acheronfail/nxkit/lib/xtsn"
)

const (
	defaultTitleID     uint64 = 0x0162696bc58e0000
	defaultSDKVersion  uint32 = 0x000c1100
	ncaHeaderSize             = 0xc00
	mediaSize                 = 0x200
	ivfcHashBlockSize         = 0x4000
	exefsHashBlockSize uint32 = 0x10000
	logoHashBlockSize  uint32 = 0x1000
	metaHashBlockSize  uint32 = 0x1000
)

type Options struct {
	OutPath          string
	KeysPath         string
	ExefsMainPath    string
	ExefsNPDMPath    string
	ProgramRomFSDir  string
	LogoDir          string
	NintendoLogoPath string
	StartupMoviePath string
	ControlDir       string
	IconPath         string
	HtmlDocDir       string
	LegalInfoDir     string
	TitleID          uint64
	TitleName        string
	TitlePublisher   string
	Version          string
	NROPath          string
	NROArgv          []string
	SDKVersion       uint32
	KeyGeneration    int
	KeyAreaKey       []byte
	Plaintext        bool
	NoRomFS          bool
	NoLogo           bool
	NoSignNCASig2    bool
	NoPatchNACPLogo  bool
	// RandomPSS uses normal randomized RSA-PSS for the Program NCA header signature.
	// Leave false for reproducible byte-for-byte NSP output.
	RandomPSS bool
}

type contentRecord struct {
	hash  [32]byte
	ncaID [16]byte
	size  [6]byte
	typ   byte
	id    byte
}

type buildContext struct {
	opt           Options
	tmpDir        string
	keyArea       []byte
	header        []byte
	priv          *rsa.PrivateKey
	titleID       uint64
	sdkVersion    uint32
	keyGeneration int
	program       contentRecord
	control       contentRecord
	htmlDoc       contentRecord
	legalInfo     contentRecord
	meta          contentRecord
}

func BuildForwarderNSP(opt Options) error {
	if err := opt.setDefaults(); err != nil {
		return err
	}
	ctx := &buildContext{
		opt:           opt,
		keyArea:       append([]byte(nil), opt.KeyAreaKey...),
		titleID:       opt.TitleID,
		sdkVersion:    opt.SDKVersion,
		keyGeneration: opt.KeyGeneration,
	}
	if ctx.opt.KeysPath == "" {
		return fmt.Errorf("KeysPath is required")
	}

	keyset, err := keys.NewFromPath(ctx.opt.KeysPath)
	if err != nil {
		return err
	}
	if len(keyset.HeaderKey) != 32 {
		return fmt.Errorf("header_key must be 32 bytes, got %d", len(keyset.HeaderKey))
	}
	keyAreaKey, err := keyset.GetKeyAreaKey(0, ctx.keyGeneration-1)
	if err != nil {
		return err
	}
	if len(keyAreaKey) != 16 {
		return fmt.Errorf("key_area_key_application_%02x must be 16 bytes, got %d", ctx.keyGeneration-1, len(keyAreaKey))
	}
	ctx.header = keyset.HeaderKey
	ctx.priv, err = parsePrivateKey(defaultPrivateKeyPEM)
	if err != nil {
		return err
	}

	ctx.tmpDir, err = os.MkdirTemp("", "nxkit-go-hacbrewpack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(ctx.tmpDir)

	if err := ctx.stageInputs(); err != nil {
		return err
	}
	if err := ctx.createProgramNCA(keyAreaKey); err != nil {
		return err
	}
	if err := ctx.createControlNCA(keyAreaKey); err != nil {
		return err
	}
	if ctx.opt.HtmlDocDir != "" {
		if err := ctx.createManualNCA(keyAreaKey, "htmldoc", "manual_htmldoc", cnmt.ContentTypeHtmlDocument, &ctx.htmlDoc); err != nil {
			return err
		}
	}
	if ctx.opt.LegalInfoDir != "" {
		if err := ctx.createManualNCA(keyAreaKey, "legalinfo", "manual_legalinfo", cnmt.ContentTypeLegalInfo, &ctx.legalInfo); err != nil {
			return err
		}
	}
	if err := ctx.createMetaNCA(keyAreaKey); err != nil {
		return err
	}

	entries := []pfs0.BuildEntry{
		{Name: hex.EncodeToString(ctx.program.ncaID[:]) + ".nca", Path: filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(ctx.program.ncaID[:])+".nca")},
		{Name: hex.EncodeToString(ctx.control.ncaID[:]) + ".nca", Path: filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(ctx.control.ncaID[:])+".nca")},
	}
	if ctx.opt.HtmlDocDir != "" {
		entries = append(entries, pfs0.BuildEntry{Name: hex.EncodeToString(ctx.htmlDoc.ncaID[:]) + ".nca", Path: filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(ctx.htmlDoc.ncaID[:])+".nca")})
	}
	if ctx.opt.LegalInfoDir != "" {
		entries = append(entries, pfs0.BuildEntry{Name: hex.EncodeToString(ctx.legalInfo.ncaID[:]) + ".nca", Path: filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(ctx.legalInfo.ncaID[:])+".nca")})
	}
	entries = append(entries, pfs0.BuildEntry{Name: hex.EncodeToString(ctx.meta.ncaID[:]) + ".cnmt.nca", Path: filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(ctx.meta.ncaID[:])+".cnmt.nca")})
	_, err = pfs0.Build(ctx.opt.OutPath, entries)
	return err
}

func (opt *Options) setDefaults() error {
	if opt.TitleID == 0 {
		opt.TitleID = defaultTitleID
	}
	if opt.SDKVersion == 0 {
		opt.SDKVersion = defaultSDKVersion
	}
	if opt.SDKVersion < 0x000b0000 {
		return fmt.Errorf("SDKVersion must be at least 000b0000, got %08x", opt.SDKVersion)
	}
	if opt.KeyGeneration == 0 {
		opt.KeyGeneration = 1
	}
	if opt.KeyGeneration < 1 || opt.KeyGeneration > 32 {
		return fmt.Errorf("KeyGeneration must be in range 1-32, got %d", opt.KeyGeneration)
	}
	if opt.OutPath == "" {
		opt.OutPath = "0162696bc58e0000_title=1_publisher=2_nroPath=3.nsp"
	}
	if opt.TitleName == "" {
		opt.TitleName = "1"
	}
	if opt.TitlePublisher == "" {
		opt.TitlePublisher = "2"
	}
	if opt.Version == "" {
		opt.Version = "1.0.0"
	}
	if opt.NROPath == "" {
		opt.NROPath = "sdmc:3"
	}
	if opt.KeyAreaKey == nil {
		opt.KeyAreaKey = bytes.Repeat([]byte{0x04}, 16)
	}
	if len(opt.KeyAreaKey) != 16 {
		return fmt.Errorf("KeyAreaKey must be 16 bytes, got %d", len(opt.KeyAreaKey))
	}
	return nil
}

func (ctx *buildContext) stageInputs() error {
	for _, dir := range []string{"control", "exefs", "logo", "romfs", "nca", "temp", "backup", "htmldoc", "legalinfo"} {
		if err := os.MkdirAll(filepath.Join(ctx.tmpDir, dir), 0o755); err != nil {
			return err
		}
	}

	n, err := ctx.loadOrCreateNACP()
	if err != nil {
		return err
	}
	n.SetID(ctx.titleID)
	n.SetTitle(ctx.opt.TitleName)
	n.SetAuthor(ctx.opt.TitlePublisher)
	n.SetVersion(ctx.opt.Version)
	n.SetStartupUserAccount(0)
	n.SetScreenshot(1)
	n.SetVideoCapture(0)
	if !ctx.opt.NoPatchNACPLogo {
		n.SetLogoType(2)
		n.SetLogoHandling(0)
	}
	if err := nacp.Process(n, ctx.titleID); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(ctx.tmpDir, "control", "control.nacp"), n.Buffer(), 0o644); err != nil {
		return err
	}
	if ctx.opt.IconPath != "" {
		if err := copyFile(ctx.opt.IconPath, filepath.Join(ctx.tmpDir, "control", "icon_AmericanEnglish.dat")); err != nil {
			return err
		}
	} else if err := os.WriteFile(filepath.Join(ctx.tmpDir, "control", "icon_AmericanEnglish.dat"), defaultIcon, 0o644); err != nil {
		return err
	}
	if err := ctx.stageExeFS(); err != nil {
		return err
	}
	pubKeyPath := filepath.Join(ctx.tmpDir, "hacbrewpack.pub.pem")
	if err := os.WriteFile(pubKeyPath, defaultPublicKeyPEM, 0o644); err != nil {
		return err
	}
	if _, err := npdm.ProcessWithPublicKeyPath(
		filepath.Join(ctx.tmpDir, "exefs"),
		filepath.Join(ctx.tmpDir, "backup"),
		ctx.titleID,
		ctx.opt.NoSignNCASig2,
		pubKeyPath,
	); err != nil {
		return err
	}
	if !ctx.opt.NoLogo {
		if ctx.opt.LogoDir != "" {
			if err := copyDirContents(ctx.opt.LogoDir, filepath.Join(ctx.tmpDir, "logo")); err != nil {
				return err
			}
		} else {
			if err := copyPathOrBytes(ctx.opt.NintendoLogoPath, defaultNintendoLogo, filepath.Join(ctx.tmpDir, "logo", "NintendoLogo.png")); err != nil {
				return err
			}
			if err := copyPathOrBytes(ctx.opt.StartupMoviePath, defaultStartupMovie, filepath.Join(ctx.tmpDir, "logo", "StartupMovie.gif")); err != nil {
				return err
			}
		}
	}
	if !ctx.opt.NoRomFS {
		if ctx.opt.ProgramRomFSDir != "" {
			if err := copyDirContents(ctx.opt.ProgramRomFSDir, filepath.Join(ctx.tmpDir, "romfs")); err != nil {
				return err
			}
		} else {
			nextArgv := strings.Join(append([]string{ctx.opt.NROPath}, ctx.opt.NROArgv...), " ")
			if err := os.WriteFile(filepath.Join(ctx.tmpDir, "romfs", "nextArgv"), []byte(nextArgv), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(ctx.tmpDir, "romfs", "nextNroPath"), []byte(ctx.opt.NROPath), 0o644); err != nil {
				return err
			}
		}
	}
	if ctx.opt.HtmlDocDir != "" {
		if err := copyDirContents(ctx.opt.HtmlDocDir, filepath.Join(ctx.tmpDir, "htmldoc")); err != nil {
			return err
		}
	}
	if ctx.opt.LegalInfoDir != "" {
		if err := copyDirContents(ctx.opt.LegalInfoDir, filepath.Join(ctx.tmpDir, "legalinfo")); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *buildContext) stageExeFS() error {
	if err := copyPathOrBytes(ctx.opt.ExefsMainPath, defaultExefsMain, filepath.Join(ctx.tmpDir, "exefs", "main")); err != nil {
		return err
	}
	return copyPathOrBytes(ctx.opt.ExefsNPDMPath, defaultExefsNPDM, filepath.Join(ctx.tmpDir, "exefs", "main.npdm"))
}

func (ctx *buildContext) loadOrCreateNACP() (*nacp.Nacp, error) {
	if ctx.opt.ControlDir == "" {
		return nacp.NewNacp(nil), nil
	}
	if err := copyDirContents(ctx.opt.ControlDir, filepath.Join(ctx.tmpDir, "control")); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(ctx.tmpDir, "control", "control.nacp"))
	if err != nil {
		if os.IsNotExist(err) {
			return nacp.NewNacp(nil), nil
		}
		return nil, err
	}
	return nacp.NewNacp(data), nil
}

func (ctx *buildContext) createProgramNCA(kek []byte) error {
	nca := newNCA(0)
	exefs, err := ctx.pfs0Section("exefs", "program_sec0_exefs", exefsHashBlockSize, []pfs0.BuildEntry{
		{Name: "main", Path: filepath.Join(ctx.tmpDir, "exefs", "main")},
		{Name: "main.npdm", Path: filepath.Join(ctx.tmpDir, "exefs", "main.npdm")},
	})
	if err != nil {
		return err
	}
	if err := nca.addPFS0Section(exefs, !ctx.opt.Plaintext); err != nil {
		return err
	}
	if !ctx.opt.NoRomFS {
		rfs, err := ctx.romfsSection("romfs", "program_sec1_ivfc")
		if err != nil {
			return err
		}
		if err := nca.addRomFSSection(rfs, !ctx.opt.Plaintext); err != nil {
			return err
		}
	}
	if !ctx.opt.NoLogo {
		logoEntries, err := buildEntriesFromDir(filepath.Join(ctx.tmpDir, "logo"))
		if err != nil {
			return err
		}
		logo, err := ctx.pfs0Section("logo", "program_sec2_logo", logoHashBlockSize, logoEntries)
		if err != nil {
			return err
		}
		if err := nca.addPFS0Section(logo, false); err != nil {
			return err
		}
	}
	rec, err := nca.finalize(ctx, kek, true)
	if err != nil {
		return err
	}
	ctx.program = rec
	return os.WriteFile(filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(rec.ncaID[:])+".nca"), nca.data, 0o644)
}

func (ctx *buildContext) createManualNCA(kek []byte, sourceDir, name string, recordType byte, dst *contentRecord) error {
	nca := newNCA(3)
	rfs, err := ctx.romfsSection(sourceDir, name+"_sec0_ivfc")
	if err != nil {
		return err
	}
	if err := nca.addRomFSSection(rfs, !ctx.opt.Plaintext); err != nil {
		return err
	}
	rec, err := nca.finalize(ctx, kek, false)
	if err != nil {
		return err
	}
	rec.typ = recordType
	*dst = rec
	return os.WriteFile(filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(rec.ncaID[:])+".nca"), nca.data, 0o644)
}

func (ctx *buildContext) createControlNCA(kek []byte) error {
	nca := newNCA(2)
	rfs, err := ctx.romfsSection("control", "control_sec0_ivfc")
	if err != nil {
		return err
	}
	if err := nca.addRomFSSection(rfs, !ctx.opt.Plaintext); err != nil {
		return err
	}
	rec, err := nca.finalize(ctx, kek, false)
	if err != nil {
		return err
	}
	ctx.control = rec
	return os.WriteFile(filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(rec.ncaID[:])+".nca"), nca.data, 0o644)
}

func (ctx *buildContext) createMetaNCA(kek []byte) error {
	cnmtDir := filepath.Join(ctx.tmpDir, "temp", "cnmt")
	if err := os.MkdirAll(cnmtDir, 0o755); err != nil {
		return err
	}
	cnmtName := fmt.Sprintf("Application_%016x.cnmt", ctx.titleID)
	cnmtPath := filepath.Join(cnmtDir, cnmtName)
	records := []cnmt.ContentRecord{
		ctx.program.cnmtRecord(),
		ctx.control.cnmtRecord(),
	}
	if ctx.opt.HtmlDocDir != "" {
		records = append(records, ctx.htmlDoc.cnmtRecord())
	}
	if ctx.opt.LegalInfoDir != "" {
		records = append(records, ctx.legalInfo.cnmtRecord())
	}
	cnmtData, err := cnmt.BuildApplication(ctx.titleID, records)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cnmtPath, cnmtData, 0o644); err != nil {
		return err
	}
	nca := newNCA(1)
	sec, err := ctx.pfs0Section("cnmt", "meta_sec0_pfs0", metaHashBlockSize, []pfs0.BuildEntry{
		{Name: cnmtName, Path: cnmtPath},
	})
	if err != nil {
		return err
	}
	if err := nca.addPFS0Section(sec, !ctx.opt.Plaintext); err != nil {
		return err
	}
	rec, err := nca.finalize(ctx, kek, false)
	if err != nil {
		return err
	}
	ctx.meta = rec
	return os.WriteFile(filepath.Join(ctx.tmpDir, "nca", hex.EncodeToString(rec.ncaID[:])+".cnmt.nca"), nca.data, 0o644)
}

type pfs0Section struct {
	pfs0Path      string
	hashTablePath string
	pfs0Size      uint64
	hashTableSize uint64
	pfs0Offset    uint64
	masterHash    [32]byte
	hashBlockSize uint32
}

func (ctx *buildContext) pfs0Section(sourceDir, name string, blockSize uint32, entries []pfs0.BuildEntry) (*pfs0Section, error) {
	pfs0Path := filepath.Join(ctx.tmpDir, "temp", name)
	hashPath := filepath.Join(ctx.tmpDir, "temp", name+"_hashtable")
	pfs0Size, err := pfs0.Build(pfs0Path, entries)
	if err != nil {
		return nil, err
	}
	hashSize, pfs0Offset, err := pfs0.CreateHashTable(pfs0Path, hashPath, blockSize)
	if err != nil {
		return nil, err
	}
	master, err := pfs0.MasterHash(hashPath, hashSize)
	if err != nil {
		return nil, err
	}
	_ = sourceDir
	return &pfs0Section{pfs0Path: pfs0Path, hashTablePath: hashPath, pfs0Size: pfs0Size, hashTableSize: hashSize, pfs0Offset: pfs0Offset, masterHash: master, hashBlockSize: blockSize}, nil
}

type romfsSection struct {
	levels     [6]string
	levelSizes [6]uint64
	masterHash [32]byte
}

func (ctx *buildContext) romfsSection(sourceDir, name string) (*romfsSection, error) {
	var sec romfsSection
	level6 := filepath.Join(ctx.tmpDir, "temp", name+"_lvl6")
	if err := romfs.Build(filepath.Join(ctx.tmpDir, sourceDir), level6); err != nil {
		return nil, err
	}
	info, err := os.Stat(level6)
	if err != nil {
		return nil, err
	}
	sec.levels[5] = level6
	sec.levelSizes[5] = uint64(info.Size())
	if err := appendPadding(level6, ivfcHashBlockSize); err != nil {
		return nil, err
	}
	for i := 4; i >= 0; i-- {
		dst := filepath.Join(ctx.tmpDir, "temp", fmt.Sprintf("%s_lvl%d", name, i+1))
		size, err := createHashLevel(dst, sec.levels[i+1])
		if err != nil {
			return nil, err
		}
		sec.levels[i] = dst
		sec.levelSizes[i] = size
	}
	first, err := os.ReadFile(sec.levels[0])
	if err != nil {
		return nil, err
	}
	sec.masterHash = sha256.Sum256(first)
	return &sec, nil
}

type ncaBuilder struct {
	data        []byte
	contentType byte
	fsHeaders   [4][]byte
	entries     [4][16]byte
	section     int
}

func newNCA(contentType byte) *ncaBuilder {
	return &ncaBuilder{data: make([]byte, ncaHeaderSize), contentType: contentType}
}

func (n *ncaBuilder) addPFS0Section(sec *pfs0Section, encrypted bool) error {
	idx := n.section
	start := len(n.data)
	for _, path := range []string{sec.hashTablePath, sec.pfs0Path} {
		if err := appendFile(&n.data, path); err != nil {
			return err
		}
	}
	padMedia(&n.data)
	end := len(n.data)
	n.entries[idx] = sectionEntry(start, end)
	fs := make([]byte, 0x200)
	binary.LittleEndian.PutUint16(fs[0:], 2)
	fs[2] = 1
	fs[3] = 2
	if encrypted {
		fs[4] = 3
	} else {
		fs[4] = 1
	}
	copy(fs[8:], sec.masterHash[:])
	binary.LittleEndian.PutUint32(fs[0x28:], sec.hashBlockSize)
	binary.LittleEndian.PutUint32(fs[0x2c:], 2)
	binary.LittleEndian.PutUint64(fs[0x38:], sec.hashTableSize)
	binary.LittleEndian.PutUint64(fs[0x40:], sec.pfs0Offset)
	binary.LittleEndian.PutUint64(fs[0x48:], sec.pfs0Size)
	n.fsHeaders[idx] = fs
	n.section++
	return nil
}

func (n *ncaBuilder) addRomFSSection(sec *romfsSection, encrypted bool) error {
	idx := n.section
	start := len(n.data)
	for i := 0; i < 6; i++ {
		if err := appendFile(&n.data, sec.levels[i]); err != nil {
			return err
		}
	}
	padMedia(&n.data)
	end := len(n.data)
	n.entries[idx] = sectionEntry(start, end)
	fs := make([]byte, 0x200)
	binary.LittleEndian.PutUint16(fs[0:], 2)
	fs[3] = 3
	if encrypted {
		fs[4] = 3
	} else {
		fs[4] = 1
	}
	binary.LittleEndian.PutUint32(fs[0x8:], 0x43465649)
	binary.LittleEndian.PutUint32(fs[0xc:], 0x20000)
	binary.LittleEndian.PutUint32(fs[0x10:], 0x20)
	binary.LittleEndian.PutUint32(fs[0x14:], 0x7)
	var logical uint64
	for i := 0; i < 6; i++ {
		off := 0x18 + i*0x18
		binary.LittleEndian.PutUint64(fs[off:], logical)
		binary.LittleEndian.PutUint64(fs[off+0x8:], sec.levelSizes[i])
		binary.LittleEndian.PutUint32(fs[off+0x10:], 0x0e)
		logical += sec.levelSizes[i]
	}
	copy(fs[0xc8:], sec.masterHash[:])
	n.fsHeaders[idx] = fs
	n.section++
	return nil
}

func (n *ncaBuilder) finalize(ctx *buildContext, kek []byte, sign bool) (contentRecord, error) {
	header := make([]byte, ncaHeaderSize)
	binary.LittleEndian.PutUint32(header[0x200:], 0x3341434e)
	header[0x205] = n.contentType
	setKeyGeneration(header, ctx.keyGeneration)
	binary.LittleEndian.PutUint64(header[0x208:], uint64(len(n.data)))
	binary.LittleEndian.PutUint64(header[0x210:], ctx.titleID)
	binary.LittleEndian.PutUint32(header[0x21c:], ctx.sdkVersion)
	for i := 0; i < 4; i++ {
		copy(header[0x240+i*0x10:], n.entries[i][:])
		if n.fsHeaders[i] != nil {
			sum := sha256.Sum256(n.fsHeaders[i])
			copy(header[0x280+i*0x20:], sum[:])
			copy(header[0x400+i*0x200:], n.fsHeaders[i])
		}
	}
	copy(header[0x320:], ctx.keyArea)
	if err := encryptSectionData(n.data, header, ctx.keyArea); err != nil {
		return contentRecord{}, err
	}
	encryptECB(kek, header[0x300:0x340])
	if sign && !ctx.opt.NoSignNCASig2 {
		sig, err := signPSS(ctx.priv, header[0x200:0x400], ctx.opt.RandomPSS)
		if err != nil {
			return contentRecord{}, err
		}
		copy(header[0x100:0x200], sig)
	}
	if err := encryptHeader(ctx.header, header); err != nil {
		return contentRecord{}, err
	}
	copy(n.data[:ncaHeaderSize], header)
	sum := sha256.Sum256(n.data)
	var rec contentRecord
	copy(rec.hash[:], sum[:])
	copy(rec.ncaID[:], sum[:16])
	putUint48(rec.size[:], uint64(len(n.data)))
	switch n.contentType {
	case 0:
		rec.typ = 1
	case 1:
		rec.typ = 2
	case 2:
		rec.typ = 3
	}
	return rec, nil
}

func setKeyGeneration(header []byte, keyGeneration int) {
	if keyGeneration == 1 {
		return
	}
	header[0x206] = 0x02
	if keyGeneration != 2 {
		header[0x220] = byte(keyGeneration)
	}
}

func (rec contentRecord) cnmtRecord() cnmt.ContentRecord {
	return cnmt.ContentRecord{
		Hash:  rec.hash,
		NCAID: rec.ncaID,
		Size:  rec.size,
		Type:  rec.typ,
		ID:    rec.id,
	}
}

func encryptSectionData(data []byte, header []byte, key []byte) error {
	for i := 0; i < 4; i++ {
		entry := header[0x240+i*0x10:]
		start := int(binary.LittleEndian.Uint32(entry[0:]) * mediaSize)
		end := int(binary.LittleEndian.Uint32(entry[4:]) * mediaSize)
		if start == 0 || end == 0 || header[0x400+i*0x200+4] != 3 {
			continue
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return err
		}
		ctr := make([]byte, 16)
		ofs := uint64(start) >> 4
		for j := 0; j < 8; j++ {
			ctr[15-j] = byte(ofs & 0xff)
			ofs >>= 8
		}
		cipher.NewCTR(block, ctr).XORKeyStream(data[start:end], data[start:end])
	}
	return nil
}

func encryptHeader(headerKey []byte, header []byte) error {
	c, err := xtsn.NewXtsnCipher(headerKey[16:], headerKey[:16], 0x200)
	if err != nil {
		return err
	}
	c.Encrypt(header, 0)
	return nil
}

func encryptECB(key, data []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	for start := 0; start < len(data); start += aes.BlockSize {
		block.Encrypt(data[start:start+aes.BlockSize], data[start:start+aes.BlockSize])
	}
}

type deterministicReader struct {
	seed    [32]byte
	counter uint64
	buf     []byte
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(r.buf) == 0 {
			input := make([]byte, 40)
			copy(input, r.seed[:])
			binary.LittleEndian.PutUint64(input[32:], r.counter)
			sum := sha256.Sum256(input)
			r.buf = sum[:]
			r.counter++
		}
		copied := copy(p[n:], r.buf)
		n += copied
		r.buf = r.buf[copied:]
	}
	return n, nil
}

func signPSS(priv *rsa.PrivateKey, data []byte, random bool) ([]byte, error) {
	hash := sha256.Sum256(data)
	var reader io.Reader = &deterministicReader{seed: hash}
	if random {
		reader = cryptorand.Reader
	}
	return rsa.SignPSS(reader, priv, crypto.SHA256, hash[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
}

func createHashLevel(dstPath, srcPath string) (uint64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()
	dst, err := os.Create(dstPath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()
	buf := make([]byte, ivfcHashBlockSize)
	var size uint64
	for {
		n, readErr := io.ReadFull(src, buf)
		if readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return 0, readErr
		}
		sum := sha256.Sum256(buf[:n])
		if _, err := dst.Write(sum[:]); err != nil {
			return 0, err
		}
		size += sha256.Size
		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	padding := uint64(ivfcHashBlockSize) - (size % ivfcHashBlockSize)
	if padding != 0 {
		if _, err := dst.Write(make([]byte, padding)); err != nil {
			return 0, err
		}
		size += padding
	}
	return size, nil
}

func appendPadding(path string, block uint64) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	padding := block - (uint64(info.Size()) % block)
	if padding != 0 {
		_, err = f.Write(make([]byte, padding))
	}
	return err
}

func appendFile(dst *[]byte, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	*dst = append(*dst, data...)
	return nil
}

func padMedia(data *[]byte) {
	padding := mediaSize - (len(*data) % mediaSize)
	if padding != mediaSize {
		*data = append(*data, make([]byte, padding)...)
	}
}

func sectionEntry(start, end int) [16]byte {
	var out [16]byte
	binary.LittleEndian.PutUint32(out[0:], uint32(start/mediaSize))
	binary.LittleEndian.PutUint32(out[4:], uint32(end/mediaSize))
	out[8] = 1
	return out
}

func putUint48(dst []byte, v uint64) {
	for i := 0; i < 6; i++ {
		dst[i] = byte(v >> (8 * i))
	}
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func copyPathOrBytes(src string, data []byte, dst string) error {
	if src != "" {
		return copyFile(src, dst)
	}
	return os.WriteFile(dst, data, 0o644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDirContents(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func buildEntriesFromDir(dir string) ([]pfs0.BuildEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]pfs0.BuildEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		entries = append(entries, pfs0.BuildEntry{Name: entry.Name(), Path: filepath.Join(dir, entry.Name())})
	}
	return entries, nil
}
