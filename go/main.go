package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/acheronfail/nxkit/gui"
	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/fat/fat_auto"
	"github.com/acheronfail/nxkit/lib/inject"
	"github.com/acheronfail/nxkit/lib/nacp"
	"github.com/acheronfail/nxkit/lib/nand"
	"github.com/acheronfail/nxkit/lib/npdm"
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
//     - [ ] nca program
//     - [ ] nca control
//     - [ ] nca htmldoc
//     - [ ] nca legalinfo
//     - [ ] nca meta
//     - [ ] nsp
// - [ ] have a way to ship assets (*.nso, etc)

func createNsp() {
	// validate and patch npdm as needed
	titleId, err := npdm.Process("staging", "staging.bkp", 0, false)
	if err != nil {
		panic(err)
	}

	// validate and patch nacp as needed
	launcherNacp := nacp.NewNacp(nil)
	launcherNacp.SetTitle("Test Title")
	launcherNacp.SetAuthor("Test Author")
	err = nacp.Process(launcherNacp, titleId)
	if err != nil {
		panic(err)
	}

	// TODO: nca_create_program
	// TODO: nca_create_control
	// TODO: nca_create_manual_htmldoc
	// TODO: nca_create_manual_legalinfo
	// TODO: nca_create_meta
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
		createNsp()
	} else {
		gui.StartGuiApp()
	}
}
