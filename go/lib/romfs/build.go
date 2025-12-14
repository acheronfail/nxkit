package romfs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Build creates a RomFS image from the input directory and writes it to the output path.
func Build(inDirectory string, outputPath string) error {
	fOut, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer fOut.Close()

	// Use a buffered writer for better performance
	bw := bufio.NewWriter(fOut)
	defer bw.Flush()

	return buildRomFsIntoWriter(inDirectory, bw)
}

type dirCtx struct {
	path        string // Full path on disk
	name        string // Name in RomFS (just the directory name)
	entryOffset uint32
	parent      *dirCtx
	child       *dirCtx
	sibling     *dirCtx
	file        *fileCtx
}

type fileCtx struct {
	path        string // Full path on disk
	name        string // Name in RomFS (just the file name)
	entryOffset uint32
	offset      uint64
	size        uint64
	parent      *dirCtx
	sibling     *fileCtx
}

type buildCtx struct {
	files             []*fileCtx
	dirs              []*dirCtx
	dirTableSize      uint64
	fileTableSize     uint64
	dirHashTableSize  uint64
	fileHashTableSize uint64
	filePartitionSize uint64
	numDirs           uint32
	numFiles          uint32
}

func buildRomFsIntoWriter(inDirectory string, w *bufio.Writer) error {
	rootCtx := &dirCtx{
		path: inDirectory,
		name: "", // Root has no name
	}
	// In C, root_ctx->parent = root_ctx;
	rootCtx.parent = rootCtx

	bCtx := &buildCtx{
		dirTableSize: 0x18, // Root directory
		numDirs:      1,
		dirs:         []*dirCtx{rootCtx},
	}

	if err := visitDir(rootCtx, bCtx); err != nil {
		return err
	}

	// Sort directories by path (excluding root which is always first)
	if len(bCtx.dirs) > 1 {
		others := bCtx.dirs[1:]
		sort.Slice(others, func(i, j int) bool {
			// Compare paths. Note: we should probably compare the relative path from root,
			// but since they all start with root path, comparing full path is fine.
			// However, we need to ensure separator consistency if we were on Windows.
			// But we are using filepath.Join which uses OS separator.
			// The C code uses `sum_path` which seems to use `/` or `\` depending on OS?
			// C code: filepath_os_append uses PATH_SEPERATOR.
			// So sorting by OS path is correct.
			return others[i].path < others[j].path
		})
	}

	// Sort files by path
	sort.Slice(bCtx.files, func(i, j int) bool {
		return bCtx.files[i].path < bCtx.files[j].path
	})

	dirHashTableEntryCount := getHashTableCount(bCtx.numDirs)
	fileHashTableEntryCount := getHashTableCount(bCtx.numFiles)
	bCtx.dirHashTableSize = 4 * uint64(dirHashTableEntryCount)
	bCtx.fileHashTableSize = 4 * uint64(fileHashTableEntryCount)

	// Calculate metadata
	// Determine file offsets
	entryOffset := uint32(0)
	for _, curFile := range bCtx.files {
		bCtx.filePartitionSize = align64(bCtx.filePartitionSize, 0x10)
		curFile.offset = bCtx.filePartitionSize
		bCtx.filePartitionSize += curFile.size
		curFile.entryOffset = entryOffset
		entryOffset += 0x20 + align(uint32(len(curFile.name)), 4)
	}

	// Determine dir offsets
	entryOffset = 0
	for _, curDir := range bCtx.dirs {
		curDir.entryOffset = entryOffset
		if curDir == rootCtx {
			entryOffset += 0x18
		} else {
			entryOffset += 0x18 + align(uint32(len(curDir.name)), 4)
		}
	}

	// Prepare tables
	dirHashTable := make([]uint32, dirHashTableEntryCount)
	for i := range dirHashTable {
		dirHashTable[i] = romFsEmptyEntry
	}

	fileHashTable := make([]uint32, fileHashTableEntryCount)
	for i := range fileHashTable {
		fileHashTable[i] = romFsEmptyEntry
	}

	// Populate file tables and hash table
	fileTableData := new(bytes.Buffer)
	for _, curFile := range bCtx.files {
		// romfs_fentry_t
		parentOffset := curFile.parent.entryOffset
		siblingOffset := uint32(romFsEmptyEntry)
		if curFile.sibling != nil {
			siblingOffset = curFile.sibling.entryOffset
		}

		binary.Write(fileTableData, binary.LittleEndian, parentOffset)
		binary.Write(fileTableData, binary.LittleEndian, siblingOffset)
		binary.Write(fileTableData, binary.LittleEndian, curFile.offset)
		binary.Write(fileTableData, binary.LittleEndian, curFile.size)

		hash := calcPathHash(curFile.parent.entryOffset, curFile.name)
		bucket := hash % fileHashTableEntryCount
		binary.Write(fileTableData, binary.LittleEndian, fileHashTable[bucket])
		fileHashTable[bucket] = curFile.entryOffset

		nameSize := uint32(len(curFile.name))
		binary.Write(fileTableData, binary.LittleEndian, nameSize)
		fileTableData.Write([]byte(curFile.name))

		// Padding
		padding := align(nameSize, 4) - nameSize
		for range padding {
			fileTableData.WriteByte(0)
		}
	}

	// Populate dir tables and hash table
	dirTableData := new(bytes.Buffer)
	for _, curDir := range bCtx.dirs {
		// romfs_direntry_t
		parentOffset := curDir.parent.entryOffset
		siblingOffset := uint32(romFsEmptyEntry)
		if curDir.sibling != nil {
			siblingOffset = curDir.sibling.entryOffset
		}
		childOffset := uint32(romFsEmptyEntry)
		if curDir.child != nil {
			childOffset = curDir.child.entryOffset
		}
		fileOffset := uint32(romFsEmptyEntry)
		if curDir.file != nil {
			fileOffset = curDir.file.entryOffset
		}

		binary.Write(dirTableData, binary.LittleEndian, parentOffset)
		binary.Write(dirTableData, binary.LittleEndian, siblingOffset)
		binary.Write(dirTableData, binary.LittleEndian, childOffset)
		binary.Write(dirTableData, binary.LittleEndian, fileOffset)

		hash := uint32(0)
		nameSize := uint32(0)
		if curDir != rootCtx {
			nameSize = uint32(len(curDir.name))
			hash = calcPathHash(curDir.parent.entryOffset, curDir.name)
			bucket := hash % dirHashTableEntryCount
			binary.Write(dirTableData, binary.LittleEndian, dirHashTable[bucket])
			dirHashTable[bucket] = curDir.entryOffset
		} else {
			// Root dir hash
			hash = calcPathHash(0, "")
			bucket := hash % dirHashTableEntryCount
			binary.Write(dirTableData, binary.LittleEndian, dirHashTable[bucket])
			dirHashTable[bucket] = curDir.entryOffset
		}

		binary.Write(dirTableData, binary.LittleEndian, nameSize)
		if nameSize > 0 {
			dirTableData.Write([]byte(curDir.name))
		}

		// Padding
		padding := align(nameSize, 4) - nameSize
		for i := uint32(0); i < padding; i++ {
			dirTableData.WriteByte(0)
		}
	}

	// Header
	header := romFsHeader{}
	header.headerSize = 0x50 // sizeof(header)
	header.fileHashTableSize = bCtx.fileHashTableSize
	header.fileMetaTableSize = uint64(fileTableData.Len())
	header.dirHashTableSize = bCtx.dirHashTableSize
	header.dirMetaTableSize = uint64(dirTableData.Len())
	header.dataOffset = 0x200 // ROMFS_FILEPARTITION_OFS

	dirHashTableOffset := align64(bCtx.filePartitionSize+header.dataOffset, 4)
	header.dirHashTableOffset = dirHashTableOffset
	header.dirMetaTableOffset = header.dirHashTableOffset + header.dirHashTableSize
	header.fileHashTableOffset = header.dirMetaTableOffset + header.dirMetaTableSize
	header.fileMetaTableOffset = header.fileHashTableOffset + header.fileHashTableSize

	// Write Header
	if err := binary.Write(w, binary.LittleEndian, header.headerSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.dirHashTableOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.dirHashTableSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.dirMetaTableOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.dirMetaTableSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.fileHashTableOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.fileHashTableSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.fileMetaTableOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.fileMetaTableSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, header.dataOffset); err != nil {
		return err
	}

	// Write Files
	currentPos := uint64(0x50)
	padding := header.dataOffset - currentPos
	for i := uint64(0); i < padding; i++ {
		w.WriteByte(0)
	}
	currentPos += padding

	dataStart := header.dataOffset
	currentDataOffset := uint64(0)

	for _, curFile := range bCtx.files {
		// Align
		alignedOffset := align64(currentDataOffset, 0x10)
		padding := alignedOffset - currentDataOffset
		for range padding {
			w.WriteByte(0)
		}
		currentDataOffset += padding

		if currentDataOffset != curFile.offset {
			return fmt.Errorf("offset mismatch for file %s: expected %d, got %d", curFile.name, curFile.offset, currentDataOffset)
		}

		fIn, err := os.Open(curFile.path)
		if err != nil {
			return fmt.Errorf("failed to open input file %s: %w", curFile.path, err)
		}

		n, err := io.Copy(w, fIn)
		fIn.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", curFile.name, err)
		}
		if uint64(n) != curFile.size {
			return fmt.Errorf("wrote partial file %s", curFile.name)
		}
		currentDataOffset += curFile.size
	}

	// Write Metadata
	currentPos = dataStart + currentDataOffset
	padding = header.dirHashTableOffset - currentPos
	for i := uint64(0); i < padding; i++ {
		w.WriteByte(0)
	}

	// Write Dir Hash Table
	if err := binary.Write(w, binary.LittleEndian, dirHashTable); err != nil {
		return err
	}

	// Write Dir Table
	if _, err := w.Write(dirTableData.Bytes()); err != nil {
		return err
	}

	// Write File Hash Table
	if err := binary.Write(w, binary.LittleEndian, fileHashTable); err != nil {
		return err
	}

	// Write File Table
	if _, err := w.Write(fileTableData.Bytes()); err != nil {
		return err
	}

	return nil
}

