package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	payloadOutputPaddingX  = float32(12)
	payloadOutputPaddingY  = float32(8)
	payloadOutputMinHeight = float32(84)
)

type payloadOutput struct {
	widget.BaseWidget

	Text string
}

type payloadOutputRenderer struct {
	output     *payloadOutput
	background *canvas.Rectangle
	border     *canvas.Rectangle
	text       *widget.Label
	objects    []fyne.CanvasObject
}

func newPayloadOutput() *payloadOutput {
	output := &payloadOutput{}
	output.ExtendBaseWidget(output)
	return output
}

func (o *payloadOutput) SetText(text string) {
	o.Text = text
	o.Refresh()
}

func (o *payloadOutput) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	border.StrokeWidth = 1
	border.CornerRadius = 4
	text := widget.NewLabelWithStyle(o.Text, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	text.Wrapping = fyne.TextWrapBreak

	renderer := &payloadOutputRenderer{
		output:     o,
		background: background,
		border:     border,
		text:       text,
		objects:    []fyne.CanvasObject{background, text, border},
	}
	renderer.Refresh()
	return renderer
}

func (r *payloadOutputRenderer) Layout(size fyne.Size) {
	r.background.Move(fyne.NewPos(0, 0))
	r.background.Resize(size)
	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(size)

	textWidth := size.Width - payloadOutputPaddingX*2
	if textWidth < 0 {
		textWidth = 0
	}
	textHeight := size.Height - payloadOutputPaddingY*2
	if textHeight < 0 {
		textHeight = 0
	}
	r.text.Move(fyne.NewPos(payloadOutputPaddingX, payloadOutputPaddingY))
	r.text.Resize(fyne.NewSize(textWidth, textHeight))
}

func (r *payloadOutputRenderer) MinSize() fyne.Size {
	textMin := r.text.MinSize()
	width := textMin.Width + payloadOutputPaddingX*2
	if width < 320 {
		width = 320
	}
	height := textMin.Height + payloadOutputPaddingY*2
	if height < payloadOutputMinHeight {
		height = payloadOutputMinHeight
	}
	return fyne.NewSize(width, height)
}

func (r *payloadOutputRenderer) Refresh() {
	r.background.FillColor = theme.Color(theme.ColorNameInputBackground)
	r.background.Refresh()

	r.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	r.border.Refresh()

	r.text.TextStyle = fyne.TextStyle{Monospace: true}
	r.text.SetText(r.output.Text)
	r.text.Refresh()
}

func (r *payloadOutputRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *payloadOutputRenderer) Destroy() {}
