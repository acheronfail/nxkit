package gui

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
)

func TestHandleToolDropRoutesFileToPaneAtPosition(t *testing.T) {
	test.NewTempApp(t)

	var firstPath, secondPath string
	firstRegion := container.NewStack()
	secondRegion := container.NewStack()
	first := newToolDropTarget(firstRegion, func(path string) {
		firstPath = path
	})
	second := newToolDropTarget(secondRegion, func(path string) {
		secondPath = path
	})
	content := container.NewGridWithColumns(
		2,
		firstRegion,
		secondRegion,
	)
	window := fyne.CurrentApp().NewWindow("drop test")
	t.Cleanup(window.Close)
	window.SetContent(content)
	window.Resize(fyne.NewSize(400, 200))
	fyne.DoAndWait(func() {})

	path := filepath.Join(t.TempDir(), "game.nsp")
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(secondRegion)
	position := origin.Add(fyne.NewPos(secondRegion.Size().Width/2, secondRegion.Size().Height/2))
	handleToolDrop(position, []fyne.URI{storage.NewFileURI(path)}, []*toolDropTarget{first, second})

	if firstPath != "" {
		t.Fatalf(
			"first pane received path %q (first pos=%v size=%v, second pos=%v size=%v, drop=%v)",
			firstPath,
			fyne.CurrentApp().Driver().AbsolutePositionForObject(firstRegion),
			firstRegion.Size(),
			origin,
			secondRegion.Size(),
			position,
		)
	}
	if secondPath != path {
		t.Fatalf("second pane received path %q, want %q", secondPath, path)
	}
}
