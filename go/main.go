package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/acheronfail/nxkit/gui"
	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/fat/fat_auto"
	"github.com/acheronfail/nxkit/lib/hacbrewpack"
	"github.com/acheronfail/nxkit/lib/inject"
	"github.com/acheronfail/nxkit/lib/nand"
	"github.com/acheronfail/nxkit/lib/xtsn"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/jpillora/sizestr"
)

// If I am do to this, I would first need:
// - [x] port XTSN to golang
// - [x] support reading split dumps
// - [x] support injecting payloads (port web injector)
// - [x] add FAT12/16 support (own filesystem driver?)
// - [-] port hacbrewpack to golang (or compile it and then spawn it?)
//     - [x] nacp
//     - [x] npdm
//     - [x] nca reading
//     - [x] pfs0 reading
//     - [ ] pfs0 creation
//     - [x] romfs reading
//     - [ ] romfs creation
//     - [ ] cnmt reading
//     - [ ] cnmt creation
//     - [ ] nca id calculation (will be used to compute names for the following nca's)
//     - [ ] create nca program
//         - [ ] section 0 pfs0  with main + main.npdm
//         - [ ] section 1 romfs with /nextArgv + /nextNroPath
//         - [ ] section 2 pfs0  with NintendoLogo.png + StartupMovie.gif
//     - [ ] create nca control
//         - [ ] section 0 romfs with /control.nacp + /icon_AmericanEnglish.dat
//     - [ ] create nca htmldoc
//     - [ ] create nca legalinfo
//         - [ ] section 0 romfs with ???
//     - [ ] create nca meta
//         - [ ] section 0 pfs0 with Application_${TITLE_ID}.cnmt
//     - [ ] package nsp (a pfs0 filesystem with nca's inside)
// - [ ] have a way to ship assets (*.nso, etc)

func createNsp(randomPSS bool) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		panic(err)
	}

	outPath := filepath.Join(mustGetwd(), "0162696bc58e0000_title=1_publisher=2_nroPath=3.nsp")
	if err := hacbrewpack.BuildForwarderNSP(hacbrewpack.Options{
		RepoRoot:  repoRoot,
		OutPath:   outPath,
		IconPath:  filepath.Join(repoRoot, ".data", "test_magenta_test.canvas.jpg"),
		RandomPSS: randomPSS,
	}); err != nil {
		panic(err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created %s\nMD5: %x\n", outPath, md5.Sum(data))
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "vendor", "hacbrewpack", "hacbrewpack")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("failed to locate repository root from %s", mustGetwd())
		}
		dir = parent
	}
}

func openNand(path string) {
	// open dump
	dumpBackend, err := nand.NewDumpBackend(path, true)
	if err != nil {
		panic(err)
	}
	defer dumpBackend.Close()

	// read gpt partition table
	gptTable, err := gpt.Read(dumpBackend, 512, 512)
	if err != nil {
		panic(err)
	}

	var userPartition *gpt.Partition
	var prodInfoFPartition *gpt.Partition
	for i, part := range gptTable.Partitions {
		if part.Name == "USER" {
			userPartition = gptTable.Partitions[i]
		}
		if part.Name == "PRODINFOF" {
			prodInfoFPartition = gptTable.Partitions[i]
		}
	}

	if prodInfoFPartition == nil {
		panic("PRODINFOF partition not found")
	}
	if userPartition == nil {
		panic("USER partition not found")
	}

	listPartition(dumpBackend, userPartition)
	listPartition(dumpBackend, prodInfoFPartition)
}

func listPartition(dumpBackend backend.Storage, partition *gpt.Partition) {
	fmt.Printf("Found %s partition, start=%d end=%d size=%s\n", partition.Name, partition.Start, partition.End, sizestr.ToString(int64(partition.Size)))

	// setup nand backend
	fsSectorSize := int64(512)
	cryptoBlockSize := uint64(16)
	cryptoSectorSize := uint64(16384)
	partBackend := nand.NewNxPartBackend(dumpBackend, partition.Start*uint64(fsSectorSize), (partition.End+1)*uint64(fsSectorSize), cryptoBlockSize, uint64(fsSectorSize))

	crypto, err := xtsn.NewXtsnCipher(make([]byte, 16), make([]byte, 16), cryptoSectorSize)
	if err != nil {
		panic(err)
	}
	partBackend.SetCrypto(crypto)

	fs, err := fat_auto.Open(partBackend, int64(partition.Start)*fsSectorSize)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Opened partition %s, detected fs type: %d\n", partition.Name, fs.GetType())

	entries, err := fs.ReadDir("/")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		fmt.Printf("Entry: %s\n", entry.LongName())
	}
}

func main() {
	if slices.Contains(os.Args, "--nand") {
		openNand("../.data/rawnand.bin")
	} else if slices.Contains(os.Args, "--inject") {
		inject.Inject("/home/acheronfail/.switch/payloads/hekate_ctcaer_6.2.2.bin")
	} else if slices.Contains(os.Args, "--nsp") {
		createNsp(slices.Contains(os.Args, "--random-pss"))
	} else {
		gui.StartGuiApp()
	}
}
