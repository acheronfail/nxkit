package fat16

import (
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
