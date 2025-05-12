package internal

import "encoding/binary"

func NewFat16Table(fatBytes []byte) FatTable {
	fatEntrySize := uint32(2)
	eoc := uint32(0xfff8)

	maxCluster := uint32(len(fatBytes)) / fatEntrySize
	fatId := uint32(binary.LittleEndian.Uint16(fatBytes[0:fatEntrySize]))
	clusters := make([]uint32, maxCluster+1)

	for cluster := uint32(2); cluster < maxCluster; cluster++ {
		start := cluster * fatEntrySize
		end := start + fatEntrySize
		val := binary.LittleEndian.Uint16(fatBytes[start:end])
		if val != 0 {
			clusters[cluster] = uint32(val)
		}
	}

	return NewFatTable(fatId, eoc, maxCluster, clusters, func(fs *FileSystem, cluster, target uint32) error {
		for i := range uint32(fs.BootSector.BPB_NumFATs) {
			offset := fs.GetFatSectorOffset(i)
			clusterOffset := offset + int64(cluster*fatEntrySize)
			toWrite := make([]byte, 2)
			binary.LittleEndian.PutUint16(toWrite, uint16(target))
			_, err := fs.BackendWriter.WriteAt(toWrite, clusterOffset)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
