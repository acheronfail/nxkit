package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/inject"
	"github.com/acheronfail/nxkit/lib/nand"
	"github.com/acheronfail/nxkit/lib/xtsn"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/google/gousb"
	"github.com/jpillora/sizestr"
)

// If I am do to this, I would first need:
// - [x] port XTSN to golang
// - [x] support reading split dumps
// - [x] support injecting payloads (port web injector)
// - [ ] port hacbrewpack to golang (or compile it and then spawn it?)
// - [ ] have a way to ship assets (*.nso, etc)

// TODO: don't leave devices open? open in inject?
func injectPayload(payloadPath string) {
	// Read payload from file
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		panic(err)
	}

	// Initialize USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Look for RCM devices
	devices, err := inject.FindRCMDevices(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		for _, d := range devices {
			d.Close()
		}
	}()

	if len(devices) == 0 {
		panic("No Nintendo Switch RCM devices found.")
	}

	fmt.Printf("Found %d RCM device(s)\n", len(devices))

	// Send payload to the first device found
	device := devices[0]
	err = inject.InjectPayload(device, payload)
	if err != nil {
		panic(err)
	}

	fmt.Println("Injection completed successfully!")
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
	openNand("../.data/rawnand.bin.00")
	injectPayload("/Users/cosmotherly/.switch/payloads/hekate_ctcaer_6.2.2.bin")

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
