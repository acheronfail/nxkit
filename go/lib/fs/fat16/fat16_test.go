package fat16_test

import (
	"fmt"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/acheronfail/nxkit/lib/fs/fat16"
	"github.com/acheronfail/nxkit/lib/fs/testdata"
	"github.com/acheronfail/nxkit/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestReadDir(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
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
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	testSingleMkdirWorked := func(fs *fat16.FileSystem) {
		t.Helper()
		entries, err := fs.ReadDir("/mkdir/single")
		assert.Nil(t, err)
		assert.Len(t, entries, 3)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "new"}, shortNames)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "new"}, longNames)

		assert.True(t, entries[2].IsDir())
		assert.Equal(t, "new", entries[2].ShortName())
	}

	t.Run("mkdir /mkdir/single/new", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/single/new")
		assert.Nil(t, err)

		testSingleMkdirWorked(fs)

		newFs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
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

	t.Run("mkdir with 1 lfn", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/long1/aBcDe")
		assert.Nil(t, err)

		entries, err := fs.ReadDir("/mkdir/long1")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "ABCDE"}, shortNames)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "aBcDe"}, longNames)
	})

	t.Run("mkdir with multiple lfns", func(t *testing.T) {
		err := fs.Mkdir("/mkdir/long2/a_directory_with_a_long_name")
		assert.Nil(t, err)

		entries, err := fs.ReadDir("/mkdir/long2")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.Equal(t, []string{".", "..", "A_DIRE~1"}, shortNames)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.Equal(t, []string{".", "..", "a_directory_with_a_long_name"}, longNames)
	})
}

func TestOpenFile(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
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
		file, err := fs.OpenFile("/write/created.txt", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		data := make([]byte, 10)
		_, err = file.ReadAt(data, 0)
		assert.EqualError(t, err, "permission denied")

		file, err = fs.OpenFile("/write/created.txt", os.O_RDWR|os.O_CREATE)
		assert.Nil(t, err)
		n, err := file.ReadAt(data, 0)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)

		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "created.txt"))
	})

	t.Run("write - not exist", func(t *testing.T) {
		_, err := fs.OpenFile("/write/not_here", os.O_RDONLY)
		assert.EqualError(t, err, "no such file or directory /write/not_here")
	})

	t.Run("write - create if not exist", func(t *testing.T) {
		file, err := fs.OpenFile("/write/new.txt", os.O_RDWR|os.O_CREATE)
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

		file, err := fs.OpenFile("/write/extend_cluster.txt", os.O_RDWR|os.O_CREATE)
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

	t.Run("write - truncate", func(t *testing.T) {
		// create the file and write some data
		file, err := fs.OpenFile("/write/trunc.txt", os.O_RDWR|os.O_CREATE)
		assert.Nil(t, err)
		n, err := file.WriteAt([]byte{0, 1, 2, 3, 4}, 0)
		assert.Nil(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, int64(5), file.Size())
		err = file.Close()
		assert.Nil(t, err)

		// now re-open the file with truncate
		file, err = fs.OpenFile("/write/trunc.txt", os.O_RDWR|os.O_TRUNC)
		assert.Nil(t, err)
		assert.Equal(t, int64(0), file.Size())

		// double check reading the file returns EOF
		data := make([]byte, 5)
		n, err = file.ReadAt(data, 0)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 0, n)
	})

	// TODO: more extensive write test suite (offsets, edge cases, etc)
}

