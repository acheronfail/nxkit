package romfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
)

const (
	romFsHeaderSize = 0x50
	romFsEmptyEntry = 0xffffffff
)

type romFsHeader struct {
	headerSize          uint64
	dirHashTableOffset  uint64
	dirHashTableSize    uint64
	dirMetaTableOffset  uint64
	dirMetaTableSize    uint64
	fileHashTableOffset uint64
	fileHashTableSize   uint64
	fileMetaTableOffset uint64
	fileMetaTableSize   uint64
	dataOffset          uint64
}

func parseHeader(r io.Reader) (*romFsHeader, error) {
	var data [0x50]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return nil, fmt.Errorf("failed to read romFsHeader header: %w", err)
	}

	header := &romFsHeader{
		headerSize:          binary.LittleEndian.Uint64(data[0:0x8]),
		dirHashTableOffset:  binary.LittleEndian.Uint64(data[0x8:0x10]),
		dirHashTableSize:    binary.LittleEndian.Uint64(data[0x10:0x18]),
		dirMetaTableOffset:  binary.LittleEndian.Uint64(data[0x18:0x20]),
		dirMetaTableSize:    binary.LittleEndian.Uint64(data[0x20:0x28]),
		fileHashTableOffset: binary.LittleEndian.Uint64(data[0x28:0x30]),
		fileHashTableSize:   binary.LittleEndian.Uint64(data[0x30:0x38]),
		fileMetaTableOffset: binary.LittleEndian.Uint64(data[0x38:0x40]),
		fileMetaTableSize:   binary.LittleEndian.Uint64(data[0x40:0x48]),
		dataOffset:          binary.LittleEndian.Uint64(data[0x48:0x50]),
	}

	return header, nil
}

type romFsDirEntry struct {
	parent   uint32
	sibling  uint32
	child    uint32
	file     uint32
	hash     uint32
	nameSize uint32
	name     []byte
}

func parseDirEntry(r io.Reader) (*romFsDirEntry, error) {
	var data [0x18]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return nil, fmt.Errorf("failed to read romFsDirEntry header: %w", err)
	}

	dirEntry := &romFsDirEntry{
		parent:   binary.LittleEndian.Uint32(data[0x0:0x4]),
		sibling:  binary.LittleEndian.Uint32(data[0x4:0x8]),
		child:    binary.LittleEndian.Uint32(data[0x8:0xc]),
		file:     binary.LittleEndian.Uint32(data[0xc:0x10]),
		hash:     binary.LittleEndian.Uint32(data[0x10:0x14]),
		nameSize: binary.LittleEndian.Uint32(data[0x14:0x18]),
	}

	if dirEntry.nameSize > 0 {
		dirEntry.name = make([]byte, dirEntry.nameSize)
		if _, err := io.ReadFull(r, dirEntry.name); err != nil {
			return nil, fmt.Errorf("failed to read dir name: %s", err)
		}
	}

	return dirEntry, nil
}

type romFsFileEntry struct {
	parent   uint32
	sibling  uint32
	offset   uint64
	size     uint64
	hash     uint32
	nameSize uint32
	name     []byte
}

func parseFileEntry(r io.Reader) (*romFsFileEntry, error) {
	var data [0x20]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return nil, fmt.Errorf("failed to read romFsFileEntry header: %w", err)
	}

	fileEntry := &romFsFileEntry{
		parent:   binary.LittleEndian.Uint32(data[0x0:0x4]),
		sibling:  binary.LittleEndian.Uint32(data[0x4:0x8]),
		offset:   binary.LittleEndian.Uint64(data[0x8:0x10]),
		size:     binary.LittleEndian.Uint64(data[0x10:0x18]),
		hash:     binary.LittleEndian.Uint32(data[0x18:0x1c]),
		nameSize: binary.LittleEndian.Uint32(data[0x1c:0x20]),
	}

	if fileEntry.nameSize > 0 {
		fileEntry.name = make([]byte, fileEntry.nameSize)
		if _, err := io.ReadFull(r, fileEntry.name); err != nil {
			return nil, fmt.Errorf("failed to read file name: %w", err)
		}
	}

	return fileEntry, nil
}

