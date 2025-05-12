package fat32

import (
	"encoding/binary"

	"github.com/acheronfail/nxkit/lib/fat"
	"github.com/acheronfail/nxkit/lib/fat/boot_sector"
	"github.com/acheronfail/nxkit/lib/fat/internal"
)

type FS struct {
	fs *internal.FileSystem
}

// Close implements fs.FileSystem.
func (f *FS) Close() error {
	return f.fs.Close()
}

// GetVolumeId implements fs.FileSystem.
func (f *FS) GetVolumeId() (string, error) {
	return f.fs.GetVolumeId()
}

// Info implements fs.FileSystem.
func (f *FS) Info() map[string]any {
	return f.fs.Info()
}

// Mkdir implements fs.FileSystem.
func (f *FS) Mkdir(path string) error {
	return f.fs.Mkdir(path)
}

// OpenFile implements fs.FileSystem.
func (f *FS) OpenFile(path string, flags int) (fat.File, error) {
	return f.fs.OpenFile(path, flags)
}

// ReadDir implements fs.FileSystem.
func (f *FS) ReadDir(path string) ([]fat.DirectoryEntry, error) {
	return f.fs.ReadDir(path)
}

// Rename implements fs.FileSystem.
func (f *FS) Rename(srcPath string, dstPath string) error {
	return f.fs.Rename(srcPath, dstPath)
}

// Rmdir implements fs.FileSystem.
func (f *FS) Rmdir(path string) error {
	return f.fs.Rmdir(path)
}

// Stat implements fs.FileSystem.
func (f *FS) Stat(path string) (fat.Stat, error) {
	return f.fs.Stat(path)
}

// Unlink implements fs.FileSystem.
func (f *FS) Unlink(path string) error {
	return f.fs.Unlink(path)
}

func NewFromPath(path string) (fat.FileSystem, error) {
	fs, err := internal.NewFileSystemFromPath(
		path,
		boot_sector.Fat32,
		parseFat32Table,
		func(fs *internal.FileSystem) ([]byte, error) {
			return fs.GetClusterChainBytes(2)
		},
	)
	if err != nil {
		return nil, err
	}

	return &FS{fs}, nil
}

const (
	fatEntrySize = uint32(4)
	eoc          = uint32(0xffffff8)
)

type fat32Table struct {
	fatId      uint32
	clusters   []uint32
	maxCluster uint32
}

func parseFat32Table(fatBytes []byte) internal.FatTable {
	maxCluster := uint32(len(fatBytes)) / fatEntrySize
	fatTable := fat32Table{
		fatId:      binary.LittleEndian.Uint32(fatBytes[0:fatEntrySize]),
		clusters:   make([]uint32, maxCluster+1),
		maxCluster: maxCluster,
	}

	for cluster := uint32(2); cluster < maxCluster; cluster++ {
		start := cluster * fatEntrySize
		end := start + fatEntrySize
		val := binary.LittleEndian.Uint32(fatBytes[start:end])
		if val != 0 {
			fatTable.clusters[cluster] = val
		}
	}

	return &fatTable
}

func (t *fat32Table) IsEoc(cluster uint32) bool              { return cluster >= eoc }
func (t *fat32Table) GetEoc() uint32                         { return eoc }
func (t *fat32Table) GetMaxCluster() uint32                  { return t.maxCluster }
func (t *fat32Table) GetClusterTarget(cluster uint32) uint32 { return t.clusters[cluster] }
func (t *fat32Table) WriteClusterTarget(fs *internal.FileSystem, cluster, target uint32) error {
	// set in memory cluster table
	t.clusters[cluster] = target

	// also write back to all FATs
	for i := range uint32(fs.BootSector.BPB_NumFATs) {
		offset := fs.GetFatSectorOffset(i)
		clusterOffset := offset + int64(cluster*fatEntrySize)
		toWrite := make([]byte, 4)
		binary.LittleEndian.PutUint32(toWrite, target)
		_, err := fs.Backend.WriteAt(toWrite, clusterOffset)
		if err != nil {
			return err
		}
	}

	return nil
}
