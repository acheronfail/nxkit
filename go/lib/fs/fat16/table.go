package fat16

import (
	"encoding/binary"
	"fmt"
)

const (
	entrySize = uint16(2)
)

type table struct {
	fatId      uint16
	eoc        uint16
	clusters   []uint16
	maxCluster uint16
}

func parseFat16Table(fatBytes []byte) table {
	maxCluster := uint16(len(fatBytes)) / entrySize
	fatTable := table{
		fatId:      binary.LittleEndian.Uint16(fatBytes[0:entrySize]),
		eoc:        binary.LittleEndian.Uint16(fatBytes[entrySize : entrySize*2]),
		clusters:   make([]uint16, maxCluster+1),
		maxCluster: maxCluster,
	}

	for cluster := uint16(2); cluster < maxCluster; cluster++ {
		start := cluster * entrySize
		end := start + entrySize
		val := binary.LittleEndian.Uint16(fatBytes[start:end])
		if val != 0 {
			fatTable.clusters[cluster] = val
		}
	}

	return fatTable
}

func (fs *FileSystem) getFatSectorOffset(fatIndex uint32) int64 {
	return int64((fs.fatsSectorStart + (fatIndex * fs.fatSectorCount)) * fs.bytesPerSector)
}

func (fs *FileSystem) getFatSectorBytes(fatIndex uint32) ([]byte, error) {
	bytes := make([]byte, int64(fs.fatSectorCount*fs.bytesPerSector))
	_, err := fs.file.ReadAt(bytes, fs.getFatSectorOffset(fatIndex))
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (fs *FileSystem) writeClusterToFats(cluster, target uint16) error {
	// set in memory cluster table
	fs.table.clusters[cluster] = target

	// also write back to all FATs
	for i := range uint32(fs.bootSector.BPB_NumFATs) {
		offset := fs.getFatSectorOffset(i)
		clusterOffset := offset + int64(cluster*entrySize)
		toWrite := make([]byte, 2)
		binary.LittleEndian.PutUint16(toWrite, target)
		_, err := fs.file.WriteAt(toWrite, clusterOffset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) allocateNextFreeCluster() (uint16, error) {
	for cluster := uint16(2); cluster < fs.table.maxCluster; cluster++ {
		if fs.table.clusters[cluster] == 0x0000 {
			err := fs.writeClusterToFats(cluster, eoc)
			if err != nil {
				return 0, err
			}

			return cluster, nil
		}
	}

	return 0, fmt.Errorf("no available clusters")
}
