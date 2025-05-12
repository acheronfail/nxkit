package fat32

import (
	"encoding/binary"

	"github.com/acheronfail/nxkit/lib/fat"
	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/fat/boot_sector"
	"github.com/acheronfail/nxkit/lib/fat/internal"
)

type FS struct {
	fs *internal.FileSystem
}

// GetType implements fat.FileSystem.
func (f *FS) GetType() boot_sector.FatType {
	return boot_sector.Fat32
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

func Open(backend backend.Storage, offset int64) (fat.FileSystem, error) {
	fs, err := internal.NewFileSystem(
		backend,
		offset,
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

func parseFat32Table(fatBytes []byte) internal.FatTable {
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

	return internal.NewFatTable(fatId, eoc, maxCluster, clusters, func(fs *internal.FileSystem, cluster, target uint32) error {
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
