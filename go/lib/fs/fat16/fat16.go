package fat12

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/acheronfail/nxkit/lib/fs/boot_sector"
)

type table16 struct {
	fatId      uint16
	eoc        uint16
	clusters   []uint16
	maxCluster uint16
}

type FileSystem struct {
	file                     *os.File
	bytesPerSector           uint32
	fatStartSector           uint32
	rootDirectorySectorStart uint32
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
	fatStartSector := uint32(bootSector.BPB_RsvdSecCnt)
	fatSectorCount := bootSector.FatSectorSize() * uint32(bootSector.BPB_NumFATs)

	fatBytes := make([]byte, int64(fatSectorCount*bytesPerSector))
	_, err = file.ReadAt(fatBytes, int64(fatStartSector*bytesPerSector))
	if err != nil {
		file.Close()
		return nil, err
	}

	table := parseFat16Table(fatBytes)
	rootDirectorySectorStart := fatStartSector + fatSectorCount

	// TODONICE: validate both fats are identical

	return &FileSystem{
		bytesPerSector:           bytesPerSector,
		fatStartSector:           fatStartSector,
		rootDirectorySectorStart: rootDirectorySectorStart,
		file:                     file,
		table:                    table,
	}, nil
}

func (fs *FileSystem) Close() error {
	return fs.file.Close()
}

type DirectoryEntry struct{}

func (fs *FileSystem) ReadDir(path string) ([]DirectoryEntry, error) {
	// TODO: get cluster chain
	// TODO: get bytes
	// TODO: parse into directory entries
	panic("TODO: not implemented")
}
