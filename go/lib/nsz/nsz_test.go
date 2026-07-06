package nsz

import (
	"bytes"
	"context"
	"crypto/aes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/nca"
	"github.com/acheronfail/nxkit/lib/pfs0"
	"github.com/klauspost/compress/zstd"
)

func TestUpstreamSolidNCZFixture(t *testing.T) {
	expected := syntheticNCAWithPayload(upstreamSolidPayload())
	decompressed := decompressFixtureNCZ(t, "upstream_solid_tail.hex", expected)
	if !bytes.Equal(decompressed, expected) {
		t.Fatalf("decompressed upstream solid fixture differs from expected NCA")
	}
}

func TestUpstreamSolidNCZFixtureWithGap(t *testing.T) {
	expected := syntheticNCAWithPayload(upstreamSolidGapPayload())
	decompressed := decompressFixtureNCZ(t, "upstream_solid_gap_tail.hex", expected)
	if !bytes.Equal(decompressed, expected) {
		t.Fatalf("decompressed upstream solid gap fixture differs from expected NCA")
	}
}

func TestUpstreamBlockNCZFixture(t *testing.T) {
	expected := syntheticNCAWithPayload(upstreamBlockPayload())
	decompressed := decompressFixtureNCZ(t, "upstream_block_tail.hex", expected)
	if !bytes.Equal(decompressed, expected) {
		t.Fatalf("decompressed upstream block fixture differs from expected NCA")
	}
}