func parseHashTable(data []byte) ([]uint32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("hash table size is not a multiple of 4")
	}

	count := len(data) / 4
	table := make([]uint32, count)

	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &table)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hash table: %w", err)
	}

	return table, nil
}

type RomFsReader interface {
	io.Reader
	io.ReaderAt
}

type RomFs struct {
	r              RomFsReader
	header         romFsHeader
	directories    []byte
	files          []byte
	directoryTable []uint32
	fileTable      []uint32
}

type romFsFile struct {
	Path string
	Size uint64
}
type romFsDir struct {
	Path string
}

type visitor struct {
	visitFile func(romFsFile)
	visitDir  func(romFsDir)
}

func (fs *RomFs) getFileEntry(offset uint32) (*romFsFileEntry, error) {
	return parseFileEntry(bytes.NewReader(fs.files[offset:]))
}

func (fs *RomFs) getDirEntry(offset uint32) (*romFsDirEntry, error) {
	return parseDirEntry(bytes.NewReader(fs.directories[offset:]))
}

func (fs *RomFs) visitFile(offset uint32, currentPath string, v visitor) error {
	for offset != romFsEmptyEntry {
		file, err := fs.getFileEntry(offset)
		if err != nil {
			return err
		}

		v.visitFile(romFsFile{
			Path: filepath.Join(currentPath, string(file.name)),
			Size: file.size,
		})

		offset = file.sibling
	}

	return nil
}

func (fs *RomFs) visitDir(offset uint32, currentPath string, v visitor) error {
	currentDir, err := fs.getDirEntry(offset)
	if err != nil {
		return fmt.Errorf("failed to read dir at offset %d: %s", offset, err)
	}

	currentPath = filepath.Join(currentPath, string(currentDir.name))
	v.visitDir(romFsDir{Path: currentPath})

	if currentDir.file != romFsEmptyEntry {
		if err := fs.visitFile(currentDir.file, currentPath, v); err != nil {
			return err
		}
	}
	if currentDir.child != romFsEmptyEntry {
		if err := fs.visitDir(currentDir.child, currentPath, v); err != nil {
			return err
		}
	}
	if currentDir.sibling != romFsEmptyEntry {
		if err := fs.visitDir(currentDir.sibling, currentPath, v); err != nil {
			return err
		}
	}

	return nil
}

func (fs *RomFs) ListEntries() (entries []string, err error) {
	err = fs.visitDir(0, "/", visitor{
		visitFile: func(f romFsFile) { entries = append(entries, f.Path) },
		visitDir:  func(d romFsDir) { entries = append(entries, d.Path) },
	})

	return
}

func (fs *RomFs) String() string {
	var sb strings.Builder

	sb.WriteString("ROMFS contents:\n")
	_ = fs.visitDir(0, "/", visitor{
		visitFile: func(f romFsFile) { sb.WriteString(f.Path + "\n") },
		visitDir:  func(d romFsDir) { sb.WriteString(d.Path + "\n") },
	})

	return sb.String()
}

func calcPathHash(parent uint32, name string) uint32 {
	hash := parent ^ 123456789
	for i := range len(name) {
		hash = (hash >> 5) | (hash << 27) // rotate right 5 bits
		hash ^= uint32(name[i])
	}
	return hash
}

func (fs *RomFs) findDir(path string) (*romFsDirEntry, uint32, error) {
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return nil, 0, fmt.Errorf("empty path")
	}

	// root
	if !slices.ContainsFunc(segments, func(s string) bool { return s != "" }) {
		root, err := fs.getDirEntry(0)
		return root, 0, err
	}

	// dir root is defined as 0
	var parentHash uint32

	// walk through all but the last segment (directories)
