package romfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type RomFSHeader struct {
	HeaderSize        uint64
	DirHashTableOfs   uint64
	DirHashTableSize  uint64
	DirTableOfs       uint64
	DirTableSize      uint64
	FileHashTableOfs  uint64
	FileHashTableSize uint64
	FileTableOfs      uint64
	FileTableSize     uint64
	FilePartitionOfs  uint64
}

type DirEntry struct {
	Parent   uint32
	Sibling  uint32
	Child    uint32
	File     uint32
	Hash     uint32
	NameSize uint32
	Name     []byte
}

type FileEntry struct {
	Parent   uint32
	Sibling  uint32
	Offset   uint64
	Size     uint64
	Hash     uint32
	NameSize uint32
	Name     []byte
}

type DirContext struct {
	Path        string
	EntryOffset uint32
	Parent      *DirContext
	Child       *DirContext
	Sibling     *DirContext
	File        *FileContext
	Next        *DirContext
}

type FileContext struct {
	Path        string
	EntryOffset uint32
	Offset      uint64
	Size        uint64
	Parent      *DirContext
	Sibling     *FileContext
	Next        *FileContext
}

type RomFSContext struct {
	Files             *FileContext
	NumDirs           uint64
	NumFiles          uint64
	DirTableSize      uint64
	FileTableSize     uint64
	DirHashTableSize  uint64
	FileHashTableSize uint64
	FilePartitionSize uint64
}

const (
	RomFSEntryEmpty       = 0xFFFFFFFF
	RomFSFilePartitionOfs = 0x200
	IVFCHashBlockSize     = 0x4000
)

func BuildRomFS(inputPath string, outputPath string) (uint64, error) {
	// Create root context
	rootCtx := &DirContext{
		Path: inputPath,
	}
	rootCtx.Parent = rootCtx

	// Initialize RomFS context
	ctx := &RomFSContext{
		DirTableSize: 0x18, // Root directory
		NumDirs:      1,
	}

	// Visit all directories recursively
	if err := visitDir(rootCtx, ctx); err != nil {
		return 0, err
	}

	// Calculate hash table sizes
	dirHashTableCount := getHashTableCount(ctx.NumDirs)
	fileHashTableCount := getHashTableCount(ctx.NumFiles)
	ctx.DirHashTableSize = 4 * dirHashTableCount
	ctx.FileHashTableSize = 4 * fileHashTableCount

	// Create hash tables
	dirHashTable := make([]uint32, dirHashTableCount)
	fileHashTable := make([]uint32, fileHashTableCount)
	for i := range dirHashTable {
		dirHashTable[i] = RomFSEntryEmpty
	}
	for i := range fileHashTable {
		fileHashTable[i] = RomFSEntryEmpty
	}

	// Calculate file offsets
	for file := ctx.Files; file != nil; file = file.Next {
		ctx.FilePartitionSize = align64(ctx.FilePartitionSize, 0x10)
		file.Offset = ctx.FilePartitionSize
		ctx.FilePartitionSize += file.Size
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Write header placeholder
	header := RomFSHeader{
		HeaderSize:       uint64(binary.Size(RomFSHeader{})),
		FilePartitionOfs: RomFSFilePartitionOfs,
	}

	if err := binary.Write(f, binary.LittleEndian, header); err != nil {
		return 0, err
	}

	// Write file data
	for file := ctx.Files; file != nil; file = file.Next {
		if err := writeFileData(f, file, RomFSFilePartitionOfs); err != nil {
			return 0, err
		}
	}

	// Calculate and write tables
	dirHashTableOffset := align64(ctx.FilePartitionSize+RomFSFilePartitionOfs, 4)

	// Update header with final values
	header.DirHashTableOfs = dirHashTableOffset
	header.DirHashTableSize = ctx.DirHashTableSize
	header.DirTableOfs = header.DirHashTableOfs + ctx.DirHashTableSize
	header.FileHashTableOfs = header.DirTableOfs + ctx.DirTableSize
	header.FileTableOfs = header.FileHashTableOfs + ctx.FileHashTableSize

	// Write tables
	if err := writeTables(f, ctx, dirHashTable, fileHashTable); err != nil {
		return 0, err
	}

	// Write final header
	if _, err := f.Seek(0, 0); err != nil {
		return 0, err
	}
	if err := binary.Write(f, binary.LittleEndian, header); err != nil {
		return 0, err
	}

	// Add padding
	size, err := f.Seek(0, 2) // Get file size
	if err != nil {
		return 0, err
	}

	paddingSize := IVFCHashBlockSize - (uint64(size) % IVFCHashBlockSize)
	if paddingSize > 0 {
		padding := make([]byte, paddingSize)
		if _, err := f.Write(padding); err != nil {
			return 0, err
		}
	}

	return uint64(size), nil
}

// writeFileData writes the actual file contents to the RomFS
func writeFileData(f *os.File, fileCtx *FileContext, baseOffset uint64) error {
	// Open source file
	src, err := os.Open(fileCtx.Path)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", fileCtx.Path, err)
	}
	defer src.Close()

	// Seek to correct position in output file
	offset := baseOffset + fileCtx.Offset
	if _, err := f.Seek(int64(offset), 0); err != nil {
		return fmt.Errorf("failed to seek output file: %w", err)
	}

	// Use a 100MB buffer for copying
	bufSize := uint64(100 * 1024 * 1024) // 100 MB
	buf := make([]byte, bufSize)

	// Copy file data
	remaining := fileCtx.Size
	for remaining > 0 {
		if remaining < bufSize {
			bufSize = remaining
		}

		n, err := src.Read(buf[:bufSize])
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read source file: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := f.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}

		remaining -= uint64(n)
	}

	return nil
}

