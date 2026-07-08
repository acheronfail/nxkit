package gui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func TestScrollableTabItemsDoNotSetTallMinimumHeight(t *testing.T) {
	test.NewTempApp(t)

	tallContent := canvas.NewRectangle(color.Black)
	tallContent.SetMinSize(fyne.NewSize(300, 1000))
	otherTallContent := canvas.NewRectangle(color.Black)
	otherTallContent.SetMinSize(fyne.NewSize(300, 1000))

	tabs := container.NewAppTabs(
		newScrollableTabItem("Tall A", tallContent),
		newScrollableTabItem("Tall B", otherTallContent),
	)

	if got, wantLessThan := tabs.MinSize().Height, tallContent.MinSize().Height; got >= wantLessThan {
		t.Fatalf("tab minimum height = %v, want less than tall content height %v", got, wantLessThan)
	}
}
