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

func (t nxkitTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInnerPadding:
		return 5
	case theme.SizeNamePadding:
		return 3
	case theme.SizeNameText:
		return 13
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 17
	case theme.SizeNameCaptionText:
		return 10
	case theme.SizeNameLineSpacing:
		return 3
	default:
		return t.Theme.Size(name)
	}
}
