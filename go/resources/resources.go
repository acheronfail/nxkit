package resources

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed nxkit-icon-subtle-256.png
var nxkitIcon []byte

var NXKitIcon = fyne.NewStaticResource("NXKit.png", nxkitIcon)
