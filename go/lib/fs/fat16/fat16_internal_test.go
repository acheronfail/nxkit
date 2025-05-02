package fat16

import (
	"testing"

	"github.com/acheronfail/nxkit/lib/fs/testutils"
	"github.com/stretchr/testify/assert"
)

// File.txt	FILE.TXT
// foo.tar.gz	FOOTAR~1.GZ
// .conf	CONF~1
// a+b=c	A_B_C~1
// 💩.png	3F04~1.PNG
// Asakura Otome.jpeg	ASAKUR~1.JPE
// Asakura Yume.jpeg	ASAKUR~2.JPE

func mockDirEntry(sfn, lfn string) DirectoryEntry {
	var name [11]byte
	copy(name[:], sfn)
	return DirectoryEntry{
		longFileName: lfn,
		fatDirectoryEntry: fatDirectoryEntry{
			DIR_Name: name,
			DIR_Attr: 0x20,
		},
	}
}

func TestShortFileName(t *testing.T) {
	t.Skip()

	fs, err := NewFromPath(testutils.DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	testcases := []struct {
		longFileName  string
		shortFileName string
	}{
		{"file.txt", "FILE.TXT"},
		{"foo.tar.gz", "FOOTAR~1.GZ"},
		{".conf", "CONF~1"},
		{"a+b=c", "A_B_C~1"},
		{"💩.png", "3F04~1.PNG"},
		{"Asakura Otome.jpeg", "ASAKUR~1.JPE"},
		{"Asakura Yume.jpeg", "ASAKUR~2.JPE"},
	}

	siblingEntries := []DirectoryEntry{}
	for _, tc := range testcases {
		siblingEntries = append(siblingEntries, mockDirEntry(tc.shortFileName, tc.longFileName))
	}

	for _, tc := range testcases {
		t.Run(tc.longFileName, func(t *testing.T) {
			shortName, err := fs.createShortName(tc.longFileName, siblingEntries)
			assert.Nil(t, err)
			assert.Equal(t, tc.shortFileName, shortName)
		})
	}
}