func visitDir(parent *dirCtx, bCtx *buildCtx) error {
	entries, err := os.ReadDir(parent.path)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %w", parent.path, err)
	}

	var dirs []*dirCtx
	var files []*fileCtx

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		fullPath := filepath.Join(parent.path, name)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.IsDir() {
			dirs = append(dirs, &dirCtx{
				path:   fullPath,
				name:   name,
				parent: parent,
			})
		} else {
			files = append(files, &fileCtx{
				path:   fullPath,
				name:   name,
				parent: parent,
				size:   uint64(info.Size()),
			})
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].name < dirs[j].name
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})

	var prevDir *dirCtx
	for _, d := range dirs {
		if prevDir == nil {
			parent.child = d
		} else {
			prevDir.sibling = d
		}
		prevDir = d

		bCtx.numDirs++
		bCtx.dirs = append(bCtx.dirs, d)

		if err := visitDir(d, bCtx); err != nil {
			return err
		}
	}

	var prevFile *fileCtx
	for _, f := range files {
		if prevFile == nil {
			parent.file = f
		} else {
			prevFile.sibling = f
		}
		prevFile = f

		bCtx.numFiles++
		bCtx.files = append(bCtx.files, f)
	}

	return nil
}

func align(offset, alignment uint32) uint32 {
	mask := ^(alignment - 1)
	return (offset + (alignment - 1)) & mask
}

func align64(offset, alignment uint64) uint64 {
	mask := ^(alignment - 1)
	return (offset + (alignment - 1)) & mask
}

func getHashTableCount(numEntries uint32) uint32 {
	if numEntries < 3 {
		return 3
	} else if numEntries < 19 {
		return numEntries | 1
	}
	count := numEntries
	for {
		if count%2 == 0 || count%3 == 0 || count%5 == 0 || count%7 == 0 || count%11 == 0 || count%13 == 0 || count%17 == 0 {
			count++
		} else {
			break
		}
	}
	return count
}