bucketLoop:
	for i, name := range segments {
		if name == "" {
			continue
		}

		hash := calcPathHash(parentHash, name)
		entryOffset := fs.directoryTable[hash%uint32(len(fs.directoryTable))]

		// iterate through directories in bucket until we find our directory
		for entryOffset != romFsEmptyEntry {
			dirEntry, err := fs.getDirEntry(entryOffset)
			if err != nil {
				return nil, 0, err
			}

			if string(dirEntry.name) == name {
				if i == len(segments)-1 {
					return dirEntry, entryOffset, nil
				}

				parentHash = hash
				continue bucketLoop
			}

			entryOffset = dirEntry.sibling
		}

		return nil, 0, fmt.Errorf("directory not found: %s", path)
	}

	return nil, 0, fmt.Errorf("directory not found: %s", path)

}

func (fs *RomFs) findFile(path string) (*romFsFileEntry, error) {
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	dirPath := filepath.Join(segments[:len(segments)-1]...)
	dirEntry, dirOffset, err := fs.findDir(dirPath)
	if err != nil {
		return nil, err
	}

	var parentHash uint32
	if dirOffset != 0 {
		parentHash = dirEntry.hash
	}

	// last segment in the path is the file
	fileName := segments[len(segments)-1]
	fileHash := calcPathHash(parentHash, fileName)
	entryOffset := fs.fileTable[fileHash%uint32(len(fs.fileTable))]

	// iterate through files in bucket until we find our file
	for entryOffset != romFsEmptyEntry {
		fileEntry, err := fs.getFileEntry(entryOffset)
		if err != nil {
			return nil, err
		}

		if fileEntry.parent == dirOffset && string(fileEntry.name) == fileName {
			return fileEntry, nil
		}

		entryOffset = fileEntry.sibling
	}

	return nil, fmt.Errorf("file not found: %s", path)
}

type Stat struct {
	path string
	size uint64
}

func (s *Stat) Path() string { return s.path }
func (s *Stat) Size() uint64 { return s.size }

func (fs *RomFs) StatFile(path string) (*Stat, error) {
	if _, _, err := fs.findDir(path); err == nil {
		return &Stat{path: path, size: 0}, nil
	}

	if fileEntry, err := fs.findFile(path); err == nil {
		return &Stat{path: path, size: fileEntry.size}, nil
	}

	return nil, fmt.Errorf("no such file or directory: %s", path)
}

func (fs *RomFs) OpenFile(path string) (*io.SectionReader, error) {
	fileEntry, err := fs.findFile(path)
	if err != nil {
		return nil, err
	}

	dataStart := int64(fs.header.dataOffset + fileEntry.offset)
	return io.NewSectionReader(fs.r, dataStart, int64(fileEntry.size)), nil
}

func NewRomFs(r RomFsReader) (*RomFs, error) {
	header, err := parseHeader(r)
	if err != nil {
		return nil, err
	}

	directories := make([]byte, header.dirMetaTableSize)
	if _, err := r.ReadAt(directories[:], int64(header.dirMetaTableOffset)); err != nil {
		return nil, fmt.Errorf("failed to read RomFs directories section: %w", err)
	}

	files := make([]byte, header.fileMetaTableSize)
	if _, err := r.ReadAt(files[:], int64(header.fileMetaTableOffset)); err != nil {
		return nil, fmt.Errorf("failed to read RomFs files section: %w", err)
	}

	directoryTable := make([]byte, header.dirHashTableSize)
	if _, err := r.ReadAt(directoryTable[:], int64(header.dirHashTableOffset)); err != nil {
		return nil, fmt.Errorf("failed to read RomFs directory table: %w", err)
	}
	hashDirectoryTable, err := parseHashTable(directoryTable)
	if err != nil {
		return nil, err
	}

	fileTable := make([]byte, header.fileHashTableSize)
	if _, err := r.ReadAt(fileTable[:], int64(header.fileHashTableOffset)); err != nil {
		return nil, fmt.Errorf("failed to read RomFs directory table: %w", err)
	}
	hashFileTable, err := parseHashTable(fileTable)
	if err != nil {
		return nil, err
	}

	return &RomFs{
		r:              r,
		header:         *header,
		directories:    directories,
		files:          files,
		directoryTable: hashDirectoryTable,
		fileTable:      hashFileTable,
	}, nil
}
