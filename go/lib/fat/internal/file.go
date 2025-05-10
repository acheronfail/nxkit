package internal

import (
	"io"
	"os"
)

type fatFile struct {
	Entry
	parentDirCluster *uint16
	// this value is the same as `os.OpenFile`'s flags ANDed with `0b11`
	accessMode int
	fs         *FileSystem
}

func (f *fatFile) Close() error {
	f.fs = nil
	return nil
}

func (f *fatFile) Size() int64 {
	return int64(f.Entry.DIR_FileSize)
}

func (f *fatFile) ReadAt(p []byte, off int64) (n int, err error) {
	if f.fs == nil {
		return 0, os.ErrClosed
	}

	canRead := f.accessMode == os.O_RDONLY || f.accessMode == os.O_RDWR
	if !canRead {
		return 0, os.ErrPermission
	}

	fileSize := int64(f.DIR_FileSize)

	// if file size is zero, then this file has no cluster chain so read nothing
	if fileSize == 0 {
		if len(p) > 0 {
			return 0, io.EOF
		}

		return 0, nil
	}

	// if size non-zero, then find cluster chain and read from it
	bytes, err := f.fs.getClusterChainBytes(f.clusterNumber())
	if err != nil {
		return 0, err
	}

	// though we have allocated more space than the current file size, we don't
	// allow reading beyond the file contents itself
	if off >= int64(fileSize) {
		return 0, io.EOF
	}

	end := min(off+int64(len(p)), int64(fileSize))
	n = copy(p, bytes[off:end])
	if len(p) > n {
		return n, io.EOF
	}

	return n, nil
}

func (f *fatFile) WriteAt(p []byte, off int64) (n int, err error) {
	if f.fs == nil {
		return 0, os.ErrClosed
	}

	canWrite := f.accessMode == os.O_WRONLY || f.accessMode == os.O_RDWR
	if !canWrite || f.IsReadOnly() {
		return 0, os.ErrPermission
	}

	writeLen := int64(len(p))
	totalLen := writeLen + off

	if writeLen == 0 {
		return 0, nil
	}

	// allocate cluster if file is empty
	if f.clusterNumber() == 0 {
		cluster, err := f.fs.allocateClusterChain(totalLen)
		if err != nil {
			return 0, err
		}

		f.setCluster(cluster)
	}

	cluster := f.clusterNumber()
	clusterBytes, err := f.fs.getClusterChainBytes(cluster)
	if err != nil {
		return 0, err
	}

	// extend cluster chain if current doesn't have enough space
	if len(clusterBytes) < int(totalLen) {
		err = f.fs.extendClusterChain(cluster, totalLen)
		if err != nil {
			return 0, err
		}

		clusterBytes, err = f.fs.getClusterChainBytes(cluster)
		if err != nil {
			return 0, err
		}
	}

	// write data into cluster's bytes
	n = copy(clusterBytes[off:off+writeLen], p)

	// write cluster back to disk
	err = f.fs.writeClusterChain(cluster, clusterBytes)
	if err != nil {
		return 0, err
	}

	// update file size
	f.DIR_FileSize = uint32(totalLen)

	// write file entry back to parent
	err = f.writeEntryToParent()
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (f *fatFile) writeEntryToParent() error {
	index, parentDirBytes, err := f.fs.findIndexInParentBytes(&f.Entry, f.parentDirCluster)
	if err != nil {
		return err
	}

	// write back to disk
	err = f.fs.writeEntriesToParent(
		[]to32Bytes{&f.fatDirectoryEntry},
		f.parentDirCluster,
		parentDirBytes,
		index,
	)
	if err != nil {
		return err
	}

	return nil
}