func TestStat(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("stat - file", func(t *testing.T) {
		stat, err := fs.Stat("/INFO.TXT")
		assert.Nil(t, err)
		assert.Equal(t, int64(10), stat.Size())
		assert.True(t, !stat.IsReadOnly())
		assert.True(t, !stat.IsHidden())
		assert.True(t, !stat.IsSystem())
		assert.True(t, !stat.IsVolumeId())
		assert.True(t, !stat.IsDir())
		assert.True(t, stat.IsFile())
	})

	t.Run("stat - dir", func(t *testing.T) {
		stat, err := fs.Stat("/dir")
		assert.Nil(t, err)
		assert.Equal(t, int64(0), stat.Size())
		assert.True(t, !stat.IsReadOnly())
		assert.True(t, !stat.IsHidden())
		assert.True(t, !stat.IsSystem())
		assert.True(t, !stat.IsVolumeId())
		assert.True(t, stat.IsDir())
		assert.True(t, !stat.IsFile())
	})

	t.Run("stat - volume id", func(t *testing.T) {
		stat, err := fs.Stat("/FAT16-TEST")
		assert.Nil(t, err)
		assert.Equal(t, int64(0), stat.Size())
		assert.True(t, !stat.IsReadOnly())
		assert.True(t, !stat.IsHidden())
		assert.True(t, !stat.IsSystem())
		assert.True(t, stat.IsVolumeId())
	})
}

func TestUnlink(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("unlink - empty file", func(t *testing.T) {
		// create empty file
		_, err := fs.OpenFile("/write/unlink", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "unlink"))

		// unlink
		err = fs.Unlink("/write/unlink")
		assert.Nil(t, err)

		// check it doesn't exist in parent dir
		entries, err = fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.False(t, slices.Contains(shortNames, "unlink"))
	})

	t.Run("unlink - non-empty file", func(t *testing.T) {
		// create non-empty file
		file, err := fs.OpenFile("/write/unlink", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)
		n, err := file.WriteAt([]byte{0, 1, 2, 3, 4}, 0)
		assert.Nil(t, err)
		assert.Equal(t, 5, n)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "unlink"))

		// unlink
		err = fs.Unlink("/write/unlink")
		assert.Nil(t, err)

		// check it doesn't exist in parent dir
		entries, err = fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.False(t, slices.Contains(shortNames, "unlink"))

		// TODO: internal test which test lfn entries properly deleted
		// TODO: internal test which checks the cluster chain is deleted
	})

	t.Run("unlink - lfn", func(t *testing.T) {
		// create empty file
		_, err := fs.OpenFile("/write/a_file_with_a_long_name", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.True(t, slices.Contains(longNames, "a_file_with_a_long_name"))

		// unlink
		err = fs.Unlink("/write/a_file_with_a_long_name")
		assert.Nil(t, err)

		// check it doesn't exist in parent dir
		entries, err = fs.ReadDir("/write")
		assert.Nil(t, err)
		longNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.False(t, slices.Contains(longNames, "a_file_with_a_long_name"))
	})

	t.Run("unlink - not found", func(t *testing.T) {
		err := fs.Unlink("/not_here")
		assert.EqualError(t, err, "no such file or directory /not_here")
	})

	t.Run("unlink - dir", func(t *testing.T) {
		err := fs.Unlink("/write")
		assert.EqualError(t, err, "cannot unlink directory /write")
	})
}

func TestRmdir(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("rmdir - empty dir", func(t *testing.T) {
		// create empty dir
		err := fs.Mkdir("/write/rmdir/empty")
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write/rmdir")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "empty"))

		// rmdir
		err = fs.Rmdir("/write/rmdir/empty")
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err = fs.ReadDir("/write/rmdir")
		assert.Nil(t, err)
		shortNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.False(t, slices.Contains(shortNames, "empty"))
	})

	t.Run("rmdir - lfn", func(t *testing.T) {
		// create dir
		err := fs.Mkdir("/write/rmdir/a_dir_with_a_long_name")
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write/rmdir")
		assert.Nil(t, err)
		longNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.True(t, slices.Contains(longNames, "a_dir_with_a_long_name"))

		// rmdir
		err = fs.Rmdir("/write/rmdir/a_dir_with_a_long_name")
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err = fs.ReadDir("/write/rmdir")
		assert.Nil(t, err)
		longNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.LongName() })
		assert.False(t, slices.Contains(longNames, "a_dir_with_a_long_name"))
	})

	t.Run("rmdir - non-empty dir", func(t *testing.T) {
		err := fs.Rmdir("/dir")
		assert.EqualError(t, err, "directory /dir is not empty")
	})
}

