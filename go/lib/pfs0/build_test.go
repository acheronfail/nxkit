package pfs0

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWritesExpectedLayoutAndRoundTrips(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "archive.pfs0")
	entries := []BuildEntry{
		{Name: "alpha.bin", Data: []byte{0x01, 0x02, 0x03}},
		{Name: "empty.dat", Data: []byte{}},
		{Name: "gamma.txt", Data: []byte{0xaa}},
	}

	size, err := Build(outPath, entries)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	data := readFile(t, outPath)
	if got, want := size, uint64(len(data)); got != want {
		t.Fatalf("reported size = %d, want %d", got, want)
	}
	if got, want := len(data), headerSize+len(entries)*entryListSize+0x20+4; got != want {
		t.Fatalf("file size = %d, want %d", got, want)
	}

	assertHeader(t, data, len(entries), 0x20)
	assertEntry(t, data, 0, 0, 3, 0)
	assertEntry(t, data, 1, 3, 0, 10)
	assertEntry(t, data, 2, 3, 1, 20)

	stringTableOffset := headerSize + len(entries)*entryListSize
	stringTable := data[stringTableOffset : stringTableOffset+0x20]
	wantStringTable := append([]byte("alpha.bin\x00empty.dat\x00gamma.txt\x00"), make([]byte, 2)...)
	if !bytes.Equal(stringTable, wantStringTable) {
		t.Fatalf("string table = %x, want %x", stringTable, wantStringTable)
	}

	payload := data[stringTableOffset+0x20:]
	if want := []byte{0x01, 0x02, 0x03, 0xaa}; !bytes.Equal(payload, want) {
		t.Fatalf("payload = %x, want %x", payload, want)
	}

	assertRoundTrip(t, data, map[string][]byte{
		"alpha.bin": {0x01, 0x02, 0x03},
		"empty.dat": {},
		"gamma.txt": {0xaa},
	})
}

func TestBuildSupportsFileAndDataEntries(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "from-disk.bin")
	if err := os.WriteFile(filePath, []byte("from disk"), 0o644); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmpDir, "mixed.pfs0")
	_, err := Build(outPath, []BuildEntry{
		{Name: "memory.txt", Data: []byte("from memory")},
		{Name: "disk.txt", Path: filePath},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	assertRoundTrip(t, readFile(t, outPath), map[string][]byte{
		"memory.txt": []byte("from memory"),
		"disk.txt":   []byte("from disk"),
	})
}

func TestBuildFromDirSortsFilesAndSkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	inDir := filepath.Join(tmpDir, "input")
	if err := os.Mkdir(inDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "zeta.txt"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "alpha.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(inDir, "ignored"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "middle.txt"), []byte("m"), 0o644); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmpDir, "dir.pfs0")
	if _, err := BuildFromDir(inDir, outPath); err != nil {
		t.Fatalf("BuildFromDir failed: %v", err)
	}

	data := readFile(t, outPath)
	fs, err := NewPfs0(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewPfs0 failed: %v", err)
	}

	entries := fs.Entries()
	gotNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		gotNames = append(gotNames, entry.Name())
	}
	wantNames := []string{"alpha.txt", "middle.txt", "zeta.txt"}
	if strings.Join(gotNames, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("entry names = %v, want %v", gotNames, wantNames)
	}

	assertRoundTrip(t, data, map[string][]byte{
		"alpha.txt":  []byte("a"),
		"middle.txt": []byte("m"),
		"zeta.txt":   []byte("z"),
	})
}

func TestBuildRejectsDirectoryAndMissingPathEntries(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "dir")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Build(filepath.Join(tmpDir, "dir-entry.pfs0"), []BuildEntry{{Name: "dir", Path: dirPath}}); err == nil {
		t.Fatal("Build succeeded for directory entry")
	}
	if _, err := Build(filepath.Join(tmpDir, "missing-entry.pfs0"), []BuildEntry{{Name: "missing", Path: filepath.Join(tmpDir, "missing")}}); err == nil {
		t.Fatal("Build succeeded for missing entry path")
	}
}

func TestBuildEmptyArchive(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "empty.pfs0")
	size, err := Build(outPath, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if got, want := size, uint64(headerSize); got != want {
		t.Fatalf("reported size = %d, want %d", got, want)
	}

	data := readFile(t, outPath)
	assertHeader(t, data, 0, 0)
	fs, err := NewPfs0(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewPfs0 failed: %v", err)
	}
	if got := len(fs.Entries()); got != 0 {
		t.Fatalf("entry count = %d, want 0", got)
	}
}