func TestUpstreamNSZContainerFixture(t *testing.T) {
	dir := t.TempDir()
	nszPath := filepath.Join(dir, "fixture.nsz")
	outPath := filepath.Join(dir, "fixture.nsp")

	ncaBytes := syntheticNCAWithPayload(upstreamSolidPayload())
	nczBytes := append(bytes.Clone(ncaBytes[:ncaHeaderSize]), fixtureHex(t, "upstream_solid_tail.hex")...)
	note := []byte("fixture generated from upstream NSZ NCZ/PFS0 format\n")
	if _, err := pfs0.Build(nszPath, []pfs0.BuildEntry{
		{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.ncz", Data: nczBytes},
		{Name: "readme.txt", Data: note},
	}); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := DecompressNSZWithProgressContext(context.Background(), nszPath, outPath, nil); err != nil {
		t.Fatalf("DecompressNSZWithProgressContext() error = %v", err)
	}
	assertPFS0Entry(t, outPath, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.nca", ncaBytes)
	assertPFS0Entry(t, outPath, "readme.txt", note)
}

func TestNCZSolidRoundTrip(t *testing.T) {
	source := syntheticNCA()
	var compressed bytes.Buffer
	if err := CompressNCZWithProgressContext(context.Background(), bytes.NewReader(source), int64(len(source)), &compressed, keys.Keys{}, nil); err != nil {
		t.Fatalf("CompressNCZWithProgressContext() error = %v", err)
	}

	var decompressed bytes.Buffer
	if err := DecompressNCZWithProgressContext(context.Background(), bytes.NewReader(compressed.Bytes()), int64(compressed.Len()), &decompressed, nil); err != nil {
		t.Fatalf("DecompressNCZWithProgressContext() error = %v", err)
	}

	if !bytes.Equal(decompressed.Bytes(), source) {
		t.Fatalf("decompressed NCZ differs from source NCA")
	}
}

func TestNCZBlockDecompress(t *testing.T) {
	payload := bytes.Repeat([]byte("block-data-"), 1800)
	source := syntheticNCAWithPayload(payload)
	block := makeBlockNCZ(t, source[:ncaHeaderSize], source[ncaHeaderSize:])

	var decompressed bytes.Buffer
	if err := DecompressNCZWithProgressContext(context.Background(), bytes.NewReader(block), int64(len(block)), &decompressed, nil); err != nil {
		t.Fatalf("DecompressNCZWithProgressContext() error = %v", err)
	}

	if !bytes.Equal(decompressed.Bytes(), source) {
		t.Fatalf("decompressed block NCZ differs from source NCA")
	}
}

func TestNSPCompressDecompressRoundTrip(t *testing.T) {
	dir := t.TempDir()
	nspPath := filepath.Join(dir, "input.nsp")
	nszPath := filepath.Join(dir, "input.nsz")
	roundTripPath := filepath.Join(dir, "roundtrip.nsp")

	sourceNCA := syntheticNCA()
	note := []byte("not compressed")
	if _, err := pfs0.Build(nspPath, []pfs0.BuildEntry{
		{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.nca", Data: sourceNCA},
		{Name: "note.txt", Data: note},
	}); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := CompressNSPWithProgressContext(context.Background(), nspPath, nszPath, keys.Keys{}, nil); err != nil {
		t.Fatalf("CompressNSPWithProgressContext() error = %v", err)
	}
	assertPFS0Entry(t, nszPath, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.ncz", nil)
	assertPFS0Entry(t, nszPath, "note.txt", note)

	if err := DecompressNSZWithProgressContext(context.Background(), nszPath, roundTripPath, nil); err != nil {
		t.Fatalf("DecompressNSZWithProgressContext() error = %v", err)
	}
	assertPFS0Entry(t, roundTripPath, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.nca", sourceNCA)
	assertPFS0Entry(t, roundTripPath, "note.txt", note)
}

func TestTitleKeysFromPFS0(t *testing.T) {
	dir := t.TempDir()
	nspPath := filepath.Join(dir, "ticket.nsp")
	rightsID := [0x10]byte{0x01, 0x02, 0x03}
	titleKey := bytes.Repeat([]byte{0x44}, aes.BlockSize)
	titleKek := bytes.Repeat([]byte{0x9a}, aes.BlockSize)

	if _, err := pfs0.Build(nspPath, []pfs0.BuildEntry{
		{Name: "test.tik", Data: syntheticTicket(t, rightsID, titleKey, titleKek)},
	}); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	file, err := os.Open(nspPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()
	fs, err := pfs0.NewPfs0(file)
	if err != nil {
		t.Fatalf("NewPfs0() error = %v", err)
	}

	titleKeys, err := titleKeysFromPFS0(fs, keys.Keys{Titlekek: [][]byte{titleKek}})
	if err != nil {
		t.Fatalf("titleKeysFromPFS0() error = %v", err)
	}
	if !bytes.Equal(titleKeys[rightsID], titleKey) {
		t.Fatalf("decrypted title key = %x, want %x", titleKeys[rightsID], titleKey)
	}
}

func decompressFixtureNCZ(t *testing.T, fixtureName string, expectedNCA []byte) []byte {
	t.Helper()

	nczBytes := append(bytes.Clone(expectedNCA[:ncaHeaderSize]), fixtureHex(t, fixtureName)...)
	var decompressed bytes.Buffer
	if err := DecompressNCZWithProgressContext(context.Background(), bytes.NewReader(nczBytes), int64(len(nczBytes)), &decompressed, nil); err != nil {
		t.Fatalf("DecompressNCZWithProgressContext(%s) error = %v", fixtureName, err)
	}
	return decompressed.Bytes()
}

func fixtureHex(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}
	compact := strings.Join(strings.Fields(string(raw)), "")
	data, err := hex.DecodeString(compact)
	if err != nil {
		t.Fatalf("DecodeString(%s) error = %v", name, err)
	}
	return data
}

func syntheticNCA() []byte {
	gap := bytes.Repeat([]byte{0x47}, 0x200)
	sectionA := bytes.Repeat([]byte("A"), 0x400)
	sectionB := bytes.Repeat([]byte("B"), 0x400)
	return syntheticNCAWithPayload(bytes.Join([][]byte{gap, sectionA, sectionB}, nil))
}

func upstreamSolidPayload() []byte {
	return append(bytes.Repeat([]byte("UPSTREAM-SOLID-FIXTURE\n"), 20), byteRange(64)...)
}

func upstreamSolidGapPayload() []byte {
	payload := []byte("GAP!")
	payload = append(payload, bytes.Repeat([]byte{0x5a}, 0x180)...)
	payload = append(payload, []byte("SECTION-A")...)
	payload = append(payload, byteRange(32)...)
	return payload
}

func upstreamBlockPayload() []byte {
	payload := bytes.Repeat([]byte("UPSTREAM-BLOCK-FIXTURE\n"), 1800)
	for range 12 {
		payload = append(payload, byteRange(256)...)
	}
	return payload
}

func byteRange(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

func syntheticNCAWithPayload(payload []byte) []byte {
	total := ncaHeaderSize + len(payload)
	out := make([]byte, total)
	binary.LittleEndian.PutUint32(out[0x200:], 0x3341434e)
	binary.LittleEndian.PutUint64(out[0x208:], uint64(total))

	startMedia := uint32(ncaHeaderSize / 0x200)
	endMedia := uint32((total + 0x1ff) / 0x200)
	binary.LittleEndian.PutUint32(out[0x240:], startMedia)
	binary.LittleEndian.PutUint32(out[0x244:], endMedia)
	out[0x400+2] = byte(nca.SectionFsTypePfs0)
	out[0x400+4] = byte(nca.CryptNone)

	copy(out[ncaHeaderSize:], payload)
	return out
}

func syntheticTicket(t *testing.T, rightsID [0x10]byte, titleKey, titleKek []byte) []byte {
	t.Helper()

	signatureSize := 0x100
	padding := 0x40 - ((signatureSize + 4) % 0x40)
	base := 4 + signatureSize + padding
	data := make([]byte, base+0x170)
	binary.LittleEndian.PutUint32(data[:4], 0x010004)

	block, err := aes.NewCipher(titleKek)
	if err != nil {
		t.Fatalf("aes.NewCipher() error = %v", err)
	}
	block.Encrypt(data[base+0x40:base+0x50], titleKey)
	data[base+0x145] = 1
	copy(data[base+0x160:base+0x170], rightsID[:])
	return data
}

func makeBlockNCZ(t *testing.T, header []byte, payload []byte) []byte {
	t.Helper()

	var out bytes.Buffer
	out.Write(header)
	if err := writeSectionHeader(&out, []nczSection{{
		offset:    ncaHeaderSize,
		size:      int64(len(payload)),
		cryptType: nca.CryptNone,
	}}); err != nil {
		t.Fatalf("writeSectionHeader() error = %v", err)
	}
	out.WriteString(nczBlockMagic)

	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("zstd.NewWriter() error = %v", err)
	}
	compressed := encoder.EncodeAll(payload, nil)
	encoder.Close()
	if len(compressed) >= len(payload) {
		compressed = payload
	}

	var fixed [16]byte
	fixed[0] = 2
	fixed[3] = 15
	binary.LittleEndian.PutUint32(fixed[4:], 1)
	binary.LittleEndian.PutUint64(fixed[8:], uint64(len(payload)))
	out.Write(fixed[:])
	if err := binary.Write(&out, binary.LittleEndian, uint32(len(compressed))); err != nil {
		t.Fatalf("binary.Write() error = %v", err)
	}
	out.Write(compressed)
	return out.Bytes()
}

func assertPFS0Entry(t *testing.T, path, name string, expected []byte) {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", path, err)
	}
	defer file.Close()

	fs, err := pfs0.NewPfs0(file)
	if err != nil {
		t.Fatalf("NewPfs0(%q) error = %v", path, err)
	}
	for _, entry := range fs.Entries() {
		if entry.Name() != name {
			continue
		}
		if expected == nil {
			return
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatalf("Open entry %q error = %v", name, err)
		}
		actual, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll entry %q error = %v", name, err)
		}
		if !bytes.Equal(actual, expected) {
			t.Fatalf("entry %q data differs", name)
		}
		return
	}
	t.Fatalf("entry %q not found in %q", name, path)
}
