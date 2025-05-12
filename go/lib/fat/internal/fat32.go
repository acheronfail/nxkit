package internal

import "encoding/binary"

func NewFat32Table(fatBytes []byte) FatTable {
	eoc := uint32(0xffffff8)
	fatEntrySize := uint32(4)

	maxCluster := uint32(len(fatBytes)) / fatEntrySize
	fatId := binary.LittleEndian.Uint32(fatBytes[0:fatEntrySize])
	clusters := make([]uint32, maxCluster+1)

	for cluster := uint32(2); cluster < maxCluster; cluster++ {
		start := cluster * fatEntrySize
		end := start + fatEntrySize
		val := binary.LittleEndian.Uint32(fatBytes[start:end])
		if val != 0 {
			clusters[cluster] = val
		}
	}

	return NewFatTable(fatId, eoc, maxCluster, clusters, func(fs *FileSystem, cluster, target uint32) error {
		for i := range uint32(fs.BootSector.BPB_NumFATs) {
			offset := fs.GetFatSectorOffset(i)
			clusterOffset := offset + int64(cluster*fatEntrySize)
			toWrite := make([]byte, 4)
			binary.LittleEndian.PutUint32(toWrite, target)
			_, err := fs.BackendWriter.WriteAt(toWrite, clusterOffset)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