func TestCreateHashTableHashesBlocksAndMasterHashIgnoresPadding(t *testing.T) {
	tmpDir := t.TempDir()
	pfs0Path := filepath.Join(tmpDir, "source.pfs0")
	hashPath := filepath.Join(tmpDir, "source.hashtable")
	source := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := os.WriteFile(pfs0Path, source, 0o644); err != nil {
		t.Fatal(err)
	}

	hashTableSize, pfs0Offset, err := CreateHashTable(pfs0Path, hashPath, 4)
	if err != nil {
		t.Fatalf("CreateHashTable failed: %v", err)
	}
	if got, want := hashTableSize, uint64(sha256.Size*3); got != want {
		t.Fatalf("hashTableSize = %d, want %d", got, want)
	}
	if got, want := pfs0Offset, uint64(hashTablePadding); got != want {
		t.Fatalf("pfs0Offset = %d, want %d", got, want)
	}

	hashTable := readFile(t, hashPath)
	if got, want := len(hashTable), hashTablePadding; got != want {
		t.Fatalf("hash table file size = %d, want %d", got, want)
	}

	expectedHashes := appendHash(nil, source[0:4])
	expectedHashes = appendHash(expectedHashes, source[4:8])
	expectedHashes = appendHash(expectedHashes, source[8:10])
	if !bytes.Equal(hashTable[:len(expectedHashes)], expectedHashes) {
		t.Fatalf("hash table prefix = %x, want %x", hashTable[:len(expectedHashes)], expectedHashes)
	}
	if !bytes.Equal(hashTable[len(expectedHashes):], make([]byte, hashTablePadding-len(expectedHashes))) {
		t.Fatal("hash table padding is not zero-filled")
	}

	master, err := MasterHash(hashPath, hashTableSize)
	if err != nil {
		t.Fatalf("MasterHash failed: %v", err)
	}
	expectedMaster := sha256.Sum256(expectedHashes)
	if master != expectedMaster {
		t.Fatalf("master hash = %x, want %x", master, expectedMaster)
	}
}

func TestCreateHashTableAddsFullPaddingBlockWhenAligned(t *testing.T) {
	tmpDir := t.TempDir()
	pfs0Path := filepath.Join(tmpDir, "source.pfs0")
	hashPath := filepath.Join(tmpDir, "source.hashtable")
	source := bytes.Repeat([]byte{0xab}, 64)
	if err := os.WriteFile(pfs0Path, source, 0o644); err != nil {
		t.Fatal(err)
	}

	hashTableSize, pfs0Offset, err := CreateHashTable(pfs0Path, hashPath, 4)
	if err != nil {
		t.Fatalf("CreateHashTable failed: %v", err)
	}
	if got, want := hashTableSize, uint64(hashTablePadding); got != want {
		t.Fatalf("hashTableSize = %d, want %d", got, want)
	}
	if got, want := pfs0Offset, uint64(hashTablePadding*2); got != want {
		t.Fatalf("pfs0Offset = %d, want %d", got, want)
	}
	if got, want := len(readFile(t, hashPath)), hashTablePadding*2; got != want {
		t.Fatalf("hash table file size = %d, want %d", got, want)
	}
}

func TestCreateHashTableForEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	pfs0Path := filepath.Join(tmpDir, "empty.pfs0")
	hashPath := filepath.Join(tmpDir, "empty.hashtable")
	if err := os.WriteFile(pfs0Path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	hashTableSize, pfs0Offset, err := CreateHashTable(pfs0Path, hashPath, 4)
	if err != nil {
		t.Fatalf("CreateHashTable failed: %v", err)
	}
	if hashTableSize != 0 {
		t.Fatalf("hashTableSize = %d, want 0", hashTableSize)
	}
	if got, want := pfs0Offset, uint64(hashTablePadding); got != want {
		t.Fatalf("pfs0Offset = %d, want %d", got, want)
	}
	if got, want := len(readFile(t, hashPath)), hashTablePadding; got != want {
		t.Fatalf("hash table file size = %d, want %d", got, want)
	}

	master, err := MasterHash(hashPath, 0)
	if err != nil {
		t.Fatalf("MasterHash failed: %v", err)
	}
	if want := sha256.Sum256(nil); master != want {
		t.Fatalf("master hash = %x, want %x", master, want)
	}
}

func TestNewPfs0RejectsInvalidMagic(t *testing.T) {
	data := make([]byte, headerSize)
	binary.LittleEndian.PutUint32(data[0:], 0xdeadbeef)

	if _, err := NewPfs0(bytes.NewReader(data)); err == nil {
		t.Fatal("NewPfs0 succeeded for invalid magic")
	}
}

