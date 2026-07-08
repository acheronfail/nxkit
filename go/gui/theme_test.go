package gui

import (
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
