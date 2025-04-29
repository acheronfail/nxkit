package fat12

import (
	"fmt"
	"os"

	"github.com/acheronfail/nxkit/lib/fs/boot_sector"
)

type FileSystem struct {
	file *os.File
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
		return nil, fmt.Errorf("not a FAT12 filesystem, got FAT%d", bootSector.FatType())
	}

	// TODO: parse fat table and save cluster map into struct for quick access
	// TODO: calculate and save root directory location

	return &FileSystem{
		file: file,
	}, nil
}

func (fs *FileSystem) Close() error {
	return fs.file.Close()
}
