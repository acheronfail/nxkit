package fat16_test

import (
	"fmt"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/acheronfail/nxkit/lib/fs/fat16"
	"github.com/acheronfail/nxkit/lib/fs/testutils"
	"github.com/acheronfail/nxkit/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestReadDir(t *testing.T) {
	fs, err := fat16.NewFromPath(testutils.DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("path:/", func(t *testing.T) {
		entries, err := fs.ReadDir("/")
		assert.Nil(t, err)
		assert.Len(t, entries, 9)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{"dir", "empty.bin", "INFO.TXT", "AFILEW~1.DAT", "ANOTHE~1", "lower83", "mkdir", "write", "FAT16-TEST"}, shortNames) // TODO
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{"dir", "empty.bin", "INFO.TXT", "a file with a long name.dat", "another file", "lower83", "mkdir", "write", "FAT16-TEST"}, longNames) // TODO

		assert.True(t, entries[0].IsDir())
		assert.True(t, !entries[1].IsDir())
		assert.True(t, !entries[2].IsDir())
		assert.True(t, !entries[3].IsDir())
		assert.True(t, !entries[4].IsDir())
		assert.True(t, entries[5].IsDir())
		assert.True(t, entries[6].IsDir())
		assert.True(t, entries[7].IsDir())
		assert.True(t, entries[8].IsVolumeId())
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
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "SOME_L~1"}, shortNames)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "some_long_embedded_nameא"}, longNames)
	})

	t.Run("path:/lower83", func(t *testing.T) {
		entries, err := fs.ReadDir("/lower83")
		assert.Nil(t, err)
		assert.Len(t, entries, 6)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "lower.low", "lower.UPP", "UPPER.low", "UPPER.UPP"}, shortNames)
	})

	t.Run("path:/not_here", func(t *testing.T) {
		_, err := fs.ReadDir("/not_here")
		assert.EqualError(t, err, "no such file or directory /not_here") // TODO
	})

	t.Run("path:/dir/not_here", func(t *testing.T) {
		_, err := fs.ReadDir("/dir/not_here")
		assert.EqualError(t, err, "no such file or directory /dir/not_here") // TODO
	})
}

func TestMkdir(t *testing.T) {
	fs, err := fat16.NewFromPath(testutils.DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	// TODO: verify should this be uppercase?
	testSingleMkdirWorked := func(fs *fat16.FileSystem) {
		t.Helper()
		entries, err := fs.ReadDir("/mkdir/single")
		assert.Nil(t, err)
		assert.Len(t, entries, 3)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "NEW"}, shortNames)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "NEW"}, longNames)

		assert.True(t, entries[2].IsDir())
		assert.Equal(t, "NEW", entries[2].ShortName())
	}

	t.Run("mkdir /mkdir/single/new", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/single/new")
		assert.Nil(t, err)

		testSingleMkdirWorked(fs)

		newFs, err := fat16.NewFromPath(testutils.DiskImagePath)
		assert.Nil(t, err)
		testSingleMkdirWorked(newFs)
	})

	t.Run("mkdir requiring new cluster", func(t *testing.T) {
		info := fs.Info()
		bytesPerCluster := info["bytesPerCluster"].(int64)

		// keep adding directories until we go over the cluster size
		i := int64(0)
		for ; i*32 < bytesPerCluster+32; i++ {
			err := fs.Mkdir(fmt.Sprintf("/mkdir/many/%d", i))
			assert.Nil(t, err)
		}

		entries, err := fs.ReadDir("/mkdir/many")
		assert.Nil(t, err)
		entryNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Len(t, entryNames, int(i+2)) // +2 for . and ..
	})

	t.Run("mkdir with long name", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/long/a_directory_with_a_long_name")
		assert.Nil(t, err)

		entries, err := fs.ReadDir("/mkdir/long")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "A_DIRE~1"}, shortNames)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "a_directory_with_a_long_name"}, longNames)
	})
}

