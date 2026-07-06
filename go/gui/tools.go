package gui

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	splitDescription := widget.NewRichTextFromMarkdown(splitAsFileMarkdown)
	splitAsArchive := widget.NewCheck("As archive", func(checked bool) {})
	splitAsArchive.OnChanged = func(checked bool) {
		if checked {
			splitDescription.ParseMarkdown(splitAsArchiveMarkdown)
		} else {
			splitDescription.ParseMarkdown(splitAsFileMarkdown)
		}
	}
	splitMakeCopy := widget.NewCheck("Make copy", func(checked bool) {})
	splitMakeCopy.SetChecked(true)

	mergeMakeCopy := widget.NewCheck("Make copy", func(checked bool) {})
	mergeMakeCopy.SetChecked(true)

	splitProgressLabel := widget.NewLabel("")
	splitProgress := widget.NewProgressBar()
	splitProgressContainer := container.NewVBox(splitProgressLabel, splitProgress)
	splitProgressContainer.Hide()
	var splitCancel context.CancelFunc
	splitCancelButton := widget.NewButton("Cancel", func() {
		if splitCancel != nil {
			splitCancel()
		}
	})
	splitCancelButton.Disable()

	mergeProgressLabel := widget.NewLabel("")
	mergeProgress := widget.NewProgressBar()
	mergeProgressContainer := container.NewVBox(mergeProgressLabel, mergeProgress)
	mergeProgressContainer.Hide()
	var mergeCancel context.CancelFunc
	mergeCancelButton := widget.NewButton("Cancel", func() {
		if mergeCancel != nil {
			mergeCancel()
		}
	})
	mergeCancelButton.Disable()

	setProgress := func(label *widget.Label, bar *widget.ProgressBar, done, total int64) {
		fyne.Do(func() {
			if total <= 0 {
				bar.SetValue(1)
				label.SetText("Complete")
				return
			}
			value := float64(done) / float64(total)
			bar.SetValue(value)
			label.SetText(fmt.Sprintf("%s / %s", formatBytes(done), formatBytes(total)))
		})
	}

	var splitButton *widget.Button
	splitButton = widget.NewButton("Choose file to split", func() {
		chooseNativeFile("Choose file to split", nil, func(path string) {
			asArchive := splitAsArchive.Checked
			inPlace := !splitMakeCopy.Checked
			splitButton.Disable()
			splitAsArchive.Disable()
			splitMakeCopy.Disable()
			splitProgress.SetValue(0)
			splitProgressLabel.SetText("Starting split...")
			splitProgressContainer.Show()
			ctx, cancel := context.WithCancel(context.Background())
			splitCancel = cancel
			splitCancelButton.Enable()
			go func() {
				output, err := tools.SplitWithProgressContext(ctx, path, asArchive, inPlace, 0, func(done, total int64) {
					setProgress(splitProgressLabel, splitProgress, done, total)
				})
				fyne.Do(func() {
					splitButton.Enable()
					splitAsArchive.Enable()
					splitMakeCopy.Enable()
					splitCancel = nil
					splitCancelButton.Disable()
				})
				if err != nil {
					fyne.Do(func() {
						if errors.Is(err, context.Canceled) {
							splitProgressLabel.SetText("Split canceled")
							return
						}
						splitProgressLabel.SetText("Split failed")
						showError(err)
					})
					return
				}
				fyne.Do(func() {
					splitProgressLabel.SetText("Split complete")
					splitProgress.SetValue(1)
					showError(revealPath(output))
					sendAppNotification("Split file complete", fmt.Sprintf("Successfully split file: %s", filepath.Base(path)))
				})
			}()
		})
	})

	var mergeButton *widget.Button
	mergeButton = widget.NewButton("Choose file to merge", func() {
		chooseNativeFile("Choose file to merge", nil, func(inputPath string) {
			inPlace := !mergeMakeCopy.Checked
			mergeButton.Disable()
			mergeMakeCopy.Disable()
			mergeProgress.SetValue(0)
			mergeProgressLabel.SetText("Starting merge...")
			mergeProgressContainer.Show()
			ctx, cancel := context.WithCancel(context.Background())
			mergeCancel = cancel
			mergeCancelButton.Enable()
			go func() {
				path := inputPath
				stat, err := os.Stat(path)
				if err == nil && stat.IsDir() {
					path = filepath.Join(path, "00")
				}
				var output string
				if err == nil {
					output, err = tools.MergeWithProgressContext(ctx, path, inPlace, func(done, total int64) {
						setProgress(mergeProgressLabel, mergeProgress, done, total)
					})
				}
				fyne.Do(func() {
					mergeButton.Enable()
					mergeMakeCopy.Enable()
					mergeCancel = nil
					mergeCancelButton.Disable()
				})
				if err != nil {
					fyne.Do(func() {
						if errors.Is(err, context.Canceled) {
							mergeProgressLabel.SetText("Merge canceled")
							return
						}
						mergeProgressLabel.SetText("Merge failed")
						showError(err)
					})
					return
				}
				fyne.Do(func() {
					mergeProgressLabel.SetText("Merge complete")
					mergeProgress.SetValue(1)
					showError(revealPath(output))
					sendAppNotification("Merge file complete", fmt.Sprintf("Successfully merged file: %s", filepath.Base(inputPath)))
				})
			}()
		})
	})

	objects = []fyne.CanvasObject{
		widget.NewLabelWithStyle("File Splitter", fyne.TextAlignCenter, s),
		splitDescription,
		container.NewHBox(
			splitButton,
			splitCancelButton,
			layout.NewSpacer(),
			splitAsArchive,
			splitMakeCopy,
		),
		splitProgressContainer,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("File Merger", fyne.TextAlignCenter, s),
		widget.NewRichTextFromMarkdown(mergeMarkdown),
		container.NewHBox(
			mergeButton,
			mergeCancelButton,
			layout.NewSpacer(),
			mergeMakeCopy,
		),
		mergeProgressContainer,
	}

	for _, obj := range objects {
		box.Add(obj)
	}

	return box
}
