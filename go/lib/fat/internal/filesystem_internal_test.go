package internal

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	"github.com/acheronfail/nxkit/lib/fat/testdata"
	"github.com/stretchr/testify/assert"
)

type mockDirEntryNames struct {
	longFileName       string
	shortFileName      string
	shortFileNameBytes [11]byte
	ntRes              uint8
	lfnCount           int
}

var (
	pad       = byte(0x20)
	mockNames = []mockDirEntryNames{
		{"lower.low", "lower.low", [11]byte{0x4c, 0x4f, 0x57, 0x45, 0x52, pad, pad, pad, 0x4c, 0x4f, 0x57}, 0x18, 0},
		{"lower.UPP", "lower.UPP", [11]byte{0x4c, 0x4f, 0x57, 0x45, 0x52, pad, pad, pad, 0x55, 0x50, 0x50}, 0x08, 0},
		{"UPPER.low", "UPPER.low", [11]byte{0x55, 0x50, 0x50, 0x45, 0x52, pad, pad, pad, 0x4c, 0x4f, 0x57}, 0x10, 0},
		{"UPPER.UPP", "UPPER.UPP", [11]byte{0x55, 0x50, 0x50, 0x45, 0x52, pad, pad, pad, 0x55, 0x50, 0x50}, 0x00, 0},
		{"MiXeD.mIx", "MIXED.MIX", [11]byte{0x4d, 0x49, 0x58, 0x45, 0x44, pad, pad, pad, 0x4d, 0x49, 0x58}, 0x00, 1},
		{"file.txt", "file.txt", [11]byte{0x46, 0x49, 0x4C, 0x45, pad, pad, pad, pad, 0x54, 0x58, 0x54}, 0x18, 0},
		{"foo.tar.gz", "FOOTAR~1.GZ", [11]byte{0x46, 0x4F, 0x4F, 0x54, 0x41, 0x52, 0x7E, 0x31, 0x47, 0x5A, pad}, 0x00, 1},
		{".conf", "CONF~1", [11]byte{0x43, 0x4F, 0x4E, 0x46, 0x7E, 0x31, pad, pad, pad, pad, pad}, 0x00, 1},
		{"a+b=c", "A_B_C~1", [11]byte{0x41, 0x5F, 0x42, 0x5F, 0x43, 0x7E, 0x31, pad, pad, pad, pad}, 0x00, 1},
		{"💩.png", "2FFA~1.PNG", [11]byte{0x32, 0x46, 0x46, 0x41, 0x7E, 0x31, pad, pad, 0x50, 0x4E, 0x47}, 0x00, 1},
		{"Asakura Otome.jpeg", "ASAKUR~1.JPE", [11]byte{0x41, 0x53, 0x41, 0x4B, 0x55, 0x52, 0x7E, 0x31, 0x4A, 0x50, 0x45}, 0x00, 2},
		{"Asakura Yume.jpeg", "ASAKUR~2.JPE", [11]byte{0x41, 0x53, 0x41, 0x4B, 0x55, 0x52, 0x7E, 0x32, 0x4A, 0x50, 0x45}, 0x00, 2},
		{"AFILEW~1.DAT", "AFILEW~1.DAT", [11]byte{0x41, 0x46, 0x49, 0x4C, 0x45, 0x57, 0x7E, 0x31, 0x44, 0x41, 0x54}, 0x00, 0},
	}
)

func mockDirEntry(names mockDirEntryNames) Entry {
	return Entry{
		longFileName: names.longFileName,
		fatDirectoryEntry: fatDirectoryEntry{
			DIR_Name:  names.shortFileNameBytes,
			DIR_Attr:  0x20,
			DIR_NTRes: names.ntRes,
		},
	}
}

type mockFatTable struct{}

func (m mockFatTable) GetClusterTarget(_ uint16) uint16                    { panic("unused") }
func (m mockFatTable) IsEoc(_ uint16) bool                                 { panic("unused") }
func (m mockFatTable) GetMaxCluster() uint16                               { panic("unused") }
func (m mockFatTable) GetEoc() uint16                                      { panic("unused") }
func (m mockFatTable) WriteClusterTarget(_ *FileSystem, _, _ uint16) error { panic("unused") }
func mockParseFatTable(_ []byte) FatTable                                  { return mockFatTable{} }

func mockGetRootDirectoryBytes(_ *FileSystem) ([]byte, error) { panic("unused") }

func createFs(t *testing.T) *FileSystem {
	t.Helper()

	fs, err := NewFileSystemFromPath(
		testdata.Fat16DiskImagePath,
		mockParseFatTable,
		mockGetRootDirectoryBytes,
	)

	assert.Nil(t, err)
	return fs
}

func TestNumEntriesRequired(t *testing.T) {
	fs := createFs(t)
	defer fs.Close()

	for _, tc := range mockNames {
		t.Run(tc.longFileName, func(t *testing.T) {
			n := fs.numDirectoryEntriesRequired(tc.longFileName)
			assert.Equal(t, tc.lfnCount+1, n)
		})
	}
}

