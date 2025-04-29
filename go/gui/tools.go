package gui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/tools"
)

var (
	//go:embed markdown/split-as-file.md
	splitAsFileMarkdown string
	//go:embed markdown/split-as-archive.md
	splitAsArchiveMarkdown string
	//go:embed markdown/merge.md
	mergeMarkdown string
)

func ToolsTab() fyne.CanvasObject {
	s := fyne.TextStyle{Bold: true}

	box := container.NewVBox()
	var objects []fyne.CanvasObject

	setLoading := func(loading bool) {
		box.RemoveAll()
		if loading {
			box.Add(widget.NewLabel("Working..."))
			box.Add(widget.NewProgressBarInfinite())
		} else {
			for _, obj := range objects {
				box.Add(obj)
			}
		}
	}

	splitDescription := widget.NewRichTextFromMarkdown(splitAsFileMarkdown)
	splitAsArchive := widget.NewCheck("As archive", func(checked bool) {})
	splitAsArchive.OnChanged = func(checked bool) {
		if checked {
			splitDescription.ParseMarkdown(splitAsArchiveMarkdown)
		} else {
			splitDescription.ParseMarkdown(splitAsFileMarkdown)
		}
		fmt.Println("changing?")
		fmt.Println(splitAsArchiveMarkdown)
	}
	splitMakeCopy := widget.NewCheck("Make copy", func(checked bool) {})
	splitMakeCopy.SetChecked(true)

	mergeMakeCopy := widget.NewCheck("Make copy", func(checked bool) {})
	mergeMakeCopy.SetChecked(true)

	objects = []fyne.CanvasObject{
		widget.NewLabelWithStyle("File Splitter", fyne.TextAlignCenter, s),
		splitDescription,
		container.NewHBox(
			widget.NewButton("Choose file to split", func() {
				dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
					if err != nil {
						dialog.ShowError(err, mainWindow)
						return
					}
					if file == nil {
						return
					}

					go func() {
						setLoading(true)
						path := file.URI().Path()
						output, err := tools.Split(path, splitAsArchive.Checked, !splitMakeCopy.Checked, 0)
						setLoading(false)
						if err != nil {
							dialog.ShowError(err, mainWindow)
						} else {
							nxkitApp.SendNotification(fyne.NewNotification("Split file complete", fmt.Sprintf("Successfully split file: %s", filepath.Base(path))))
							fmt.Println("Output files:", output)
							// TODO: open in explorer/finder/etc
						}
					}()
				}, mainWindow)
			}),
			layout.NewSpacer(),
			splitAsArchive,
			splitMakeCopy,
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("File Merger", fyne.TextAlignCenter, s),
		widget.NewRichTextFromMarkdown(mergeMarkdown),
		container.NewHBox(
			widget.NewButton("Choose file to merge", func() {
				dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
					if err != nil {
						dialog.ShowError(err, mainWindow)
						return
					}
					if file == nil {
						return
					}

					go func() {
						setLoading(true)
						path := file.URI().Path()
						stat, err := os.Stat(path)
						if err != nil {
							dialog.ShowError(err, mainWindow)
							return
						}

						if stat.IsDir() {
							path = filepath.Join(path, "00")
						}

						output, err := tools.Merge(path, !mergeMakeCopy.Checked)
						setLoading(false)
						if err != nil {
							dialog.ShowError(err, mainWindow)
						} else {
							nxkitApp.SendNotification(fyne.NewNotification("Merge file complete", fmt.Sprintf("Successfully merged file: %s", filepath.Base(path))))
							fmt.Println("Output file:", output)
							// TODO: open in explorer/finder/etc
						}
					}()
				}, mainWindow)
			}),
			layout.NewSpacer(),
			mergeMakeCopy,
		),
	}

	for _, obj := range objects {
		box.Add(obj)
	}

	return box
}
