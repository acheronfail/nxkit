package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const toolPanelCornerRadius = 8

type toolPanelBorder struct {
	widget.BaseWidget
}

func newToolPanelBorder() *toolPanelBorder {
	border := &toolPanelBorder{}
	border.ExtendBaseWidget(border)
	return border
}

func (b *toolPanelBorder) CreateRenderer() fyne.WidgetRenderer {
	border := canvas.NewRectangle(color.Transparent)
	border.CornerRadius = toolPanelCornerRadius
	border.StrokeWidth = 1
	renderer := &toolPanelBorderRenderer{panel: b, border: border}
	renderer.Refresh()
	return renderer
}

type toolPanelBorderRenderer struct {
	panel  *toolPanelBorder
	border *canvas.Rectangle
}

func (r *toolPanelBorderRenderer) Layout(size fyne.Size) {
	r.border.Resize(size)
}

func (*toolPanelBorderRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 0)
}

func (r *toolPanelBorderRenderer) Refresh() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	r.border.StrokeColor = r.panel.Theme().Color(theme.ColorNameInputBorder, variant)
	r.border.Refresh()
}

func (r *toolPanelBorderRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.border}
}

func (*toolPanelBorderRenderer) Destroy() {}