func TestRename(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()

	t.Run("rename - file - same parent", func(t *testing.T) {
		// create empty file
		_, err := fs.OpenFile("/write/rename1.fil", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		// rename
		err = fs.Rename("/write/rename1.fil", "/write/rename2.fil")
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "rename2.fil"))

		// rename back
		err = fs.Rename("/write/rename2.fil", "/write/rename1.fil")
		assert.Nil(t, err)
	})

	t.Run("rename - dir - same parent", func(t *testing.T) {
		// create empty dir
		err := fs.Mkdir("/write/rename1.dir")
		assert.Nil(t, err)

		// rename
		err = fs.Rename("/write/rename1.dir", "/write/rename2.dir")
		assert.Nil(t, err)

		// check it exists in parent dir
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "rename2.dir"))

		// rename back
		err = fs.Rename("/write/rename2.dir", "/write/rename1.dir")
		assert.Nil(t, err)
	})

	t.Run("rename - file - different parent", func(t *testing.T) {
		err := fs.Mkdir("/write/rename")
		assert.Nil(t, err)

		// create empty file
		_, err = fs.OpenFile("/write/r1", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		// rename
		err = fs.Rename("/write/r1", "/write/rename/r1")
		assert.Nil(t, err)

		// check not exists in old parent
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.False(t, slices.Contains(shortNames, "r1"))
		// check exists in new parent
		entries, err = fs.ReadDir("/write/rename")
		assert.Nil(t, err)
		shortNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "r1"))

		// rename back
		err = fs.Rename("/write/rename/r1", "/write/r1")
		assert.Nil(t, err)
	})

	t.Run("rename - dir - different parent", func(t *testing.T) {
		err := fs.Mkdir("/write/rename")
		assert.Nil(t, err)

		// create empty dir
		err = fs.Mkdir("/write/r2")
		assert.Nil(t, err)

		// rename
		err = fs.Rename("/write/r2", "/write/rename/r2")
		assert.Nil(t, err)

		// check not exists in old parent dir
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.False(t, slices.Contains(shortNames, "r2"))
		// check exists in new parent dir
		entries, err = fs.ReadDir("/write/rename")
		assert.Nil(t, err)
		shortNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "r2"))

		// rename back
		err = fs.Rename("/write/rename/r2", "/write/r2")
		assert.Nil(t, err)
	})

	t.Run("rename - to existing file", func(t *testing.T) {
		// create empty files
		_, err = fs.OpenFile("/write/r3.1", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)
		_, err = fs.OpenFile("/write/r3.2", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		// check both exist
		entries, err := fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames := utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.True(t, slices.Contains(shortNames, "r3.1"))
		assert.True(t, slices.Contains(shortNames, "r3.2"))

		// rename
		err = fs.Rename("/write/r3.1", "/write/r3.2")
		assert.Nil(t, err)

		// check only one exists
		entries, err = fs.ReadDir("/write")
		assert.Nil(t, err)
		shortNames = utils.MapSlice(entries, func(entry fat16.DirectoryEntry) string { return entry.ShortName() })
		assert.False(t, slices.Contains(shortNames, "r3.1"))
		assert.True(t, slices.Contains(shortNames, "r3.2"))

		// TODO: internal test to check that the cluster chain is deleted from target file
	})

	t.Run("rename - to existing dir", func(t *testing.T) {
		// create empty file
		_, err = fs.OpenFile("/write/r4", os.O_WRONLY|os.O_CREATE)
		assert.Nil(t, err)

		// rename
		err = fs.Rename("/write/r4", "/dir")
		assert.EqualError(t, err, "cannot rename to directory /dir")
	})
}
