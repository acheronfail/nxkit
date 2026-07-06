package pfs0

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

const (
	hashTablePadding = 0x200
)

type BuildEntry struct {
	Name string
	Path string
	Data []byte
}

func BuildFromDir(inDir, outPath string) (uint64, error) {
	dirEntries, err := os.ReadDir(inDir)
	if err != nil {
		return 0, err
	}
	entries := make([]BuildEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		entries = append(entries, BuildEntry{Name: entry.Name(), Path: filepath.Join(inDir, entry.Name())})
	}
	slices.SortFunc(entries, func(a, b BuildEntry) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return Build(outPath, entries)
}

func Build(outPath string, entries []BuildEntry) (uint64, error) {
	out, err := os.Create(outPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	stringTableSize := uint32(0)
	sizes := make([]uint64, len(entries))
	for i, entry := range entries {
		size, err := entrySize(entry)
		if err != nil {
			return 0, err
		}
		sizes[i] = size
		stringTableSize += uint32(len(entry.Name) + 1)
	}
	stringTableSize = align32(stringTableSize, 0x20)

	header := make([]byte, headerSize)
	binary.LittleEndian.PutUint32(header[0x0:], Psf0Magic)
	binary.LittleEndian.PutUint32(header[0x4:], uint32(len(entries)))
	binary.LittleEndian.PutUint32(header[0x8:], stringTableSize)
	if _, err := out.Write(header); err != nil {
		return 0, err
	}

	dataOffset := uint64(0)
	stringOffset := uint32(0)
	entryTable := make([]byte, len(entries)*entryListSize)
	stringTable := make([]byte, stringTableSize)
	for i, entry := range entries {
		start := i * entryListSize
		binary.LittleEndian.PutUint64(entryTable[start:], dataOffset)
		binary.LittleEndian.PutUint64(entryTable[start+0x8:], sizes[i])
		binary.LittleEndian.PutUint32(entryTable[start+0x10:], stringOffset)
		copy(stringTable[stringOffset:], entry.Name)
		stringOffset += uint32(len(entry.Name) + 1)
		dataOffset += sizes[i]
	}
	if _, err := out.Write(entryTable); err != nil {
		return 0, err
	}
	if _, err := out.Write(stringTable); err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if err := writeEntryData(out, entry); err != nil {
			return 0, err
		}
	}

	pos, err := out.Seek(0, io.SeekCurrent)
	return uint64(pos), err
}

func CreateHashTable(pfs0Path, hashTablePath string, blockSize uint32) (hashTableSize uint64, pfs0Offset uint64, err error) {
	src, err := os.Open(pfs0Path)
	if err != nil {
		return 0, 0, err
	}
	defer src.Close()
	dst, err := os.Create(hashTablePath)
	if err != nil {
		return 0, 0, err
	}
	defer dst.Close()

	buf := make([]byte, blockSize)
	for {
		n, readErr := io.ReadFull(src, buf)
		if readErr == io.EOF {
			break
		}
		if readErr == io.ErrUnexpectedEOF {
			sum := sha256.Sum256(buf[:n])
			if _, err := dst.Write(sum[:]); err != nil {
				return 0, 0, err
			}
			hashTableSize += sha256.Size
			break
		}
		if readErr != nil {
			return 0, 0, readErr
		}
		sum := sha256.Sum256(buf[:n])
		if _, err := dst.Write(sum[:]); err != nil {
			return 0, 0, err
		}
		hashTableSize += sha256.Size
	}

	padding := uint64(hashTablePadding) - (hashTableSize % hashTablePadding)
	if padding != 0 {
		if _, err := dst.Write(make([]byte, padding)); err != nil {
			return 0, 0, err
		}
	}
	return hashTableSize, hashTableSize + padding, nil
}

func MasterHash(hashTablePath string, hashTableSize uint64) ([32]byte, error) {
	var zero [32]byte
	f, err := os.Open(hashTablePath)
	if err != nil {
		return zero, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.CopyN(h, f, int64(hashTableSize)); err != nil {
		return zero, err
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func entrySize(entry BuildEntry) (uint64, error) {
	if entry.Data != nil {
		return uint64(len(entry.Data)), nil
	}
	info, err := os.Stat(entry.Path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("pfs0 entry %s is a directory", entry.Path)
	}
	return uint64(info.Size()), nil
}

func writeEntryData(out io.Writer, entry BuildEntry) error {
	if entry.Data != nil {
		_, err := out.Write(entry.Data)
		return err
	}
	in, err := os.Open(entry.Path)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(out, in)
	return err
}

func align32(offset, alignment uint32) uint32 {
	return (offset + alignment - 1) & ^(alignment - 1)
}
