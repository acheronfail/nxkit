package fat16

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/acheronfail/nxkit/lib/fs"
	"github.com/acheronfail/nxkit/lib/fs/boot_sector"
)

const (
	directoryEntrySize = 32
	eoc                = 0xFFF8
)

type FileSystem struct {
	file                     *os.File
	bootSector               boot_sector.BootSector
	bytesPerSector           uint32
	bytesPerCluster          int64
	sectorsPerCluster        uint32
	fatSectorCount           uint32
	fatsSectorStart          uint32
	fatsSectorCount          uint32
	rootDirectorySectorStart uint32
	rootDirectorySectorCount uint32
	dataSectorStart          uint32
	table                    table
	randIntn                 *func(n int) int
}

func NewFromPath(path string) (*FileSystem, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	bootSectorBytes := make([]byte, 512)
	_, err = file.ReadAt(bootSectorBytes, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	bootSector, err := boot_sector.NewBootSector(bootSectorBytes)
	if err != nil {
		file.Close()
		return nil, err
	}

	if bootSector.FatType() != boot_sector.Fat16 {
		file.Close()
		return nil, fmt.Errorf("not a FAT16 filesystem, got FAT%d", bootSector.FatType())
	}

	// Calculate the start of the FAT tables
	sectorsPerCluster := uint32(bootSector.BPB_SecPerClus)
	bytesPerSector := uint32(bootSector.BPB_BytsPerSec)
	bytesPerCluster := int64(bytesPerSector * sectorsPerCluster)

	fatSectorCount := bootSector.FatSectorCount()
	fatsSectorStart := uint32(bootSector.BPB_RsvdSecCnt)
	fatsSectorCount := fatSectorCount * uint32(bootSector.BPB_NumFATs)

	rootDirectorySectorStart := fatsSectorStart + fatsSectorCount
	rootDirectorySectorCount := uint32((32*bootSector.BPB_RootEntCnt + bootSector.BPB_BytsPerSec - 1) / bootSector.BPB_BytsPerSec)
	dataSectorStart := rootDirectorySectorStart + rootDirectorySectorCount

	randFunc := func(n int) int { return rand.Intn(n) }
	fs := &FileSystem{
		file:                     file,
		bootSector:               *bootSector,
		bytesPerSector:           bytesPerSector,
		bytesPerCluster:          bytesPerCluster,
		sectorsPerCluster:        sectorsPerCluster,
		fatSectorCount:           fatSectorCount,
		fatsSectorStart:          fatsSectorStart,
		fatsSectorCount:          fatsSectorCount,
		rootDirectorySectorStart: rootDirectorySectorStart,
		rootDirectorySectorCount: rootDirectorySectorCount,
		dataSectorStart:          dataSectorStart,
		randIntn:                 &randFunc,
	}

	// TODONICE: support more than 2 fats
	fat1Bytes, err := fs.getFatSectorBytes(1)
	if err != nil {
		file.Close()
		return nil, err
	}

	// TODONICE: validate both fats are identical
	fs.table = parseFat16Table(fat1Bytes)

	return fs, nil
}

func (fs *FileSystem) Info() map[string]any {
	return map[string]any{
		"bootSector":               fs.bootSector,
		"bytesPerSector":           fs.bytesPerSector,
		"bytesPerCluster":          fs.bytesPerCluster,
		"sectorsPerCluster":        fs.sectorsPerCluster,
		"fatSectorCount":           fs.fatSectorCount,
		"fatsSectorStart":          fs.fatsSectorStart,
		"fatsSectorCount":          fs.fatsSectorCount,
		"rootDirectorySectorStart": fs.rootDirectorySectorStart,
		"rootDirectorySectorCount": fs.rootDirectorySectorCount,
	}
}

func (fs *FileSystem) Close() error {
	return fs.file.Close()
}

func (fs *FileSystem) ReadDir(path string) ([]DirectoryEntry, error) {
	result, err := fs.readDir(path, false)
	if err != nil {
		return nil, err
	}

	return result.entries, nil
}

func (fs *FileSystem) Mkdir(path string) error {
	_, err := fs.readDir(path, true)
	return err
}

// use os.OpenFile flags
// currently supports O_RDONLY and O_CREATE
func (fs *FileSystem) OpenFile(path string, flags int) (fs.File, error) {
	canWrite := flags&os.O_WRONLY != 0 || flags&os.O_RDWR != 0

	existing, parent, found, err := fs.findEntry(path)
	if err != nil {
		return nil, err
	}

	if found {
		if existing.IsDir() {
			return nil, fmt.Errorf("failed to open '%s': is a directory", path)
		}

		// truncate file if requested
		if flags&os.O_TRUNC != 0 {
			if !canWrite {
				return nil, os.ErrPermission
			}

			index, parentDirBytes, err := fs.findIndexInParentBytes(existing, parent.cluster)
			if err != nil {
				return nil, err
			}

			// save existing cluster
			cluster := existing.clusterNumber()

			// set file size to 0 and cluster chain to 0
			existing.DIR_FileSize = 0
			existing.setCluster(0)

			// write truncated entry back to parent
			err = fs.writeEntriesToParent(
				[]to32Bytes{existing},
				parent.cluster,
				parentDirBytes,
				index,
			)
			if err != nil {
				return nil, err
			}

			// remove any allocated clusters from FATs
			if cluster != 0 {
				err = fs.deleteClusterChain(cluster)
				if err != nil {
					return nil, err
				}
			}
		}

		return &fatFile{
			DirectoryEntry:   *existing,
			parentDirCluster: parent.cluster,
			accessMode:       flags & 0b11,
			fs:               fs,
		}, nil
	}

	if parent == nil {
		return nil, fmt.Errorf("failed to open '%s': no such file or directory", path)
	}

	if flags&os.O_CREATE == 0 {
		return nil, fmt.Errorf("no such file or directory %s", path)
	}

	if !canWrite {
		return nil, os.ErrPermission
	}

	t := asFatTime(time.Now())
	baseName := filepath.Base(path)
	newFileEntry := fs.createNewEntry(baseName, t, 0x00, 0, []DirectoryEntry{})
	nRequired := fs.numDirectoryEntriesRequired(baseName)
	startIndex, newParentDirBytes, err := fs.getAvailableDirectoryEntry(nRequired, parent.cluster, parent.bytes)
	if err != nil {
		return nil, err
	}

	err = fs.writeEntryWithLfnToParent(baseName, newFileEntry, parent.cluster, newParentDirBytes, startIndex)
	if err != nil {
		return nil, err
	}

	return &fatFile{
		DirectoryEntry:   *newFileEntry,
		parentDirCluster: parent.cluster,
		accessMode:       flags & 0b11,
		fs:               fs,
	}, nil
}

func (fs *FileSystem) Stat(path string) (fs.Stat, error) {
	entry, _, found, err := fs.findEntry(path)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("no such file or directory %s", path)
	}

	return entry, nil
}

