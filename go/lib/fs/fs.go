package fs

// TODO: FileSystem interface to share between FAT16 and FAT32

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
	IsVolumeId() bool
	IsDir() bool
	IsFile() bool
	IsArchive() bool
}
