package nand

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/acheronfail/nxkit/lib/xtsn"
	"github.com/diskfs/go-diskfs/backend"
)

// TODO: use int64/uint64 where appropriate
type NandBackend struct {
	seekOffset      int64
	rawnand         backend.Storage
	partitionStart  uint64
	partitionEnd    uint64
	crypto          xtsn.Crypto
	cryptoBlockSize uint64
	fsSectorSize    uint64
	fsSectorCount   uint64
}

// Close implements backend.Storage.
func (l *NandBackend) Close() error {
	return l.rawnand.Close()
}

// Read implements backend.Storage.
func (l *NandBackend) Read(p []byte) (int, error) {
	n, err := l.ReadAt(p, l.seekOffset)
	if err != nil {
		return 0, err
	}

	l.seekOffset += int64(n)
	return n, nil
}

// ReadAt implements backend.Storage.
func (l *NandBackend) ReadAt(p []byte, diskOffset int64) (n int, err error) {
	// can't read past the end
	if diskOffset > int64(l.partitionEnd) {
		return 0, nil
	}

	size := int64(len(p))
	if diskOffset+size > int64(l.partitionEnd) {
		size = int64(l.partitionEnd) - diskOffset
	}

	if l.crypto == nil {
		return l.rawnand.ReadAt(p, int64(diskOffset))
	}

	// In order to decrypt, we need to start at a 16 byte offset (it uses aes-based
	// encryption which works in 128 bit chunks), so if we wanted to read `xxxx` in:
	//  11112222333344xx xx55666677778888
	//  BBBBBBBBBBBBBB     AAAAAAAAAAAAAA   B = before, A = after, ^ = offset
	//                ^
	// we need to get the offset of the start of the first 16 byte chunk, and the end of
	// the second chunk. We then read both chunks and decrypt them, and return the
	// desired bytes (discarding `before` and `after`)

	partOffset := diskOffset - int64(l.partitionStart)
	before := partOffset % int64(l.cryptoBlockSize)
	after := (partOffset + size) % int64(l.cryptoBlockSize)
	if after != 0 {
		after = int64(l.cryptoBlockSize) - after
	}

	readSize := before + size + after
	alignedDiskOffset := diskOffset - before
	partByteOffset := partOffset - before

	buf := make([]byte, readSize)
	n, err = l.rawnand.ReadAt(buf, alignedDiskOffset)
	if err != nil {
		return 0, err
	}

	if n != int(readSize) {
		return 0, fmt.Errorf("read %d bytes, expected %d", n, readSize)
	}

	decrypted, err := l.crypto.Decrypt(buf, uint64(partByteOffset))
	if err != nil {
		return 0, err
	}

	copy(p, decrypted[before:before+size])
	return int(size), nil
}

// WriteAt implements backend.WritableFile.
func (l *NandBackend) WriteAt(p []byte, diskOffset int64) (n int, err error) {
	// can't write past the end
	if diskOffset > int64(l.partitionEnd) {
		return 0, nil
	}

	size := int64(len(p))
	if diskOffset+size > int64(l.partitionEnd) {
		size = int64(l.partitionEnd) - diskOffset
	}

	if l.crypto == nil {
		writable, err := l.rawnand.Writable()
		if err != nil {
			return 0, err
		}

		return writable.WriteAt(p, int64(diskOffset))
	}

	partOffset := diskOffset - int64(l.partitionStart)
	before := partOffset % int64(l.cryptoBlockSize)
	after := (partOffset + size) % int64(l.cryptoBlockSize)
	if after != 0 {
		after = int64(l.cryptoBlockSize) - after
	}

	alignedDiskOffset := diskOffset - before
	partByteOffset := partOffset - before
	writeSize := before + size + after

	// Prepare buffer for the full encrypted block
	chunks := make([]byte, writeSize)

	// If we have leading bytes, read them first
	if before > 0 {
		n, err = l.ReadAt(chunks[:before], alignedDiskOffset)
		if err != nil {
			return 0, err
		}
		if n != int(before) {
			return 0, fmt.Errorf("read %d bytes for leading chunk, expected %d", n, before)
		}
	}

	// Copy the data we want to write
	copy(chunks[before:before+size], p[:size])

	// If we have trailing bytes, read them too
	if after > 0 {
		n, err = l.ReadAt(chunks[before+size:], alignedDiskOffset+before+size)
		if err != nil {
			return 0, err
		}
		if n != int(after) {
			return 0, fmt.Errorf("read %d bytes for trailing chunk, expected %d", n, after)
		}
	}

	// Encrypt the entire buffer
	enc, err := l.crypto.Encrypt(chunks, uint64(partByteOffset))
	if err != nil {
		return 0, err
	}

	// Get the writable backend and write the encrypted data
	writable, err := l.rawnand.Writable()
	if err != nil {
		return 0, err
	}

	n, err = writable.WriteAt(enc, alignedDiskOffset)
	if err != nil {
		return 0, err
	}

	if n != int(writeSize) {
		return 0, fmt.Errorf("wrote %d bytes, expected %d", n, writeSize)
	}

	return int(writeSize), nil
}

// Seek implements backend.Storage.
func (l *NandBackend) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		l.seekOffset = offset
	case 1:
		l.seekOffset += offset
	case 2:
		l.seekOffset = int64(l.partitionEnd) - offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if l.seekOffset < int64(l.partitionStart) || l.seekOffset > int64(l.partitionEnd) {
		return 0, fmt.Errorf("seek out of bounds: %d", l.seekOffset)
	}

	return l.seekOffset, nil
}

type NandStats struct {
	name string
	size int64
}

func (n *NandStats) IsDir() bool {
	return false
}

// ModTime implements fs.FileInfo.
func (n *NandStats) ModTime() time.Time {
	return time.Now()
}

// Mode implements fs.FileInfo.
func (n *NandStats) Mode() fs.FileMode {
	return 0
}

// Name implements fs.FileInfo.
func (n *NandStats) Name() string {
	return n.name
}

// Size implements fs.FileInfo.
func (n *NandStats) Size() int64 {
	return n.size
}

// Sys implements fs.FileInfo.
func (n *NandStats) Sys() any {
	return nil
}

// Stat implements backend.Storage.
func (l *NandBackend) Stat() (fs.FileInfo, error) {
	var name string
	if l.crypto == nil {
		name = "NAND(unencrypted)"
	} else {
		name = "NAND(encrypted)"
	}

	return &NandStats{
		name: name,
		size: int64(l.fsSectorCount * l.fsSectorSize),
	}, nil
}

// Sys implements backend.Storage.
func (l *NandBackend) Sys() (*os.File, error) {
	return l.rawnand.Sys()
}

// Writable implements backend.Storage.
func (l *NandBackend) Writable() (backend.WritableFile, error) {
	return l, nil
}

func NewNandBackend(rawnand backend.Storage, partStart, partEnd, cryptoBlockSize, fsSectorSize uint64) *NandBackend {
	return &NandBackend{
		rawnand:         rawnand,
		partitionStart:  partStart,
		partitionEnd:    partEnd,
		crypto:          nil,
		cryptoBlockSize: cryptoBlockSize,
		fsSectorSize:    fsSectorSize,
		fsSectorCount:   (partEnd - partStart) / fsSectorSize,
	}
}

func (l *NandBackend) SetCrypto(crypto xtsn.Crypto) {
	l.crypto = crypto
}
