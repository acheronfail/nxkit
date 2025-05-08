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
	rand                     *rand.Rand
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
		rand:                     rand.New(rand.NewSource(0)),
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

		currentEntries, err := fs.readDirectoryEntries(currentBytes)
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

	entries, err := fs.readDirectoryEntries(currentBytes)
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

// use os.OpenFile flags
// currently supports O_RDONLY and O_CREATE
func (fs *FileSystem) OpenFile(path string, flags int) (fs.File, error) {
	dirPath := filepath.Dir(path)
	baseName := filepath.Base(path)
	parent, err := fs.readDir(dirPath, false)
	if err != nil {
		return nil, err
	}

	t := asFatTime(time.Now())
	newFileEntry, err := fs.createNewEntry(baseName, t, 0x00, 0, []DirectoryEntry{})
	if err != nil {
		return nil, err
	}

	for _, existing := range parent.entries {
		lfnMatched := strings.EqualFold(existing.LongName(), newFileEntry.LongName())
		sfnMatched := strings.EqualFold(existing.ShortName(), newFileEntry.ShortName())
		if lfnMatched || sfnMatched {
			if existing.IsDir() {
				return nil, fmt.Errorf("failed to open '%s': is a directory", path)
			}

			return &fatFile{
				DirectoryEntry:   existing,
				parentDirCluster: parent.cluster,
				fs:               fs,
			}, nil
		}
	}

	if flags&os.O_CREATE == 0 {
		return nil, fmt.Errorf("no such file or directory %s", path)
	}

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
		fs:               fs,
	}, nil
}
