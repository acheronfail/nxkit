package testdata

import (
	"path/filepath"
	"runtime"
)

var (
	Fat16DiskImagePath string
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not get current file info")
	}

	thisDir := filepath.Dir(filepath.Clean(filename))
	Fat16DiskImagePath = filepath.Join(thisDir, "fat16", "disk.img")
}