func TestOpenFile(t *testing.T) {
	fs, err := fat16.NewFromPath(testutils.DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("read - sfn", func(t *testing.T) {
		file, err := fs.OpenFile("/INFO.TXT", os.O_RDONLY)
		assert.Nil(t, err)

		data := make([]byte, file.Size())
		n, err := file.ReadAt(data, 0)
		assert.Nil(t, err)
		assert.Equal(t, file.Size(), int64(n))
		assert.Equal(t, "text file\n", string(data))
	})

	t.Run("read - open dir", func(t *testing.T) {
		file, err := fs.OpenFile("/mkdir", os.O_RDONLY)
		assert.Nil(t, file)
		assert.EqualError(t, err, "failed to open '/mkdir': is a directory")
	})

	t.Run("read - no file", func(t *testing.T) {
		file, err := fs.OpenFile("/not_here", os.O_RDONLY)
		assert.Nil(t, file)
		assert.EqualError(t, err, "no such file or directory /not_here")
	})

	t.Run("read - lfn", func(t *testing.T) {
		file, err := fs.OpenFile("/a file with a long name.dat", os.O_RDONLY)
		assert.Nil(t, err)

		data := make([]byte, file.Size())
		n, err := file.ReadAt(data, 0)
		assert.Nil(t, err)
		assert.Equal(t, file.Size(), int64(n))
		assert.Equal(t, int64(7168), file.Size())
		assert.Equal(t, make([]byte, 7168), data)
	})

	t.Run("read - lfn by sfn", func(t *testing.T) {
		file, err := fs.OpenFile("/AFILEW~1.DAT", os.O_RDONLY)
		assert.Nil(t, err)

		data := make([]byte, file.Size())
		n, err := file.ReadAt(data, 0)
		assert.Nil(t, err)
		assert.Equal(t, file.Size(), int64(n))
		assert.Equal(t, int64(7168), file.Size())
		assert.Equal(t, make([]byte, 7168), data)
	})

	t.Run("read - less than file size", func(t *testing.T) {
		file, err := fs.OpenFile("/INFO.TXT", os.O_RDONLY)
		assert.Nil(t, err)

		data := make([]byte, 4)
		n, err := file.ReadAt(data, 0)
		assert.Nil(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, "text", string(data))
	})

	t.Run("read - more than file size", func(t *testing.T) {
		file, err := fs.OpenFile("/INFO.TXT", os.O_RDONLY)
		assert.Nil(t, err)

		data := make([]byte, 100)
		n, err := file.ReadAt(data, 0)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, "text file\n", string(data[:10]))
		assert.Equal(t, make([]byte, 90), data[10:])
	})

	t.Run("read - after close", func(t *testing.T) {
		file, err := fs.OpenFile("/INFO.TXT", os.O_RDONLY)
		assert.Nil(t, err)

		err = file.Close()
		assert.Nil(t, err)

		data := make([]byte, 100)
		_, err = file.ReadAt(data, 0)
		assert.EqualError(t, err, "file already closed")
	})

	t.Run("read - empty file", func(t *testing.T) {
		file, err := fs.OpenFile("/empty.bin", os.O_RDONLY)
		assert.Nil(t, err)

		data := make([]byte, 10)
		n, err := file.ReadAt(data, 0)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("read - at offset", func(t *testing.T) {
		file, err := fs.OpenFile("/INFO.TXT", os.O_RDONLY)
		assert.Nil(t, err)
		data := make([]byte, 5)
		n, err := file.ReadAt(data, 4)
		assert.Nil(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, " file", string(data))
	})

	t.Run("open - create if not exist", func(t *testing.T) {
		file, err := fs.OpenFile("/write/created.txt", os.O_CREATE)
		assert.Nil(t, err)

		data := make([]byte, 10)
		n, err := file.ReadAt(data, 0)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)

		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "CREATED.TXT"))
	})

	t.Run("write - not exist", func(t *testing.T) {
		_, err := fs.OpenFile("/write/not_here", os.O_RDONLY)
		assert.EqualError(t, err, "no such file or directory /write/not_here")
	})

	t.Run("write - create if not exist", func(t *testing.T) {
		file, err := fs.OpenFile("/write/new.txt", os.O_CREATE)
		assert.Nil(t, err)

		n, err := file.WriteAt([]byte{0, 1, 2, 3, 4}, 0)
		assert.Nil(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, int64(5), file.Size())

		data := make([]byte, 5)
		n, err = file.ReadAt(data, 0)
		assert.Nil(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte{0, 1, 2, 3, 4}, data)
	})

	t.Run("write - extend cluster chain", func(t *testing.T) {
		info := fs.Info()
		bytesPerCluster := info["bytesPerCluster"].(int64)

		file, err := fs.OpenFile("/write/extend_cluster.txt", os.O_CREATE)
		assert.Nil(t, err)

		toWrite := make([]byte, bytesPerCluster)
		for i := range bytesPerCluster {
			toWrite[i] = byte(i)
		}

		n, err := file.WriteAt(toWrite, 0)
		assert.Nil(t, err)
		assert.Equal(t, int(bytesPerCluster), n)
		assert.Equal(t, file.Size(), bytesPerCluster)

		n, err = file.WriteAt([]byte{42}, bytesPerCluster)
		assert.Nil(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, file.Size(), bytesPerCluster+1)
	})

	// TODO: more extensive write test suite (offsets, edge cases, etc)
}
