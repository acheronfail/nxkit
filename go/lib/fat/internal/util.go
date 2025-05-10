package internal

import (
	"fmt"
	"strings"
)

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
