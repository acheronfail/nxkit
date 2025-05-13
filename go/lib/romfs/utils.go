package romfs

import (
	"os"
	"path/filepath"
)

func visitDir(parent *DirContext, ctx *RomFSContext) error {
	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(parent.Path, entry.Name())

		if entry.IsDir() {
			dir := &DirContext{
				Path:   path,
				Parent: parent,
			}
			ctx.NumDirs++
			ctx.DirTableSize += 0x18 + align(uint32(len(entry.Name())), 4)

			// Insert into sibling/next linked lists...
			insertDirEntry(parent, dir, ctx)

			if err := visitDir(dir, ctx); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				return err
			}

			file := &FileContext{
				Path:   path,
				Size:   uint64(info.Size()),
				Parent: parent,
			}
			ctx.NumFiles++
			ctx.FileTableSize += 0x20 + align(uint32(len(entry.Name())), 4)

			// Insert into sibling/next linked lists...
			insertFileEntry(parent, file, ctx)
		}
	}
	return nil
}

func getHashTableCount(numEntries uint32) uint32 {
	if numEntries < 3 {
		return 3
	}
	if numEntries < 19 {
		return numEntries | 1
	}
	count := numEntries
	for count%2 == 0 || count%3 == 0 || count%5 == 0 ||
		count%7 == 0 || count%11 == 0 || count%13 == 0 || count%17 == 0 {
		count++
	}
	return count
}

func calcPathHash(parent uint32, path string, start, length int) uint32 {
	hash := parent ^ 123456789
	for i := 0; i < length; i++ {
		hash = (hash >> 5) | (hash << 27)
		hash ^= uint32(path[start+i])
	}
	return hash
}

func align(offset uint32, alignment uint32) uint32 {
	mask := ^(alignment - 1)
	return (offset + (alignment - 1)) & mask
}

func align64(offset uint64, alignment uint64) uint64 {
	mask := ^(alignment - 1)
	return (offset + (alignment - 1)) & mask
}
