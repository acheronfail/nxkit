package fat12

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
		boot_sector.Fat12,
		parseFat12Table,
		internal.GetRootDirectoryBytesDedicatedArea,
	)
	if err != nil {
		return nil, err
	}

	return &FS{fs}, nil
}

const (
	eoc = uint32(0xff8)
)

type fat12Table struct {
	fatId      uint32
	clusters   []uint32
	maxCluster uint32
}

func parseFat12Table(fatBytes []byte) internal.FatTable {
	// calculate max clusters based on 12-bit entries
	maxCluster := uint32(len(fatBytes)*8) / 12

	fatTable := fat12Table{
		fatId:      uint32(binary.LittleEndian.Uint16(fatBytes[0:2])),
		clusters:   make([]uint32, maxCluster+1),
		maxCluster: maxCluster,
	}

	for cluster := uint32(2); cluster < maxCluster; cluster++ {
		entryOffset := cluster + (cluster / 2)
		if cluster%2 == 0 {
			// Even cluster: lower 12 bits of the 16-bit value
			val := binary.LittleEndian.Uint16(fatBytes[entryOffset:entryOffset+2]) & 0x0FFF
			fatTable.clusters[cluster] = uint32(val)
		} else {
			// Odd cluster: upper 12 bits of the 16-bit value
			val := binary.LittleEndian.Uint16(fatBytes[entryOffset:entryOffset+2]) >> 4
			fatTable.clusters[cluster] = uint32(val)
		}
	}

	return &fatTable
}

func (t *fat12Table) IsEoc(cluster uint32) bool              { return cluster >= eoc }
func (t *fat12Table) GetEoc() uint32                         { return eoc }
func (t *fat12Table) GetMaxCluster() uint32                  { return t.maxCluster }
func (t *fat12Table) GetClusterTarget(cluster uint32) uint32 { return t.clusters[cluster] }
func (t *fat12Table) WriteClusterTarget(fs *internal.FileSystem, cluster, target uint32) error {
	// set in memory cluster table
	t.clusters[cluster] = target

	// also write back to all FATs
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
			_, err := fs.Backend.WriteAt(toWrite[:2], offset+int64(clusterOffset))
			if err != nil {
				return err
			}
		} else {
			// Odd cluster: write upper 12 bits of the target
			val := clusterValue16 & 0x000F
			val |= uint16((target & 0x0FFF) << 4)
			binary.LittleEndian.PutUint16(toWrite[:2], val)
			_, err := fs.Backend.WriteAt(toWrite[:2], offset+int64(clusterOffset))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
