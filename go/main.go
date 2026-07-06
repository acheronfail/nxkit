package main

import (
	"os"
	"slices"

	"github.com/acheronfail/nxkit/gui"
)

func main() {
	gui.StartGuiApp(gui.Options{
		RandomPSS: slices.Contains(os.Args, "--random-pss"),
	})
}
