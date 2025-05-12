package fat

import "github.com/acheronfail/nxkit/lib/fat/boot_sector"

type FileSystem interface {
	Close() error
	GetVolumeId() (string, error)
	GetType() boot_sector.FatType
	Info() map[string]any
	Mkdir(path string) error
	OpenFile(path string, flags int) (File, error)
	ReadDir(path string) ([]DirectoryEntry, error)
	Rename(srcPath, dstPath string) error
	Rmdir(path string) error
	Stat(path string) (Stat, error)
	Unlink(path string) error
}

type DirectoryEntry interface {
	ShortName() string
	LongName() string
	Stat
}

type File interface {
	Size() int64
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Close() error
}

type Stat interface {
	Size() int64
	IsReadOnly() bool
	IsHidden() bool
	IsSystem() bool
	IsDir() bool
	IsFile() bool
	IsArchive() bool
}
