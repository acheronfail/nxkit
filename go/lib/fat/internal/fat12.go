package internal

import "encoding/binary"

func NewFat12Table(fatBytes []byte) FatTable {
	eoc := uint32(0xff8)

	// calculate max clusters based on 12-bit entries
	maxCluster := uint32(len(fatBytes)*8) / 12

	fatId := uint32(binary.LittleEndian.Uint16(fatBytes[0:2]))
	clusters := make([]uint32, maxCluster+1)

	for cluster := uint32(2); cluster < maxCluster; cluster++ {
		entryOffset := cluster + (cluster / 2)
		if cluster%2 == 0 {
			// Even cluster: lower 12 bits of the 16-bit value
			val := binary.LittleEndian.Uint16(fatBytes[entryOffset:entryOffset+2]) & 0x0FFF
			clusters[cluster] = uint32(val)
		} else {
			// Odd cluster: upper 12 bits of the 16-bit value
			val := binary.LittleEndian.Uint16(fatBytes[entryOffset:entryOffset+2]) >> 4
			clusters[cluster] = uint32(val)
		}
	}

	return NewFatTable(fatId, eoc, maxCluster, clusters, func(fs *FileSystem, cluster, target uint32) error {
		for i := range uint32(fs.BootSector.BPB_NumFATs) {
			offset := fs.GetFatSectorOffset(i)
			clusterOffset := cluster + (cluster / 2)
			toWrite := make([]byte, 3)

			// read the existing 16-bit space where the cluster is stored
			clusterBytes16 := make([]byte, 2)
			_, err := fs.Backend.ReadAt(clusterBytes16, offset+int64(clusterOffset))
			if err != nil {
				return err
			}

			// this is a 16-bit value, but we only care about 12 bits of it
			clusterValue16 := binary.LittleEndian.Uint16(clusterBytes16)
			if cluster%2 == 0 {
				// Even cluster: write lower 12 bits of the target
				val := clusterValue16 & 0xF000
				val |= uint16(target & 0x0FFF)
				binary.LittleEndian.PutUint16(toWrite[:2], val)
				_, err := fs.BackendWriter.WriteAt(toWrite[:2], offset+int64(clusterOffset))
				if err != nil {
					return err
				}
			} else {
				// Odd cluster: write upper 12 bits of the target
				val := clusterValue16 & 0x000F
				val |= uint16((target & 0x0FFF) << 4)
				binary.LittleEndian.PutUint16(toWrite[:2], val)
				_, err := fs.BackendWriter.WriteAt(toWrite[:2], offset+int64(clusterOffset))
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}
