package fat16

import (
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
	return boot_sector.Fat16
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
		boot_sector.Fat16,
		internal.NewFat16Table,
		internal.GetRootDirectoryBytesDedicatedArea,
	)
	if err != nil {
		return nil, err
	}

	return &FS{fs}, nil
}