func TestReadShortFileName(t *testing.T) {
	fs := createFs(t)
	defer fs.Close()

	for _, tc := range mockNames {
		t.Run(tc.longFileName, func(t *testing.T) {
			entry := mockDirEntry(tc)
			assert.Equal(t, tc.shortFileName, entry.ShortName())
			assert.Equal(t, tc.longFileName, entry.LongName())
		})
	}
}

func TestCreateShortFileName(t *testing.T) {
	fs := createFs(t)
	defer fs.Close()

	// make the random number generator deterministic for the tests
	rng := rand.New(rand.NewSource(0))
	fs.randIntn = func(n int) int {
		return rng.Intn(n)
	}

	for _, tc := range mockNames {
		t.Run(tc.longFileName, func(t *testing.T) {
			siblingEntries := []Entry{}
			for _, other := range mockNames {
				if tc == other {
					continue
				}

				siblingEntries = append(siblingEntries, mockDirEntry(other))
			}

			sfn, sfnBytes, ntRes := fs.createShortName(tc.longFileName, siblingEntries)
			assert.Equal(t, tc.ntRes, ntRes)
			assert.Equal(t, strings.ToUpper(tc.shortFileName), sfn)
			assert.Equal(t, tc.shortFileNameBytes, sfnBytes)
		})
	}
}

func TestCreateLongFileNameEntries(t *testing.T) {
	fs := createFs(t)
	defer fs.Close()

	t.Run("lfn - 1 entry mixed case", func(t *testing.T) {
		name := "MiXeD.mIx"
		entries := fs.createLongFileNameEntries(name, 0)
		assert.Len(t, entries, 1)
		assert.Equal(t, fs.numDirectoryEntriesRequired(name)-1, len(entries))
	})

	t.Run("lfn - 1 entry 13 chars", func(t *testing.T) {
		name := "1234567890123"
		entries := fs.createLongFileNameEntries(name, 0)
		assert.Len(t, entries, 1)
		assert.Equal(t, fs.numDirectoryEntriesRequired(name)-1, len(entries))
	})

	t.Run("lfn - 2 entries", func(t *testing.T) {
		name := "longer than 13 chars"
		entries := fs.createLongFileNameEntries(name, 0)
		assert.Len(t, entries, 2)
		assert.Equal(t, fs.numDirectoryEntriesRequired(name)-1, len(entries))
	})

	// TODO: tests for multi-byte unicode characters that would break if we indexed raw utf8
}

