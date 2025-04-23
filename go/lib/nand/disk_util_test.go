package nand

import (
	"bytes"
	"io/fs"
	"os"
	"testing"

	"github.com/diskfs/go-diskfs/backend"
)

var (
	CLUSTER_SIZE   = int64(16)
	DISK_SIZE      = int64(16) * 8
	FS_SECTOR_SIZE = uint64(64)
)

/**
 * Test helpers
 */

func getLayer(size int64) (*NandBackend, *MemoryBackend) {
	backend := NewMemoryBackend(size)
	return NewNandBackend(backend, 0, uint64(size), 16, FS_SECTOR_SIZE), backend
}

func readAt(backend backend.Storage, offset int64, size int) []byte {
	data := make([]byte, size)
	_, err := backend.ReadAt(data, offset)
	if err != nil {
		panic(err)
	}

	return data
}

func writeAt(backend backend.WritableFile, offset int64, data []byte) int {
	n, err := backend.WriteAt(data, offset)
	if err != nil {
		panic(err)
	}

	return n
}

func assertBytes(t *testing.T, actual, expected []byte) {
	if !bytes.Equal(expected, actual) {
		t.Errorf("Bytes mismatch:\nexpected %v\n     got %v", expected, actual)
	}
}

func assertLength(t *testing.T, actual, expected int) {
	if expected != actual {
		t.Errorf("Length mismatch:\nexpected %v\n     got %v", expected, actual)
	}
}

/**
 * Simple XOR encryption for testing
 */

type XorCrypto struct{}

func (*XorCrypto) xorWithClusterOffset(input []byte, byteOffset uint64) []byte {
	data := make([]byte, len(input))
	copy(data, input)

	clusterOffset := int64(byteOffset) / CLUSTER_SIZE
	for i := 0; i < len(data); i += int(CLUSTER_SIZE) {
		for j := 0; j < int(CLUSTER_SIZE) && i+j < len(data); j++ {
			data[i+j] ^= byte(clusterOffset)
		}
		clusterOffset++
	}

	return data
}

func (x *XorCrypto) Decrypt(input []byte, byteOffset uint64) ([]byte, error) {
	data := x.xorWithClusterOffset(input, byteOffset)
	return data, nil
}

func (x *XorCrypto) Encrypt(input []byte, byteOffset uint64) ([]byte, error) {
	data := x.xorWithClusterOffset(input, byteOffset)
	return data, nil
}

func NewXorCrypto() *XorCrypto {
	return &XorCrypto{}
}

/**
 *  Simple type to easily create byte slices from run length encoded data
 */

type Rle struct {
	what  byte
	count int
}

func expand(rle []Rle) []byte {
	out := make([]byte, 0, len(rle))
	for _, r := range rle {
		for range r.count {
			out = append(out, r.what)
		}
	}

	return out
}

/**
 * Simple in memory backend for testing
 */

type MemoryBackend struct {
	data    []byte
	size    int64
	pointer int64
}

func NewMemoryBackend(size int64) *MemoryBackend {
	return &MemoryBackend{
		data:    make([]byte, size),
		size:    size,
		pointer: 0,
	}
}

func (m *MemoryBackend) zeroWithXor() {
	for i := 0; i*int(CLUSTER_SIZE) < int(DISK_SIZE); i++ {
		writeAt(m, int64(i)*CLUSTER_SIZE, bytes.Repeat([]byte{byte(0 ^ i)}, int(CLUSTER_SIZE)))
	}
}

// Close implements backend.Storage.
func (m *MemoryBackend) Close() error {
	return nil
}

// Read implements backend.Storage.
func (m *MemoryBackend) Read(p []byte) (int, error) {
	return m.ReadAt(p, m.pointer)
}

// ReadAt implements backend.Storage.
func (m *MemoryBackend) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= m.size {
		return 0, nil
	}

	if off+int64(len(p)) > m.size {
		p = p[:m.size-off]
	}

	n = copy(p, m.data[off:])
	m.pointer += int64(n)
	return n, nil
}

// WriteAt implements backend.WritableFile.
func (m *MemoryBackend) WriteAt(p []byte, off int64) (n int, err error) {
	if off >= m.size {
		return 0, nil
	}

	if off+int64(len(p)) > m.size {
		p = p[:m.size-off]
	}

	n = copy(m.data[off:], p)
	m.pointer += int64(n)
	return n, nil
}

// Seek implements backend.Storage.
func (m *MemoryBackend) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		m.pointer = offset
	case 1:
		m.pointer += offset
	case 2:
		m.pointer = m.size - offset
	default:
		return 0, os.ErrInvalid
	}

	if m.pointer < 0 || m.pointer > m.size {
		return 0, os.ErrInvalid
	}

	return m.pointer, nil
}

// Stat implements backend.Storage.
func (m *MemoryBackend) Stat() (fs.FileInfo, error) {
	return &NandStats{
		isEncrypted: false,
		size:        m.size,
	}, nil
}

// Sys implements backend.Storage.
func (m *MemoryBackend) Sys() (*os.File, error) {
	return nil, nil
}

// Writable implements backend.Storage.
func (m *MemoryBackend) Writable() (backend.WritableFile, error) {
	return m, nil
}
