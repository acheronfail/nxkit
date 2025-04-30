package fat16

import "encoding/binary"

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
