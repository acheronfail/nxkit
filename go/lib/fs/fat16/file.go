package fat16

import (
	"io"
	"os"
)

type fatFile struct {
	DirectoryEntry
	parent *DirectoryEntry
	fs     *FileSystem
}

func (f *fatFile) Close() error {
	f.fs = nil
	return nil
}

func (f *fatFile) Size() int64 {
	return int64(f.DirectoryEntry.DIR_FileSize)
}

func (f *fatFile) ReadAt(p []byte, off int64) (n int, err error) {
	if f.fs == nil {
		return 0, os.ErrClosed
	}

	fileSize := int64(f.DirectoryEntry.DIR_FileSize)

	// if file size is zero, then this file has no cluster chain so read nothing
	if fileSize == 0 {
		if len(p) > 0 {
			return 0, io.EOF
		}

		return 0, nil
	}

	// if size non-zero, then find cluster chain and read from it
	bytes, err := f.fs.getClusterChainBytes(f.DirectoryEntry.clusterNumber())
	if err != nil {
		return 0, err
	}

	copy(p, bytes[off:fileSize])
	n = int(fileSize - off)
	if len(p) > n {
		return n, io.EOF
	}

	return n, nil
}

// WriteAt implements fs.File.
func (f *fatFile) WriteAt(p []byte, off int64) (n int, err error) {
	panic("unimplemented")
}
