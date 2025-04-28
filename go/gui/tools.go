package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func ToolsTab() fyne.CanvasObject {
	s := fyne.TextStyle{Bold: true}

	return container.NewVBox(
		widget.NewLabelWithStyle("File Splitter", fyne.TextAlignCenter, s),
		widget.NewRichTextFromMarkdown("TODO some description here"),
		container.NewHBox(
			widget.NewButton("Choose file to split", func() {
				dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
					if err != nil {
						return
					}
					if file == nil {
						return
					}

					fmt.Println(file.URI().Path())
				}, mainWindow)
			}),
			layout.NewSpacer(),
			widget.NewCheck("As archive", func(checked bool) {}),
			widget.NewCheck("Make copy", func(checked bool) {}),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("File Merger", fyne.TextAlignCenter, s),
		widget.NewRichTextFromMarkdown("TODO some description here"),
		container.NewHBox(
			widget.NewButton("Choose file to merge", func() {
				dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
					if err != nil {
						return
					}
					if file == nil {
						return
					}

					fmt.Println(file.URI().Path())
				}, mainWindow)
			}),
			layout.NewSpacer(),
			widget.NewCheck("Make copy", func(checked bool) {}),
		),
	)
}