// writeTables writes all the RomFS tables to the output file
func writeTables(f *os.File, ctx *RomFSContext, dirHashTable, fileHashTable []uint32) error {
	// Prepare directory table
	dirTable := make([]byte, ctx.DirTableSize)
	fileTable := make([]byte, ctx.FileTableSize)

	// Helper to write a directory entry
	writeDirEntry := func(entry *DirEntry, offset uint32) error {
		buf := bytes.NewBuffer(dirTable[offset:offset])
		if err := binary.Write(buf, binary.LittleEndian, entry.Parent); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Sibling); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Child); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.File); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Hash); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.NameSize); err != nil {
			return err
		}
		copy(dirTable[offset+0x18:], entry.Name)
		return nil
	}

	// Helper to write a file entry
	writeFileEntry := func(entry *FileEntry, offset uint32) error {
		buf := bytes.NewBuffer(fileTable[offset:offset])
		if err := binary.Write(buf, binary.LittleEndian, entry.Parent); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Sibling); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Offset); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Size); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.Hash); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.LittleEndian, entry.NameSize); err != nil {
			return err
		}
		copy(fileTable[offset+0x20:], entry.Name)
		return nil
	}

	// Populate directory table and hash table
	for dir := ctx.RootDir; dir != nil; dir = dir.Next {
		entry := &DirEntry{
			Parent:   dir.Parent.EntryOffset,
			Sibling:  getSiblingOffset(dir.Sibling),
			Child:    getChildOffset(dir.Child),
			File:     getFileOffset(dir.File),
			NameSize: uint32(len(dir.Name)),
			Name:     []byte(dir.Name),
		}

		// Calculate hash and update hash table
		if dir != ctx.RootDir { // Skip for root
			hash := calcPathHash(dir.Parent.EntryOffset, dir.Name, 0, len(dir.Name))
			hashIndex := hash % uint32(len(dirHashTable))
			entry.Hash = dirHashTable[hashIndex]
			dirHashTable[hashIndex] = dir.EntryOffset
		}

		if err := writeDirEntry(entry, dir.EntryOffset); err != nil {
			return fmt.Errorf("failed to write directory entry: %w", err)
		}
	}

	// Populate file table and hash table
	for file := ctx.Files; file != nil; file = file.Next {
		entry := &FileEntry{
			Parent:   file.Parent.EntryOffset,
			Sibling:  getSiblingOffset(file.Sibling),
			Offset:   file.Offset,
			Size:     file.Size,
			NameSize: uint32(len(file.Name)),
			Name:     []byte(file.Name),
		}

		// Calculate hash and update hash table
		hash := calcPathHash(file.Parent.EntryOffset, file.Name, 0, len(file.Name))
		hashIndex := hash % uint32(len(fileHashTable))
		entry.Hash = fileHashTable[hashIndex]
		fileHashTable[hashIndex] = file.EntryOffset

		if err := writeFileEntry(entry, file.EntryOffset); err != nil {
			return fmt.Errorf("failed to write file entry: %w", err)
		}
	}

	// Write all tables to file
	if err := writeTable(f, dirHashTable); err != nil {
		return fmt.Errorf("failed to write directory hash table: %w", err)
	}

	if err := writeTable(f, dirTable); err != nil {
		return fmt.Errorf("failed to write directory table: %w", err)
	}

	if err := writeTable(f, fileHashTable); err != nil {
		return fmt.Errorf("failed to write file hash table: %w", err)
	}

	if err := writeTable(f, fileTable); err != nil {
		return fmt.Errorf("failed to write file table: %w", err)
	}

	return nil
}

// Helper functions

func getSiblingOffset(sibling interface{}) uint32 {
	switch s := sibling.(type) {
	case *DirContext:
		if s == nil {
			return RomFSEntryEmpty
		}
		return s.EntryOffset
	case *FileContext:
		if s == nil {
			return RomFSEntryEmpty
		}
		return s.EntryOffset
	default:
		return RomFSEntryEmpty
	}
}

func getChildOffset(child *DirContext) uint32 {
	if child == nil {
		return RomFSEntryEmpty
	}
	return child.EntryOffset
}

func getFileOffset(file *FileContext) uint32 {
	if file == nil {
		return RomFSEntryEmpty
	}
	return file.EntryOffset
}

func writeTable(f *os.File, data interface{}) error {
	return binary.Write(f, binary.LittleEndian, data)
}
