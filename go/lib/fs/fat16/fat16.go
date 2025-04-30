package fat12

import (
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/acheronfail/nxkit/lib/fs/boot_sector"
)

const (
	directoryEntrySize = 32
	eoc                = 0xFFFF
)

type table16 struct {
	fatId      uint16
	eoc        uint16
	clusters   []uint16
	maxCluster uint16
}

type FileSystem struct {
	file                     *os.File
	bootSector               boot_sector.BootSector
	bytesPerSector           uint32
	sectorsPerCluster        uint32
	fatSectorStart           uint32
	fatSectorCount           uint32
	rootDirectorySectorStart uint32
	rootDirectorySectorCount uint32
	dataSectorStart          uint32
	table                    table16
}

func parseFat16Table(fatBytes []byte) table16 {
	maxCluster := uint16(len(fatBytes) / 2)
	table := table16{
		fatId:      binary.LittleEndian.Uint16(fatBytes[0:2]),
		eoc:        binary.LittleEndian.Uint16(fatBytes[2:4]),
		clusters:   make([]uint16, maxCluster+1),
		maxCluster: maxCluster,
	}

	for i := uint16(2); i < maxCluster; i++ {
		start := i * 2
		end := start + 2
		val := binary.LittleEndian.Uint16(fatBytes[start:end])
		if val != 0 {
			table.clusters[i] = val
		}
	}

	return table
}

func NewFromPath(path string) (*FileSystem, error) {
	file, err := os.Open(path)
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
	bytesPerSector := uint32(bootSector.BPB_BytsPerSec)
	sectorsPerCluster := uint32(bootSector.BPB_SecPerClus)
	fatSectorStart := uint32(bootSector.BPB_RsvdSecCnt)
	fatSectorCount := bootSector.FatSectorSize() * uint32(bootSector.BPB_NumFATs)

	fatBytes := make([]byte, int64(fatSectorCount*bytesPerSector))
	_, err = file.ReadAt(fatBytes, int64(fatSectorStart*bytesPerSector))
	if err != nil {
		file.Close()
		return nil, err
	}

	table := parseFat16Table(fatBytes)
	rootDirectorySectorStart := fatSectorStart + fatSectorCount
	rootDirectorySectorCount := uint32((32*bootSector.BPB_RootEntCnt + bootSector.BPB_BytsPerSec - 1) / bootSector.BPB_BytsPerSec)
	dataSectorStart := rootDirectorySectorStart + rootDirectorySectorCount

	// TODONICE: validate both fats are identical

	return &FileSystem{
		file:                     file,
		bootSector:               *bootSector,
		bytesPerSector:           bytesPerSector,
		sectorsPerCluster:        sectorsPerCluster,
		fatSectorStart:           fatSectorStart,
		fatSectorCount:           fatSectorCount,
		rootDirectorySectorStart: rootDirectorySectorStart,
		rootDirectorySectorCount: rootDirectorySectorCount,
		dataSectorStart:          dataSectorStart,

		table: table,
	}, nil
}

func (fs *FileSystem) Close() error {
	return fs.file.Close()
}

func (fs *FileSystem) getRootDirectoryBytes() ([]byte, error) {
	start := fs.rootDirectorySectorStart * fs.bytesPerSector
	rootDirSize := fs.bootSector.BPB_RootEntCnt * directoryEntrySize
	b := make([]byte, rootDirSize)
	_, err := fs.file.ReadAt(b, int64(start))
	if err != nil {
		return nil, fmt.Errorf("could not read root directory bytes: %w", err)
	}

	return b, nil
}

func (fs *FileSystem) getDirectoryBytes(startCluster uint16) ([]byte, error) {
	bytesPerCluster := int64(fs.bytesPerSector * fs.sectorsPerCluster)

	var bytes []byte
	currentCluster := startCluster
	for {
		clusterSector := (fs.dataSectorStart + uint32(currentCluster-2)*fs.sectorsPerCluster)

		clusterBytes := make([]byte, bytesPerCluster)
		_, err := fs.file.ReadAt(clusterBytes, int64(clusterSector*fs.bytesPerSector))
		if err != nil {
			return nil, fmt.Errorf("failed to read cluster data: %w", err)
		}

		bytes = append(bytes, clusterBytes...)
		nextCluster := fs.table.clusters[currentCluster]

		// TODO: check between 0xfff8 - 0xffff
		// TODO: check invalid 0xfff7
		if nextCluster >= eoc {
			break
		}

		currentCluster = nextCluster
	}

	return bytes, nil
}

func (fs *FileSystem) ReadDir(path string) ([]DirectoryEntry, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	bytes, err := fs.getRootDirectoryBytes()
	if err != nil {
		return nil, fmt.Errorf("could not read root directory bytes: %w", err)
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		entries, err := fs.readDirectoryEntries(bytes)
		if err != nil {
			return nil, fmt.Errorf("could not read directory entries: %w", err)
		}

		found := false
		for _, entry := range entries {
			if !slices.ContainsFunc(entry.names(), func(name string) bool { return strings.EqualFold(name, part) }) {
				continue
			}

			if entry.IsDir() {
				bytes, err = fs.getDirectoryBytes(entry.clusterNumber())
				if err != nil {
					return nil, fmt.Errorf("could not read directory bytes: %w", err)
				}
				found = true
				break
			} else {
				return nil, fmt.Errorf("%s is not a directory", path)
			}

		}

		if !found {
			return nil, fmt.Errorf("no such file or directory %s", path)
		}
	}

	entries, err := fs.readDirectoryEntries(bytes)
	if err != nil {
		return nil, fmt.Errorf("could not read directory entries: %w", err)
	}

	return entries, nil
}
