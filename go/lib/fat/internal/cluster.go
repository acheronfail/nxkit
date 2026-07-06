package internal

import (
	"fmt"
)

func (fs *FileSystem) writeClusterToFats(cluster, target uint32) error {
	return fs.Table.SetClusterTarget(fs, cluster, target)
}

func (fs *FileSystem) allocateClusterChain(bytesRequired int64) (uint32, error) {
	nClustersAllocated := 0
	nClustersRequired := max(1, bytesRequired/fs.BytesPerCluster)

	prevCluster := fs.Table.GetEoc()
	for cluster := uint32(2); cluster < fs.Table.GetMaxCluster(); cluster++ {
		if fs.Table.GetClusterTarget(cluster) == 0x0000 {
			err := fs.writeClusterToFats(cluster, prevCluster)
			if err != nil {
				return 0, err
			}

			nClustersAllocated++
			if nClustersAllocated == int(nClustersRequired) {
				return cluster, nil
			}

			prevCluster = cluster
		}
	}

	return 0, fmt.Errorf("no available clusters")
}

func (fs *FileSystem) getClusterChain(startCluster uint32) ([]uint32, error) {
	var clusters []uint32
	currentCluster := startCluster
	for {
		clusters = append(clusters, currentCluster)
		nextCluster := fs.Table.GetClusterTarget(currentCluster)

		if fs.Table.IsEoc(nextCluster) {
			break
		}
		if nextCluster < 2 {
			return nil, fmt.Errorf("invalid cluster number: %d", nextCluster)
		}
		if nextCluster > fs.Table.GetMaxCluster() {
			return nil, fmt.Errorf("cluster number out of range: %d", nextCluster)
		}

		currentCluster = nextCluster
	}

	return clusters, nil
}

func (fs *FileSystem) GetClusterChainBytes(startCluster uint32) ([]byte, error) {
	clusterChain, err := fs.getClusterChain(startCluster)
	if err != nil {
		return nil, err
	}

	var bytes []byte
	for _, cluster := range clusterChain {
		clusterBytes := make([]byte, fs.BytesPerCluster)
		fileOffset := fs.BackendOffset + int64(fs.clusterToSector(cluster)*fs.BytesPerSector)
		_, err := fs.Backend.ReadAt(clusterBytes, fileOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to read cluster data: %w", err)
		}

		bytes = append(bytes, clusterBytes...)
	}

	return bytes, nil
}

func (fs *FileSystem) writeClusterChain(clusterStart uint32, clusterBytes []byte) error {
	clusterChain, err := fs.getClusterChain(clusterStart)
	if err != nil {
		return err
	}

	for i, cluster := range clusterChain {
		toWrite := make([]byte, fs.BytesPerCluster)
		copy(toWrite, clusterBytes[int64(i)*fs.BytesPerCluster:int64(i+1)*fs.BytesPerCluster])
		_, err := fs.BackendWriter.WriteAt(toWrite, fs.BackendOffset+int64(fs.clusterToSector(cluster)*fs.BytesPerSector))
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) deleteClusterChain(clusterStart uint32) error {
	clusterChain, err := fs.getClusterChain(clusterStart)
	if err != nil {
		return err
	}

	for _, cluster := range clusterChain {
		err := fs.writeClusterToFats(cluster, 0x0000)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) extendClusterChain(clusterStart uint32, totalBytesNeeded int64) error {
	clusterChain, err := fs.getClusterChain(clusterStart)
	if err != nil {
		return nil
	}

	totalClustersNeeded := max(1, totalBytesNeeded/fs.BytesPerCluster)
	if totalBytesNeeded%fs.BytesPerCluster > 0 {
		totalClustersNeeded++
	}

	nClustersToAllocate := totalClustersNeeded - int64(len(clusterChain))

	if nClustersToAllocate == 0 {
		return nil
	}

	prevCluster := clusterChain[len(clusterChain)-1]
	for cluster := uint32(2); cluster < fs.Table.GetMaxCluster(); cluster++ {
		if fs.Table.GetClusterTarget(cluster) == 0x0000 {
			err := fs.writeClusterToFats(prevCluster, cluster)
			if err != nil {
				return err
			}
			err = fs.writeClusterToFats(cluster, fs.Table.GetEoc())
			if err != nil {
				return err
			}

			nClustersToAllocate--
			prevCluster = cluster

			if nClustersToAllocate == 0 {
				break
			}
		}
	}

	return nil
}
