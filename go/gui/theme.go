package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	switchNeonBlue = color.NRGBA{R: 0x00, G: 0xc3, B: 0xe3, A: 0xff}
	switchNeonRed  = color.NRGBA{R: 0xff, G: 0x45, B: 0x54, A: 0xff}
)

type nxkitTheme struct {
	fyne.Theme
}

func newNXKitTheme() fyne.Theme {
	return nxkitTheme{Theme: theme.DefaultTheme()}
}

func (t nxkitTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink, theme.ColorNameSelection:
		return switchNeonBlue
	case theme.ColorNameForegroundOnPrimary:
		return color.Black
	case theme.ColorNameError:
		return switchNeonRed
	case theme.ColorNameForegroundOnError:
		return color.Black
	default:
		return t.Theme.Color(name, variant)
	}
}
