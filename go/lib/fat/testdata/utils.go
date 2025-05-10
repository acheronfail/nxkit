package testdata

import (
	"path/filepath"
	"runtime"
)

var (
	Fat12DiskImagePath string
	Fat16DiskImagePath string
	Fat32DiskImagePath string
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not get current file info")
	}

	thisDir := filepath.Dir(filepath.Clean(filename))
	Fat12DiskImagePath = filepath.Join(thisDir, "fat12", "disk.img")
	Fat16DiskImagePath = filepath.Join(thisDir, "fat16", "disk.img")
	Fat32DiskImagePath = filepath.Join(thisDir, "fat32", "disk.img")
}
