package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	switchNeonBlue     = color.NRGBA{R: 0x00, G: 0xc3, B: 0xe3, A: 0xff}
	switchNeonRed      = color.NRGBA{R: 0xff, G: 0x45, B: 0x54, A: 0xff}
	switchHoverLight   = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x40}
	switchPressedLight = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x80}
	switchHoverDark    = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x40}
	switchPressedDark  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x80}
)

type nxkitTheme struct {
	fyne.Theme
}

type nxkitListColorName int

const (
	nxkitListColorBackground nxkitListColorName = iota
	nxkitListColorBorder
	nxkitListColorRowEven
	nxkitListColorRowOdd
	nxkitListColorDrag
	nxkitListColorDropTarget
)

func newNXKitTheme() fyne.Theme {
	return nxkitTheme{Theme: theme.DefaultTheme()}
}

func (t nxkitTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink, theme.ColorNameSelection:
		return switchNeonBlue
	case theme.ColorNameHover:
		if variant == theme.VariantLight {
			return switchHoverLight
		}
		return switchHoverDark
	case theme.ColorNamePressed:
		if variant == theme.VariantLight {
			return switchPressedLight
		}
		return switchPressedDark
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

func nxkitCurrentThemeVariant() fyne.ThemeVariant {
	if app := fyne.CurrentApp(); app != nil {
		return app.Settings().ThemeVariant()
	}
	return theme.VariantLight
}

func nxkitThemeColor(name fyne.ThemeColorName) color.Color {
	variant := nxkitCurrentThemeVariant()
	if app := fyne.CurrentApp(); app != nil {
		return app.Settings().Theme().Color(name, variant)
	}
	return newNXKitTheme().Color(name, variant)
}

func nxkitListColor(name nxkitListColorName) color.Color {
	return nxkitListColorForVariant(name, nxkitCurrentThemeVariant())
}

func nxkitListColorForVariant(name nxkitListColorName, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantLight {
		switch name {
		case nxkitListColorBackground:
			return color.NRGBA{R: 0xf8, G: 0xfa, B: 0xfc, A: 0xff}
		case nxkitListColorBorder:
			return color.NRGBA{R: 0xcb, G: 0xd5, B: 0xe1, A: 0xff}
		case nxkitListColorRowEven:
			return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		case nxkitListColorRowOdd:
			return color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff}
		case nxkitListColorDrag:
			return color.NRGBA{R: 0xdb, G: 0xea, B: 0xfe, A: 0xff}
		case nxkitListColorDropTarget:
			return color.NRGBA{R: 0xcc, G: 0xfb, B: 0xf1, A: 0xff}
		}
	}

	switch name {
	case nxkitListColorBackground:
		return color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xff}
	case nxkitListColorBorder:
		return color.NRGBA{R: 0x3e, G: 0x42, B: 0x4a, A: 0xff}
	case nxkitListColorRowEven:
		return color.NRGBA{R: 0x10, G: 0x11, B: 0x15, A: 0xff}
	case nxkitListColorRowOdd:
		return color.NRGBA{R: 0x18, G: 0x19, B: 0x1e, A: 0xff}
	case nxkitListColorDrag:
		return color.NRGBA{R: 0x24, G: 0x2d, B: 0x3a, A: 0xff}
	case nxkitListColorDropTarget:
		return color.NRGBA{R: 0x1f, G: 0x36, B: 0x2d, A: 0xff}
	default:
		return color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xff}
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
