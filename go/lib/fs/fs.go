package fs

type File interface {
	Size() int64
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Close() error
}
