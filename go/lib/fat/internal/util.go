package internal

import (
	"fmt"
	"strings"
)

func (fs *FileSystem) clusterToSector(cluster uint32) uint32 {
	return (fs.DataSectorStart + (cluster-2)*fs.SectorsPerCluster)
}

func (fs *FileSystem) splitPath(path string) ([]string, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	return parts, nil
}

func (fs *FileSystem) GetFatSectorOffset(fatIndex uint32) int64 {
	return fs.BackendOffset + int64((fs.FatsSectorStart+(fatIndex*fs.FatSectorCount))*fs.BytesPerSector)
}

func (fs *FileSystem) getFatSectorBytes(fatIndex uint32) ([]byte, error) {
	bytes := make([]byte, int64(fs.FatSectorCount*fs.BytesPerSector))
	_, err := fs.Backend.ReadAt(bytes, fs.GetFatSectorOffset(fatIndex))
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func GetRootDirectoryBytesDedicatedArea(fs *FileSystem) ([]byte, error) {
	start := fs.BackendOffset + int64(fs.RootDirectorySectorStart*fs.BytesPerSector)
	rootDirSize := fs.BootSector.BPB_RootEntCnt * FatDirectoryEntrySize
	b := make([]byte, rootDirSize)
	_, err := fs.Backend.ReadAt(b, start)
	if err != nil {
		return nil, fmt.Errorf("could not read root directory bytes: %w", err)
	}

	return b, nil
}
