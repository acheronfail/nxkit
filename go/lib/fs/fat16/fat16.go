package fat16

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func (fs *FileSystem) clusterToSector(cluster uint16) uint32 {
	return (fs.dataSectorStart + uint32(cluster-2)*fs.sectorsPerCluster)
}

func (fs *FileSystem) splitPath(path string) ([]string, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	return parts, nil
}

type readDirResult struct {
	entries []DirectoryEntry
	bytes   []byte
	cluster *uint16
}

func (fs *FileSystem) readDir(path string, mkdir bool) (*readDirResult, error) {
	parts, err := fs.splitPath(path)
	if err != nil {
		return nil, err
	}

	var currentEntryCluster *uint16 = nil
	currentBytes, err := fs.getRootDirectoryBytes()
	if err != nil {
		return nil, fmt.Errorf("could not read root directory bytes: %w", err)
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		currentEntries, err := readDirectoryEntries(currentBytes)
		if err != nil {
			return nil, fmt.Errorf("could not read directory entries: %w", err)
		}

		found := false
		for _, entry := range currentEntries {
			if !slices.ContainsFunc(entry.names(), func(name string) bool { return strings.EqualFold(name, part) }) {
				continue
			}

			if entry.IsDir() {
				clusterNumber := entry.clusterNumber()
				currentBytes, err = fs.getClusterChainBytes(clusterNumber)
				if err != nil {
					return nil, fmt.Errorf("could not read directory bytes: %w", err)
				}

				currentEntryCluster = &clusterNumber
				found = true
				break
			} else {
				return nil, fmt.Errorf("%s is not a directory", path)
			}
		}

		if !found {
			if mkdir {
				nRequired := fs.numDirectoryEntriesRequired(part)
				freeIndex, newDirectoryBytes, err := fs.getAvailableDirectoryEntry(nRequired, currentEntryCluster, currentBytes)
				if err != nil {
					return nil, err
				}

				newDirectoryCluster, err := fs.allocateClusterChain(directoryEntrySize * 2)
				if err != nil {
					return nil, err
				}

				newDirectoryBytes, err = fs.writeDirectoryEntry(part, newDirectoryCluster, newDirectoryBytes, currentEntries, currentEntryCluster, freeIndex)
				if err != nil {
					return nil, fmt.Errorf("could not write directory entry: %w", err)
				}

				currentBytes = newDirectoryBytes
				currentEntryCluster = &newDirectoryCluster
			} else {
				return nil, fmt.Errorf("no such file or directory %s", path)
			}
		}
	}

	entries, err := readDirectoryEntries(currentBytes)
	if err != nil {
		return nil, fmt.Errorf("could not read directory entries: %w", err)
	}

	return &readDirResult{
		entries: entries,
		bytes:   currentBytes,
		cluster: currentEntryCluster,
	}, nil
}

func (fs *FileSystem) getAvailableDirectoryEntry(
	nRequired int,
	parentDirCluster *uint16,
	parentDirBytes []byte,
) (int, []byte, error) {
	freeIndex, ok := fs.findAvailableDirectoryEntry(parentDirBytes, nRequired)
	if !ok {
		// if we're in the root directory area we can't expand on fat16
		if parentDirCluster == nil {
			return 0, nil, fmt.Errorf("no available space for creating root directory entry")
		}

		// expand the current directory's cluster chain since we're out of space
		extraCluster, err := fs.allocateClusterChain(int64(nRequired * directoryEntrySize))
		if err != nil {
			return 0, nil, err
		}

		err = fs.writeClusterToFats(*parentDirCluster, extraCluster)
		if err != nil {
			return 0, nil, err
		}

		parentDirBytes, err = fs.getClusterChainBytes(*parentDirCluster)
		if err != nil {
			return 0, nil, fmt.Errorf("could not read directory bytes: %w", err)
		}

		// TODONICE: can optimise and return start of new cluster
		freeIndex, ok = fs.findAvailableDirectoryEntry(parentDirBytes, nRequired)
		if !ok {
			return 0, nil, fmt.Errorf("no available space for creating directory entry")
		}
	}

	return freeIndex, parentDirBytes, nil
}

func (fs *FileSystem) Mkdir(path string) error {
	_, err := fs.readDir(path, true)
	return err
}

func (fs *FileSystem) findEntry(path string) (*DirectoryEntry, *readDirResult, bool, error) {
	dirPath := filepath.Dir(path)
	baseName := filepath.Base(path)
	parent, err := fs.readDir(dirPath, false)
	if err != nil {
		return nil, nil, false, err
	}

	for _, existing := range parent.entries {
		if slices.ContainsFunc(existing.names(), func(name string) bool { return strings.EqualFold(name, baseName) }) {
			return &existing, parent, true, nil
		}
	}

	return nil, parent, false, nil
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
func (fs *FileSystem) removeEntryFromParent(entry *DirectoryEntry, parentDirCluster *uint16) error {
	entryIndex, parentDirBytes, err := fs.findIndexInParentBytes(entry, parentDirCluster)
	if err != nil {
		return err
	}

	// find all associated lfn entries
	sfnChecksum := calculateShortNameChecksum(entry.DIR_Name[:])
	lfnIndex := entryIndex - directoryEntrySize
	for lfnIndex >= 0 && parentDirBytes[lfnIndex+11] == 0x0F && parentDirBytes[lfnIndex+13] == sfnChecksum {
		parentDirBytes[lfnIndex] = 0xE5
		lfnIndex -= directoryEntrySize
	}

	// set first byte of entry to 0xE5 to mark as deleted
	parentDirBytes[entryIndex] = 0xE5

	// write back to disk
	err = fs.writeEntriesToParent(
		[]to32Bytes{},
		parentDirCluster,
		parentDirBytes,
		entryIndex,
	)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) removeEntryFromParentWithCluster(entry *DirectoryEntry, parentDirCluster *uint16) error {
	err := fs.removeEntryFromParent(entry, parentDirCluster)
	if err != nil {
		return err
	}

	// remove any allocated clusters from FATs
	cluster := entry.clusterNumber()
	if cluster == 0 {
		return nil
	}

	err = fs.deleteClusterChain(cluster)
	return err
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
