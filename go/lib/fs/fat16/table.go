package fat16

import (
	"encoding/binary"
	"fmt"
)

type table struct {
	fatId      uint16
	eoc        uint16
	clusters   []uint16
	maxCluster uint16
}

func parseFat16Table(fatBytes []byte) table {
	maxCluster := uint16(len(fatBytes) / 2)
	fatTable := table{
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
			fatTable.clusters[i] = val
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

func (fs *FileSystem) allocateCluster() (uint16, error) {
	for cluster := uint16(2); cluster < fs.table.maxCluster; cluster++ {
		if fs.table.clusters[cluster] == 0x0000 {
			fs.table.clusters[cluster] = eoc

			// FIXME: ensure this is working
			for i := range uint32(fs.bootSector.BPB_NumFATs) {
				offset := fs.getFatSectorOffset(i)
				clusterOffset := offset + 4 + int64(cluster*2)
				toWrite := make([]byte, 2)
				binary.LittleEndian.PutUint16(toWrite, eoc)
				_, err := fs.file.WriteAt(toWrite, clusterOffset)
				if err != nil {
					return 0, err
				}
			}

			return cluster, nil
		}
	}

	return 0, fmt.Errorf("no available clusters")
}
