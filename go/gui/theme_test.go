package gui

import (
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func TestNXKitThemeUsesCompactSpacingAndText(t *testing.T) {
	nxkit := newNXKitTheme()
	defaultTheme := theme.DefaultTheme()

	for _, name := range []string{
		string(theme.SizeNameInnerPadding),
		string(theme.SizeNamePadding),
		string(theme.SizeNameText),
		string(theme.SizeNameHeadingText),
		string(theme.SizeNameSubHeadingText),
		string(theme.SizeNameCaptionText),
		string(theme.SizeNameLineSpacing),
	} {
		sizeName := fyne.ThemeSizeName(name)
		if got, defaultSize := nxkit.Size(sizeName), defaultTheme.Size(sizeName); got >= defaultSize {
			t.Fatalf("%s size = %v, want less than default %v", name, got, defaultSize)
		}
	}

	if got, want := nxkit.Size(theme.SizeNameInlineIcon), defaultTheme.Size(theme.SizeNameInlineIcon); got != want {
		t.Fatalf("inline icon size = %v, want default %v", got, want)
	}
}

func TestNXKitListPaletteHasReadableTextInLightAndDarkVariants(t *testing.T) {
	nxkit := newNXKitTheme()

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		foreground := nxkit.Color(theme.ColorNameForeground, variant)
		for _, name := range []nxkitListColorName{
			nxkitListColorBackground,
			nxkitListColorRowEven,
			nxkitListColorRowOdd,
			nxkitListColorDrag,
			nxkitListColorDropTarget,
		} {
			if got := contrastRatio(foreground, nxkitListColorForVariant(name, variant)); got < 4.5 {
				t.Fatalf("contrast for %v on %v in variant %v = %.2f, want at least 4.5", theme.ColorNameForeground, name, variant, got)
			}
		}
	}
}

func TestNXKitPrimaryForegroundHasReadableContrast(t *testing.T) {
	nxkit := newNXKitTheme()

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		if got := contrastRatio(
			nxkit.Color(theme.ColorNameForegroundOnPrimary, variant),
			nxkit.Color(theme.ColorNamePrimary, variant),
		); got < 4.5 {
			t.Fatalf("primary foreground contrast in variant %v = %.2f, want at least 4.5", variant, got)
		}
	}
}

func contrastRatio(foreground color.Color, background color.Color) float64 {
	lighter := relativeLuminance(foreground)
	darker := relativeLuminance(background)
	if darker > lighter {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(c color.Color) float64 {
	r16, g16, b16, _ := c.RGBA()
	r := linearized(float64(r16) / 0xffff)
	g := linearized(float64(g16) / 0xffff)
	b := linearized(float64(b16) / 0xffff)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func linearized(channel float64) float64 {
	if channel <= 0.03928 {
		return channel / 12.92
	}
	return math.Pow((channel+0.055)/1.055, 2.4)
}