func TestReadDirectoryEntries(t *testing.T) {
	entryBytes := []byte{
		// DIR_Name: 11 bytes
		0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b,
		// DIR_Attr: 1 byte
		0x00,
		// DIR_NTRes: 1 byte
		0x00,
		// DIR_CrtTimeTenth: 1 byte
		0x00,
		// DIR_CrtTime: 2 bytes
		0x00, 0x00,
		// DIR_CrtDate: 2 bytes
		0x00, 0x00,
		// DIR_LstAccDate: 2 bytes
		0x00, 0x00,
		// DIR_FstClusHI: 2 bytes
		0x00, 0x00,
		// DIR_WrtTime: 2 bytes
		0x00, 0x00,
		// DIR_WrtDate: 2 bytes
		0x00, 0x00,
		// DIR_FstClusLO: 2 bytes
		0x00, 0x00,
		// DIR_FileSize: 4 bytes
		0x00, 0x00, 0x00, 0x00,
	}

	entryChecksum := calculateShortNameChecksum(entryBytes[:])

	assertEntry := func(t *testing.T, entry Entry) {
		assert.Equal(t, "ABCDEFGH.IJK", entry.ShortName())
		assert.False(t, entry.IsDir())
	}

	t.Run("read single entry", func(t *testing.T) {
		entries, err := readDirectoryEntries(entryBytes)
		assert.Nil(t, err)
		assert.Len(t, entries, 1)
		assertEntry(t, entries[0])
	})

	t.Run("read entry with 1 lfn", func(t *testing.T) {
		data := bytes.Join(
			[][]byte{
				{
					// LDIR_Ord: 1 byte
					0x41,
					// LDIR_Name1: 10 bytes (utf-16)
					0x30, 0x00, 0x31, 0x00, 0x32, 0x00, 0x33, 0x00, 0x34, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum,
					// LDIR_Name2: 12 bytes (utf-16)
					0x35, 0x00, 0x36, 0x00, 0x37, 0x00, 0x38, 0x00, 0x39, 0x00, 0x41, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0x42, 0x00, 0x43, 0x00,
				},
				entryBytes,
			},
			[]byte{},
		)
		entries, err := readDirectoryEntries(data)
		assert.Nil(t, err)
		assert.Len(t, entries, 1)
		assertEntry(t, entries[0])
		assert.Equal(t, "0123456789ABC", entries[0].longFileName)
	})

	t.Run("read entry with 3 lfns", func(t *testing.T) {
		data := bytes.Join(
			[][]byte{
				{
					// LDIR_Ord: 1 byte
					0x43,
					// LDIR_Name1: 10 bytes (utf-16)
					0x51, 0x00, 0x52, 0x00, 0x53, 0x00, 0x54, 0x00, 0x55, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum,
					// LDIR_Name2: 12 bytes (utf-16)
					0x56, 0x00, 0x57, 0x00, 0x58, 0x00, 0x59, 0x00, 0x5a, 0x00, 0x00, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0xff, 0xff, 0xff, 0xff,
				},
				{
					// LDIR_Ord: 1 byte
					0x02,
					// LDIR_Name1: 10 bytes (utf-16)
					0x44, 0x00, 0x45, 0x00, 0x46, 0x00, 0x47, 0x00, 0x48, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum,
					// LDIR_Name2: 12 bytes (utf-16)
					0x49, 0x00, 0x4a, 0x00, 0x4b, 0x00, 0x4c, 0x00, 0x4d, 0x00, 0x4e, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0x4f, 0x00, 0x50, 0x00,
				},
				{
					// LDIR_Ord: 1 byte
					0x01,
					// LDIR_Name1: 10 bytes (utf-16)
					0x30, 0x00, 0x31, 0x00, 0x32, 0x00, 0x33, 0x00, 0x34, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum,
					// LDIR_Name2: 12 bytes (utf-16)
					0x35, 0x00, 0x36, 0x00, 0x37, 0x00, 0x38, 0x00, 0x39, 0x00, 0x41, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0x42, 0x00, 0x43, 0x00,
				},
				entryBytes,
			},
			[]byte{},
		)
		entries, err := readDirectoryEntries(data)
		assert.Nil(t, err)
		assert.Len(t, entries, 1)
		assertEntry(t, entries[0])
		assert.Equal(t, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ", entries[0].longFileName)
	})

	t.Run("read entry with bad lfn", func(t *testing.T) {
		data := bytes.Join(
			[][]byte{
				{
					// LDIR_Ord: 1 byte
					0x41,
					// LDIR_Name1: 10 bytes (utf-16)
					0x41, 0x00, 0x42, 0x00, 0x43, 0x00, 0x44, 0x00, 0x45, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum + 1, // NOTE: incorrect checksum
					// LDIR_Name2: 12 bytes (utf-16)
					0x46, 0x00, 0x47, 0x00, 0x48, 0x00, 0x49, 0x00, 0x4a, 0x00, 0x4b, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0x4c, 0x00, 0x4d, 0x00,
				},
				entryBytes,
			},
			[]byte{},
		)
		entries, err := readDirectoryEntries(data)
		assert.Nil(t, err)
		assert.Len(t, entries, 1)
		assertEntry(t, entries[0])
		// should be empty, since checksum didn't match
		assert.Equal(t, "", entries[0].longFileName)
	})

	t.Run("read entry with bad lfns", func(t *testing.T) {
		data := bytes.Join(
			[][]byte{
				// invalid "BBBBBBBBBBBBB"
				{
					// LDIR_Ord: 1 byte
					0x41,
					// LDIR_Name1: 10 bytes (utf-16)
					0x42, 0x00, 0x42, 0x00, 0x42, 0x00, 0x42, 0x00, 0x42, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum + 1, // NOTE: incorrect checksum
					// LDIR_Name2: 12 bytes (utf-16)
					0x42, 0x00, 0x42, 0x00, 0x42, 0x00, 0x42, 0x00, 0x42, 0x00, 0x42, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0x42, 0x00, 0x42, 0x00,
				},
				// valid "GGGGGGGGGGGG"
				{
					// LDIR_Ord: 1 byte
					0x41,
					// LDIR_Name1: 10 bytes (utf-16)
					0x47, 0x00, 0x47, 0x00, 0x47, 0x00, 0x47, 0x00, 0x47, 0x00,
					// LDIR_Attr: 1 byte
					0x0F,
					// LDIR_Type: 1 byte
					0x00,
					// LDIR_Chksum: 1 byte
					entryChecksum,
					// LDIR_Name2: 12 bytes (utf-16)
					0x47, 0x00, 0x47, 0x00, 0x47, 0x00, 0x47, 0x00, 0x47, 0x00, 0x47, 0x00,
					// LDIR_FstClusLO: 2 bytes
					0x00, 0x00,
					// LDIR_Name3: 4 bytes (utf-16)
					0x47, 0x00, 0x47, 0x00,
				},
				entryBytes,
			},
			[]byte{},
		)
		entries, err := readDirectoryEntries(data)
		assert.Nil(t, err)
		assert.Len(t, entries, 1)
		assertEntry(t, entries[0])
		assert.Equal(t, "GGGGGGGGGGGGG", entries[0].longFileName)
	})
}
