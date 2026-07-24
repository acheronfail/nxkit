package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
)

type toolDropTarget struct {
	region fyne.CanvasObject
	onDrop func(string)
}

func newToolDropTarget(region fyne.CanvasObject, onDrop func(string)) *toolDropTarget {
	return &toolDropTarget{region: region, onDrop: onDrop}
}

func (t *toolDropTarget) contains(position fyne.Position) bool {
	app := fyne.CurrentApp()
	if app == nil || t.region == nil {
		return false
	}
	origin := app.Driver().AbsolutePositionForObject(t.region)
	size := t.region.Size()
	return position.X >= origin.X &&
		position.X <= origin.X+size.Width &&
		position.Y >= origin.Y &&
		position.Y <= origin.Y+size.Height
}

func (t *toolDropTarget) drop(path string) {
	if t.onDrop != nil {
		t.onDrop(path)
	}
}

func handleToolDrop(position fyne.Position, uris []fyne.URI, targets []*toolDropTarget) {
	var target *toolDropTarget
	for _, candidate := range targets {
		if candidate != nil && candidate.contains(position) {
			target = candidate
			break
		}
	}
	if target == nil {
		return
	}
	if len(uris) != 1 || uris[0] == nil {
		showError(fmt.Errorf("drop exactly one file onto a tool"))
		return
	}
	uri := uris[0]
	if uri.Scheme() != "file" {
		showError(fmt.Errorf("cannot use non-file URI with this tool: %s", uri.String()))
		return
	}
	target.drop(uri.Path())
}