func TestNewPfs0RejectsTruncatedHeader(t *testing.T) {
	if _, err := NewPfs0(bytes.NewReader(make([]byte, headerSize-1))); err == nil {
		t.Fatal("NewPfs0 succeeded for truncated header")
	}
}

func TestNewPfs0RejectsUnterminatedStringTableEntry(t *testing.T) {
	data := make([]byte, headerSize+entryListSize+3)
	binary.LittleEndian.PutUint32(data[0x0:], Psf0Magic)
	binary.LittleEndian.PutUint32(data[0x4:], 1)
	binary.LittleEndian.PutUint32(data[0x8:], 3)
	copy(data[headerSize+entryListSize:], []byte("abc"))

	if _, err := NewPfs0(bytes.NewReader(data)); err == nil {
		t.Fatal("NewPfs0 succeeded for unterminated string table")
	}
}

func TestNewPfs0RejectsStringTableOffsetOutsideTable(t *testing.T) {
	data := make([]byte, headerSize+entryListSize+4)
	binary.LittleEndian.PutUint32(data[0x0:], Psf0Magic)
	binary.LittleEndian.PutUint32(data[0x4:], 1)
	binary.LittleEndian.PutUint32(data[0x8:], 4)
	binary.LittleEndian.PutUint32(data[headerSize+0x10:], 4)
	copy(data[headerSize+entryListSize:], []byte("abc\x00"))

	if _, err := NewPfs0(bytes.NewReader(data)); err == nil {
		t.Fatal("NewPfs0 succeeded for out-of-bounds string table offset")
	}
}

func assertHeader(t *testing.T, data []byte, numEntries int, stringTableSize uint32) {
	t.Helper()
	if len(data) < headerSize {
		t.Fatalf("data is smaller than header: %d", len(data))
	}
	if got := binary.LittleEndian.Uint32(data[0x0:]); got != Psf0Magic {
		t.Fatalf("magic = 0x%x, want 0x%x", got, Psf0Magic)
	}
	if got := binary.LittleEndian.Uint32(data[0x4:]); got != uint32(numEntries) {
		t.Fatalf("num files = %d, want %d", got, numEntries)
	}
	if got := binary.LittleEndian.Uint32(data[0x8:]); got != stringTableSize {
		t.Fatalf("string table size = %d, want %d", got, stringTableSize)
	}
	if got := binary.LittleEndian.Uint32(data[0xc:]); got != 0 {
		t.Fatalf("reserved header value = %d, want 0", got)
	}
}

func assertEntry(t *testing.T, data []byte, index int, offset, size uint64, stringOffset uint32) {
	t.Helper()
	start := headerSize + index*entryListSize
	entry := data[start : start+entryListSize]
	if got := binary.LittleEndian.Uint64(entry[0x0:]); got != offset {
		t.Fatalf("entry %d offset = %d, want %d", index, got, offset)
	}
	if got := binary.LittleEndian.Uint64(entry[0x8:]); got != size {
		t.Fatalf("entry %d size = %d, want %d", index, got, size)
	}
	if got := binary.LittleEndian.Uint32(entry[0x10:]); got != stringOffset {
		t.Fatalf("entry %d string offset = %d, want %d", index, got, stringOffset)
	}
	if got := binary.LittleEndian.Uint32(entry[0x14:]); got != 0 {
		t.Fatalf("entry %d reserved value = %d, want 0", index, got)
	}
}

func assertRoundTrip(t *testing.T, data []byte, expected map[string][]byte) {
	t.Helper()
	fs, err := NewPfs0(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewPfs0 failed: %v", err)
	}

	entries := fs.Entries()
	if len(entries) != len(expected) {
		t.Fatalf("entry count = %d, want %d", len(entries), len(expected))
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		want, ok := expected[entry.Name()]
		if !ok {
			t.Fatalf("unexpected entry %q", entry.Name())
		}
		if entry.Size() != uint64(len(want)) {
			t.Fatalf("entry %q size = %d, want %d", entry.Name(), entry.Size(), len(want))
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatalf("Open(%q) failed: %v", entry.Name(), err)
		}
		got, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll(%q) failed: %v", entry.Name(), err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("entry %q data = %x, want %x", entry.Name(), got, want)
		}
		seen[entry.Name()] = true
	}
	for name := range expected {
		if !seen[name] {
			t.Fatalf("missing entry %q", name)
		}
	}
}

func appendHash(dst []byte, data []byte) []byte {
	sum := sha256.Sum256(data)
	return append(dst, sum[:]...)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
