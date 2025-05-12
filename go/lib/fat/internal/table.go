package internal

type FatTable struct {
	fatId              uint32
	eoc                uint32
	clusters           []uint32
	maxCluster         uint32
	writeClusterTarget func(fs *FileSystem, cluster, target uint32) error
}

func (t *FatTable) IsEoc(cluster uint32) bool              { return cluster >= t.eoc }
func (t *FatTable) GetEoc() uint32                         { return t.eoc }
func (t *FatTable) GetMaxCluster() uint32                  { return t.maxCluster }
func (t *FatTable) GetClusterTarget(cluster uint32) uint32 { return t.clusters[cluster] }
func (t *FatTable) SetClusterTarget(fs *FileSystem, cluster, target uint32) error {
	t.clusters[cluster] = target
	return t.writeClusterTarget(fs, cluster, target)
}

func NewFatTable(
	fatId,
	eoc,
	maxCluster uint32,
	clusters []uint32,
	writeClusterTarget func(fs *FileSystem, cluster, target uint32) error,
) FatTable {
	return FatTable{
		fatId:              fatId,
		eoc:                eoc,
		clusters:           clusters,
		maxCluster:         maxCluster,
		writeClusterTarget: writeClusterTarget,
	}
}
