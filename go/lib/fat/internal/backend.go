package internal

import (
	"io/fs"

	"github.com/acheronfail/nxkit/lib/fat/backend"
)

type readonlyBackend struct {
	inner          backend.Storage
	writePermError error
}

// Close implements backend.WritableFile.
func (f readonlyBackend) Close() error {
	return f.inner.Close()
}

// Read implements backend.WritableFile.
func (f readonlyBackend) Read(p []byte) (int, error) {
	return f.inner.Read(p)
}

// ReadAt implements backend.WritableFile.
func (f readonlyBackend) ReadAt(p []byte, off int64) (n int, err error) {
	return f.inner.ReadAt(p, off)
}

// Seek implements backend.WritableFile.
func (f readonlyBackend) Seek(offset int64, whence int) (int64, error) {
	return f.inner.Seek(offset, whence)
}

// Stat implements backend.WritableFile.
func (f readonlyBackend) Stat() (fs.FileInfo, error) {
	return f.inner.Stat()
}

// WriteAt implements backend.WritableFile.
func (f readonlyBackend) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, f.writePermError
}