func (fs *FileSystem) Unlink(path string) error {
	entry, parent, found, err := fs.findEntry(path)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("no such file or directory %s", path)
	}
	if entry.IsDir() {
		return fmt.Errorf("cannot unlink directory %s", path)
	}
	if entry.IsReadOnly() {
		return fmt.Errorf("cannot unlink read-only file %s", path)
	}

	return fs.removeEntryFromParentWithCluster(entry, parent.cluster)
}

func (fs *FileSystem) Rmdir(path string) error {
	entry, parent, found, err := fs.findEntry(path)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("no such file or directory %s", path)
	}
	if !entry.IsDir() {
		return fmt.Errorf("cannot rmdir file %s", path)
	}
	if entry.IsReadOnly() {
		return fmt.Errorf("cannot rmdir read-only directory %s", path)
	}

	// check if directory is empty
	entryBytes, err := fs.getClusterChainBytes(entry.clusterNumber())
	if err != nil {
		return err
	}

	entries, err := readDirectoryEntries(entryBytes)
	if err != nil {
		return err
	}

	// at least 2 for . and ..
	if len(entries) > 2 {
		return fmt.Errorf("directory %s is not empty", path)
	}

	return fs.removeEntryFromParentWithCluster(entry, parent.cluster)
}

// Rename renames (moves) srcPath to dstPath.
// If dstPath already exists and is not a directory, it is replaced.
func (fs *FileSystem) Rename(srcPath, dstPath string) error {
	// find source entry
	srcEntry, srcParent, found, err := fs.findEntry(srcPath)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("no such file or directory %s", srcPath)
	}
	if srcEntry.IsReadOnly() {
		return fmt.Errorf("cannot rename read-only file %s", srcPath)
	}

	// find destination entry (if it exists)
	dstEntry, dstParent, found, err := fs.findEntry(dstPath)
	if err != nil {
		return err
	}

	if found {
		if dstEntry.IsDir() {
			return fmt.Errorf("cannot rename to directory %s", dstPath)
		}
		if dstEntry.IsReadOnly() {
			return fmt.Errorf("cannot rename to read-only file %s", dstPath)
		}

		// remove overwritten entry and any associated clusters here
		err = fs.removeEntryFromParentWithCluster(dstEntry, dstParent.cluster)
		if err != nil {
			return err
		}
	}

	// write source entry to destination parent directory
	baseName := filepath.Base(dstPath)
	nRequired := fs.numDirectoryEntriesRequired(baseName)
	startIndex, newParentDirBytes, err := fs.getAvailableDirectoryEntry(nRequired, dstParent.cluster, dstParent.bytes)
	if err != nil {
		return err
	}

	newShortNameBytes, ntRes := fs.createShortNameBytes(baseName, dstParent.entries)

	// create new entry from src, with new name and copy over cluster number
	newEntry := srcEntry.clone()
	copy(newEntry.DIR_Name[:], newShortNameBytes[:])
	newEntry.setCluster(srcEntry.clusterNumber())
	newEntry.DIR_NTRes = ntRes

	err = fs.writeEntryWithLfnToParent(baseName, &newEntry, dstParent.cluster, newParentDirBytes, startIndex)
	if err != nil {
		return err
	}

	// remove source entry from source parent directory
	return fs.removeEntryFromParent(srcEntry, srcParent.cluster)
}

func (fs *FileSystem) GetVolumeId() (string, error) {
	dirBytes, err := fs.getRootDirectoryBytes()
	if err != nil {
		return "", err
	}

	for i := 0; i < len(dirBytes); i += directoryEntrySize {
		// at the end, didn't find it
		if dirBytes[i] == 0x00 {
			break
		}

		// skip deleted or lfn entries
		if dirBytes[i] == 0xE5 || dirBytes[i+11] == 0x0F {
			continue
		}

		// find the volume id entry
		if dirBytes[i+11]&0x08 == 0x08 {
			entry := DirectoryEntry{fatDirectoryEntry: fatDirectoryEntryFromBytes(dirBytes[i:])}
			return entry.ShortName(), nil
		}
	}

	return "", os.ErrNotExist
}

// TODO: reformat fs
// TODO: utils:
//	- defragmentation operation (since renames and such will cause fragmentation with lfn support)
//		or, alternatively re-write the entire directory clusters in a de-fragmented state each time
//	- clean up garbage lfn entries (when other systems without lfn use the disk)
