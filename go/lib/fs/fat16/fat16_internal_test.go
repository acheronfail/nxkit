package fat16

import (
	"strings"
	"testing"

	"github.com/acheronfail/nxkit/lib/fs/testutils"
	"github.com/stretchr/testify/assert"
)

func mockDirEntry(sfn, lfn string) DirectoryEntry {
	var name [11]byte
	copy(name[:], strings.ReplaceAll(sfn, ".", ""))
	return DirectoryEntry{
		longFileName: lfn,
		fatDirectoryEntry: fatDirectoryEntry{
			DIR_Name: name,
			DIR_Attr: 0x20,
		},
	}
}

func TestShortFileName(t *testing.T) {
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
		{"💩.png", "2FFA~1.PNG"},
		{"Asakura Otome.jpeg", "ASAKUR~1.JPE"},
		{"Asakura Yume.jpeg", "ASAKUR~2.JPE"},
	}

	for _, tc := range testcases {
		t.Run(tc.longFileName, func(t *testing.T) {
			siblingEntries := []DirectoryEntry{}
			for _, other := range testcases {
				if tc == other {
					continue
				}

				siblingEntries = append(siblingEntries, mockDirEntry(other.shortFileName, other.longFileName))
			}

			shortName, err := fs.createShortName(tc.longFileName, siblingEntries)
			assert.Nil(t, err)
			assert.Equal(t, tc.shortFileName, shortName)
		})
	}
}
