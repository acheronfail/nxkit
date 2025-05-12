package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/acheronfail/nxkit/lib/fat/boot_sector"
)

var (
	TestFatType   int
	diskImagePath string
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not get current file info")
	}

	fatTypeString, ok := os.LookupEnv("FAT")
	if !ok {
		panic("Please provide FAT=XX environment variable")
	}

	n, err := strconv.Atoi(fatTypeString)
	if err != nil {
		panic(fmt.Sprintf("Invalid FAT type: %s", fatTypeString))
	}

	TestFatType = n
	fixturesDirectory := filepath.Dir(filepath.Clean(filename))
	diskImagePath = filepath.Join(fixturesDirectory, fmt.Sprintf("fat%d", n), "disk.img")
}

func GetFatDiskVolumeId() string {
	return fmt.Sprintf("FAT%d-TEST", TestFatType)
}

func GetFatDiskImagePath() string {
	return diskImagePath
}

func GetFatType() boot_sector.FatType {
	return boot_sector.FatType(TestFatType)
}
