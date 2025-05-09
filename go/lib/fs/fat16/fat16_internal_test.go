package fat16

import (
	"bytes"
	"testing"

	"github.com/acheronfail/nxkit/lib/fs/testutils"
	"github.com/stretchr/testify/assert"
)

type mockDirEntryNames struct {
	longFileName       string
	shortFileName      string
	shortFileNameBytes [11]byte
}

var (
	pad       = byte(0x20)
	mockNames = []mockDirEntryNames{
		{"file.txt", "FILE.TXT", [11]byte{0x46, 0x49, 0x4C, 0x45, pad, pad, pad, pad, 0x54, 0x58, 0x54}},
		{"foo.tar.gz", "FOOTAR~1.GZ", [11]byte{0x46, 0x4F, 0x4F, 0x54, 0x41, 0x52, 0x7E, 0x31, 0x47, 0x5A, pad}},
		{".conf", "CONF~1", [11]byte{0x43, 0x4F, 0x4E, 0x46, 0x7E, 0x31, pad, pad, pad, pad, pad}},
		{"a+b=c", "A_B_C~1", [11]byte{0x41, 0x5F, 0x42, 0x5F, 0x43, 0x7E, 0x31, pad, pad, pad, pad}},
		{"💩.png", "2FFA~1.PNG", [11]byte{0x32, 0x46, 0x46, 0x41, 0x7E, 0x31, pad, pad, 0x50, 0x4E, 0x47}},
		{"Asakura Otome.jpeg", "ASAKUR~1.JPE", [11]byte{0x41, 0x53, 0x41, 0x4B, 0x55, 0x52, 0x7E, 0x31, 0x4A, 0x50, 0x45}},
		{"Asakura Yume.jpeg", "ASAKUR~2.JPE", [11]byte{0x41, 0x53, 0x41, 0x4B, 0x55, 0x52, 0x7E, 0x32, 0x4A, 0x50, 0x45}},
		{"AFILEW~1.DAT", "AFILEW~1.DAT", [11]byte{0x41, 0x46, 0x49, 0x4C, 0x45, 0x57, 0x7E, 0x31, 0x44, 0x41, 0x54}},
	}
)

func mockDirEntry(names mockDirEntryNames) DirectoryEntry {
	return DirectoryEntry{
		longFileName: names.longFileName,
		fatDirectoryEntry: fatDirectoryEntry{
			DIR_Name: names.shortFileNameBytes,
			DIR_Attr: 0x20,
		},
	}
}

func TestReadShortFileName(t *testing.T) {
	fs, err := NewFromPath(testutils.DiskImagePath)
	assert.Nil(t, err)
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
	fs, err := NewFromPath(testutils.DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	for _, tc := range mockNames {
		t.Run(tc.longFileName, func(t *testing.T) {
			siblingEntries := []DirectoryEntry{}
			for _, other := range mockNames {
				if tc == other {
					continue
				}

				siblingEntries = append(siblingEntries, mockDirEntry(other))
			}

			sfn, sfnBytes, err := fs.createShortName(tc.longFileName, siblingEntries)
			assert.Nil(t, err)
			assert.Equal(t, tc.shortFileName, sfn)
			assert.Equal(t, tc.shortFileNameBytes, sfnBytes)
		})
	}
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

	assertEntry := func(t *testing.T, entry DirectoryEntry) {
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
