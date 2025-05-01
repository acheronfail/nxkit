package fat16_test

import (
	"testing"

	"github.com/acheronfail/nxkit/lib/fs/fat16"
	"github.com/stretchr/testify/assert"
)

const (
	diskImagePath = "../fixtures/fat16/disk.img"
)

func mapSlice[T any, U any](things []T, f func(T) U) []U {
	mapped := make([]U, len(things))
	for i, thing := range things {
		mapped[i] = f(thing)
	}
	return mapped
}

func TestReadDir(t *testing.T) {
	fs, err := fat16.NewFromPath(diskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("path:/", func(t *testing.T) {
		entries, err := fs.ReadDir("/")
		assert.Nil(t, err)
		assert.Len(t, entries, 7)
		shortNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{"dir", "INFO.TXT", "AFILEW~1.DAT", "ANOTHE~1", "lower83", "mkdir", "FAT16-TEST"}, shortNames)
		longNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{"dir", "INFO.TXT", "a file with a long name.dat", "another file", "lower83", "mkdir", "FAT16-TEST"}, longNames)

		assert.True(t, entries[0].IsDir())
		assert.True(t, !entries[1].IsDir())
		assert.True(t, !entries[2].IsDir())
		assert.True(t, !entries[3].IsDir())
		assert.True(t, entries[4].IsDir())
		assert.True(t, entries[5].IsDir())
		assert.True(t, entries[6].IsVolumeId())
	})

	t.Run("path:/dir", func(t *testing.T) {
		entries, err := fs.ReadDir("/dir")
		assert.Nil(t, err)
		assert.Len(t, entries, 79)
	})

	t.Run("path:/dir/subdir", func(t *testing.T) {
		entries, err := fs.ReadDir("/dir/subdir")
		assert.Nil(t, err)
		assert.Len(t, entries, 3)
		shortNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "SOME_L~1"}, shortNames)
		longNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "some_long_embedded_nameא"}, longNames)
	})

	t.Run("path:/lower83", func(t *testing.T) {
		entries, err := fs.ReadDir("/lower83")
		assert.Nil(t, err)
		assert.Len(t, entries, 6)
		shortNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "lower.low", "lower.UPP", "UPPER.low", "UPPER.UPP"}, shortNames)
	})
}

func TestMkdir(t *testing.T) {
	fs, err := fat16.NewFromPath(diskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	testMkdirWorked := func(fs *fat16.FileSystem) {
		entries, err := fs.ReadDir("/mkdir")
		assert.Nil(t, err)
		assert.Len(t, entries, 3)
		shortNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "new"}, shortNames)
		longNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "new"}, longNames)

		assert.True(t, entries[2].IsDir())
		assert.Equal(t, entries[2].ShortName(), "new")
	}

	t.Run("mkdir /mkdir/new", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/new")
		assert.Nil(t, err)

		testMkdirWorked(fs)

		newFs, err := fat16.NewFromPath(diskImagePath)
		assert.Nil(t, err)
		testMkdirWorked(newFs)
	})
}
