package fat16_test

import (
	"fmt"
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

	testSingleMkdirWorked := func(fs *fat16.FileSystem) {
		t.Helper()
		entries, err := fs.ReadDir("/mkdir/single")
		assert.Nil(t, err)
		assert.Len(t, entries, 3)
		shortNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "new"}, shortNames)
		longNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "new"}, longNames)

		assert.True(t, entries[2].IsDir())
		assert.Equal(t, entries[2].ShortName(), "new")
	}

	t.Run("mkdir /mkdir/single/new", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/single/new")
		assert.Nil(t, err)

		testSingleMkdirWorked(fs)

		newFs, err := fat16.NewFromPath(diskImagePath)
		assert.Nil(t, err)
		testSingleMkdirWorked(newFs)
	})

	t.Run("mkdir requiring new cluster", func(t *testing.T) {
		info := fs.Info()

		// keep adding directories until we go over the cluster size
		i := int64(0)
		for ; i*32 < info["bytesPerCluster"].(int64)+32; i++ {
			err := fs.Mkdir(fmt.Sprintf("/mkdir/many/%d", i))
			assert.Nil(t, err)
		}

		entries, err := fs.ReadDir("/mkdir/many")
		assert.Nil(t, err)
		entryNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Len(t, entryNames, int(i+2)) // +2 for . and ..
	})

	t.Run("mkdir /mkdir/long/this_name_exceeds_8_characters", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/long/this_name_exceeds_8_characters")
		assert.Nil(t, err)

		fmt.Println("long")
		entries, err := fs.ReadDir("/mkdir/long")
		assert.Nil(t, err)
		assert.Len(t, entries, 3)
		shortNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "THIS_N~1"}, shortNames)
		longNames := mapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "this_name_exceeds_8_characters"}, longNames)
	})
}
