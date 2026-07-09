package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type actionListRow struct {
	widget.BaseWidget

	title            string
	index            int
	icon             fyne.Resource
	actionText       string
	actionDisabled   bool
	actionImportance widget.Importance
	onAction         func()

	actionPos  fyne.Position
	actionSize fyne.Size
}

type actionListRowRenderer struct {
	row          *actionListRow
	background   *canvas.Rectangle
	icon         *widget.Icon
	titleText    *canvas.Text
	actionButton *canvas.Rectangle
	actionText   *canvas.Text
	objects      []fyne.CanvasObject
}

func newActionListRow(
	title string,
	index int,
	icon fyne.Resource,
	actionText string,
	actionDisabled bool,
	actionImportance widget.Importance,
	onAction func(),
) *actionListRow {
	row := &actionListRow{
		title:            title,
		index:            index,
		icon:             icon,
		actionText:       actionText,
		actionDisabled:   actionDisabled,
		actionImportance: actionImportance,
		onAction:         onAction,
	}
	row.ExtendBaseWidget(row)
	return row
}

func (r *actionListRow) Tapped(event *fyne.PointEvent) {
	if r.actionDisabled || r.onAction == nil {
		return
	}
	if event.Position.X < r.actionPos.X ||
		event.Position.X > r.actionPos.X+r.actionSize.Width ||
		event.Position.Y < r.actionPos.Y ||
		event.Position.Y > r.actionPos.Y+r.actionSize.Height {
		return
	}
	r.onAction()
}

func (r *actionListRow) Cursor() desktop.Cursor {
	if r.actionDisabled {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

func (r *actionListRow) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(nxkitListColor(nxkitListColorRowEven))
	icon := widget.NewIcon(r.icon)
	titleText := canvas.NewText("", nxkitThemeColor(theme.ColorNameForeground))
	titleText.TextStyle = fyne.TextStyle{Monospace: true}
	titleText.TextSize = theme.Size(theme.SizeNameText)
	actionButton := canvas.NewRectangle(theme.Color(theme.ColorNameButton))
	actionButton.CornerRadius = 4
	actionButton.StrokeColor = nxkitListColor(nxkitListColorBorder)
	actionButton.StrokeWidth = 1
	actionText := canvas.NewText(r.actionText, nxkitThemeColor(theme.ColorNameForeground))
	actionText.Alignment = fyne.TextAlignCenter
	actionText.TextStyle = fyne.TextStyle{Bold: true}
	actionText.TextSize = theme.Size(theme.SizeNameCaptionText)
	renderer := &actionListRowRenderer{
		row:          r,
		background:   background,
		icon:         icon,
		titleText:    titleText,
		actionButton: actionButton,
		actionText:   actionText,
		objects:      []fyne.CanvasObject{background, icon, titleText, actionButton, actionText},
	}
	renderer.Refresh()
	return renderer
}

func (r *actionListRowRenderer) Layout(size fyne.Size) {
	r.background.Move(fyne.NewPos(0, 0))
	r.background.Resize(size)

	iconSize := float32(16)
	iconX := float32(6)
	iconY := (size.Height - iconSize) / 2
	r.icon.Move(fyne.NewPos(iconX, iconY))
	r.icon.Resize(fyne.NewSize(iconSize, iconSize))

	actionPaddingX := float32(14)
	actionWidth := fyne.MeasureText(r.row.actionText, r.actionText.TextSize, r.actionText.TextStyle).Width + actionPaddingX
	if actionWidth < 52 {
		actionWidth = 52
	}
	actionHeight := float32(24)
	actionX := size.Width - actionWidth - 8
	if actionX < 0 {
		actionX = 0
	}
	actionY := (size.Height - actionHeight) / 2
	if actionY < 0 {
		actionY = 0
	}
	r.row.actionPos = fyne.NewPos(actionX, actionY)
	r.row.actionSize = fyne.NewSize(actionWidth, actionHeight)
	r.actionButton.Move(fyne.NewPos(actionX, actionY))
	r.actionButton.Resize(fyne.NewSize(actionWidth, actionHeight))

	r.actionText.Move(fyne.NewPos(actionX, actionY))
	r.actionText.Resize(fyne.NewSize(actionWidth, actionHeight))

	titleX := iconX + iconSize + 8
	titleWidth := actionX - titleX - 10
	if titleWidth < 0 {
		titleWidth = 0
	}
	displayTitle := truncateTextToWidth(r.row.title, titleWidth, r.titleText.TextSize, r.titleText.TextStyle)
	if r.titleText.Text != displayTitle {
		r.titleText.Text = displayTitle
		r.titleText.Refresh()
	}
	titleHeight := r.titleText.MinSize().Height
	titleY := (size.Height - titleHeight) / 2
	if titleY < 0 {
		titleY = 0
		titleHeight = size.Height
	}
	r.titleText.Move(fyne.NewPos(titleX, titleY))
	r.titleText.Resize(fyne.NewSize(titleWidth, titleHeight))
}

func (r *actionListRowRenderer) MinSize() fyne.Size {
	return fyne.NewSize(320, 32)
}

func (r *actionListRowRenderer) Refresh() {
	if r.row.index%2 == 0 {
		r.background.FillColor = nxkitListColor(nxkitListColorRowEven)
	} else {
		r.background.FillColor = nxkitListColor(nxkitListColorRowOdd)
	}
	r.background.Refresh()

	r.icon.SetResource(r.row.icon)
	r.titleText.Color = nxkitThemeColor(theme.ColorNameForeground)
	r.titleText.TextSize = theme.Size(theme.SizeNameText)
	r.titleText.Refresh()

	r.actionButton.StrokeColor = nxkitListColor(nxkitListColorBorder)
	if r.row.actionDisabled {
		r.actionButton.FillColor = nxkitThemeColor(theme.ColorNameDisabledButton)
		r.actionText.Color = nxkitThemeColor(theme.ColorNameDisabled)
	} else {
		switch r.row.actionImportance {
		case widget.HighImportance:
			r.actionButton.FillColor = nxkitThemeColor(theme.ColorNamePrimary)
			r.actionText.Color = nxkitThemeColor(theme.ColorNameForegroundOnPrimary)
		case widget.DangerImportance:
			r.actionButton.FillColor = nxkitThemeColor(theme.ColorNameError)
			r.actionText.Color = nxkitThemeColor(theme.ColorNameForegroundOnError)
		default:
			r.actionButton.FillColor = nxkitThemeColor(theme.ColorNameButton)
			r.actionText.Color = nxkitThemeColor(theme.ColorNameForeground)
		}
	}
	r.actionButton.Refresh()
	r.actionText.Text = r.row.actionText
	r.actionText.TextSize = theme.Size(theme.SizeNameCaptionText)
	r.actionText.Refresh()
}

func (r *actionListRowRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *actionListRowRenderer) Destroy() {}
