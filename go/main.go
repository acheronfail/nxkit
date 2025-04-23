package main

import (
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/nand"
	"github.com/acheronfail/nxkit/lib/xtsn"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/jpillora/sizestr"
)

// If I am do to this, I would first need:
// - [x] port XTSN to golang
// - [ ] support reading split dumps
// - [ ] port hacbrewpack to golang (or compile it and then spawn it?)
// - [ ] have a way to ship assets (*.nso, etc)

func openNand(path string) {
	// open dump
	dumpBackend, err := nand.NewCombinedDumpBackend(path)
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
	for i, part := range gptTable.Partitions {
		if part.Name == "USER" {
			userPartition = gptTable.Partitions[i]
			break
		}
	}

	if userPartition == nil {
		panic("USER partition not found")
	}

	fmt.Printf("Found USER partition, start=%d end=%d size=%s\n", userPartition.Start, userPartition.End, sizestr.ToString(int64(userPartition.Size)))

	// setup nand backend
	fsSectorSize := int64(512)
	cryptoBlockSize := uint64(16)
	cryptoSectorSize := uint64(16384)
	nandBackend := nand.NewNandBackend(dumpBackend, userPartition.Start*uint64(fsSectorSize), (userPartition.End+1)*uint64(fsSectorSize), cryptoBlockSize, uint64(fsSectorSize))
	crypto, err := xtsn.NewXtsnCipher(make([]byte, 16), make([]byte, 16), cryptoSectorSize)
	if err != nil {
		panic(err)
	}

	nandBackend.SetCrypto(crypto)

	// read fat32 filesystem
	fs, err := fat32.Read(nandBackend, int64(userPartition.Size), int64(userPartition.Start)*fsSectorSize, fsSectorSize)
	if err != nil {
		panic(err)
	}
	entries, err := fs.ReadDir("/")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		fmt.Printf("Entry: %s\n", entry.Name())
	}
}

func main() {
	openNand("../.data/rawnand.bin")

	myApp := app.New()
	w := myApp.NewWindow("Two Way")

	str := binding.NewString()
	str.Set("Hi!")

	w.SetContent(container.NewVBox(
		widget.NewLabelWithData(str),
		widget.NewEntryWithData(str),
	))

	w.ShowAndRun()
}
