package fat16

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
		boot_sector.Fat16,
		parseFat16Table,
		internal.GetRootDirectoryBytesDedicatedArea,
	)
	if err != nil {
		return nil, err
	}

	return &FS{fs}, nil
}

const (
	fatEntrySize = uint16(2)
	eoc          = uint16(0xfff8)
)

type fat16Table struct {
	fatId      uint16
	clusters   []uint16
	maxCluster uint16
}

func parseFat16Table(fatBytes []byte) internal.FatTable {
	maxCluster := uint16(len(fatBytes)) / fatEntrySize
	fatTable := fat16Table{
		fatId:      binary.LittleEndian.Uint16(fatBytes[0:fatEntrySize]),
		clusters:   make([]uint16, maxCluster+1),
		maxCluster: maxCluster,
	}

	for cluster := uint16(2); cluster < maxCluster; cluster++ {
		start := cluster * fatEntrySize
		end := start + fatEntrySize
		val := binary.LittleEndian.Uint16(fatBytes[start:end])
		if val != 0 {
			fatTable.clusters[cluster] = val
		}
	}

	return &fatTable
}

func (t *fat16Table) IsEoc(cluster uint16) bool              { return cluster >= eoc }
func (t *fat16Table) GetEoc() uint16                         { return eoc }
func (t *fat16Table) GetMaxCluster() uint16                  { return t.maxCluster }
func (t *fat16Table) GetClusterTarget(cluster uint16) uint16 { return t.clusters[cluster] }
func (t *fat16Table) WriteClusterTarget(fs *internal.FileSystem, cluster, target uint16) error {
	// set in memory cluster table
	t.clusters[cluster] = target

	// also write back to all FATs
	for i := range uint32(fs.BootSector.BPB_NumFATs) {
		offset := fs.GetFatSectorOffset(i)
		clusterOffset := offset + int64(cluster*fatEntrySize)
		toWrite := make([]byte, 2)
		binary.LittleEndian.PutUint16(toWrite, target)
		_, err := fs.Backend.WriteAt(toWrite, clusterOffset)
		if err != nil {
			return err
		}
	}

	return nil
}
