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
	//go:embed markdown/compress-nsz.md
	compressNSZMarkdown string
	//go:embed markdown/decompress-nsz.md
	decompressNSZMarkdown string
)

const (
	toolsDividerThickness       float32 = 6
	toolsSingleColumnBreakpoint float32 = 900
)

func ToolsTab() fyne.CanvasObject {
	s := fyne.TextStyle{Bold: true}

	splitDescription := widget.NewRichTextFromMarkdown(splitAsFileMarkdown)
	splitDescription.Wrapping = fyne.TextWrapWord
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
	mergeDescription := widget.NewRichTextFromMarkdown(mergeMarkdown)
	mergeDescription.Wrapping = fyne.TextWrapWord
	compressDescription := widget.NewRichTextFromMarkdown(compressNSZMarkdown)
	compressDescription.Wrapping = fyne.TextWrapWord
	decompressDescription := widget.NewRichTextFromMarkdown(decompressNSZMarkdown)
	decompressDescription.Wrapping = fyne.TextWrapWord

	compressProgressLabel := widget.NewLabel("")
	compressProgress := widget.NewProgressBar()
	compressProgressContainer := container.NewVBox(compressProgressLabel, compressProgress)
	compressProgressContainer.Hide()
	var compressCancel context.CancelFunc
	compressCancelButton := widget.NewButton("Cancel", func() {
		if compressCancel != nil {
			compressCancel()
		}
	})
	compressCancelButton.Disable()

	decompressProgressLabel := widget.NewLabel("")
	decompressProgress := widget.NewProgressBar()
	decompressProgressContainer := container.NewVBox(decompressProgressLabel, decompressProgress)
	decompressProgressContainer.Hide()
	var decompressCancel context.CancelFunc
	decompressCancelButton := widget.NewButton("Cancel", func() {
		if decompressCancel != nil {
			decompressCancel()
		}
	})
	decompressCancelButton.Disable()

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

	var splitButton *widget.Button
	var mergeButton *widget.Button
	updateSplitButtonImportance := func() {
		if splitButton == nil {
			return
		}
		if splitMakeCopy.Checked {
			splitButton.Importance = widget.HighImportance
		} else {
			splitButton.Importance = widget.DangerImportance
		}
		splitButton.Refresh()
	}
	updateMergeButtonImportance := func() {
		if mergeButton == nil {
			return
		}
		if mergeMakeCopy.Checked {
			mergeButton.Importance = widget.HighImportance
		} else {
			mergeButton.Importance = widget.DangerImportance
		}
		mergeButton.Refresh()
	}
	splitMakeCopy.OnChanged = func(checked bool) {
		updateSplitButtonImportance()
	}
	mergeMakeCopy.OnChanged = func(checked bool) {
		updateMergeButtonImportance()
	}

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
	updateSplitButtonImportance()

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
	updateMergeButtonImportance()

	var compressButton *widget.Button
	compressButton = widget.NewButton("Choose NSP to compress", func() {
		chooseNativeFile("Choose NSP to compress", []nativeFileFilter{
			extensionFilter("Nintendo Submission Package", ".nsp"),
		}, func(path string) {
			if state.Keys == nil {
				showError(fmt.Errorf("prod.keys are required; configure them in Settings"))
				return
			}

			compressButton.Disable()
			compressProgress.SetValue(0)
			compressProgressLabel.SetText("Starting compression...")
			compressProgressContainer.Show()
			ctx, cancel := context.WithCancel(context.Background())
			compressCancel = cancel
			compressCancelButton.Enable()
			go func() {
				output, err := tools.CompressNSZWithProgressContext(ctx, path, *state.Keys, func(done, total int64) {
					setProgress(compressProgressLabel, compressProgress, done, total)
				})
				fyne.Do(func() {
					compressButton.Enable()
					compressCancel = nil
					compressCancelButton.Disable()
				})
				if err != nil {
					fyne.Do(func() {
						if errors.Is(err, context.Canceled) {
							compressProgressLabel.SetText("Compression canceled")
							return
						}
						compressProgressLabel.SetText("Compression failed")
						showError(err)
					})
					return
				}
				fyne.Do(func() {
					compressProgressLabel.SetText("Compression complete")
					compressProgress.SetValue(1)
					showError(revealPath(output))
					sendAppNotification("NSZ compression complete", fmt.Sprintf("Successfully compressed file: %s", filepath.Base(path)))
				})
			}()
		})
	})
	compressButton.Importance = widget.HighImportance

	var decompressButton *widget.Button
	decompressButton = widget.NewButton("Choose NSZ to decompress", func() {
		chooseNativeFile("Choose NSZ to decompress", []nativeFileFilter{
			extensionFilter("Nintendo Submission Package Zipped", ".nsz"),
		}, func(path string) {
			decompressButton.Disable()
			decompressProgress.SetValue(0)
			decompressProgressLabel.SetText("Starting decompression...")
			decompressProgressContainer.Show()
			ctx, cancel := context.WithCancel(context.Background())
			decompressCancel = cancel
			decompressCancelButton.Enable()
			go func() {
				output, err := tools.DecompressNSZWithProgressContext(ctx, path, func(done, total int64) {
					setProgress(decompressProgressLabel, decompressProgress, done, total)
				})
				fyne.Do(func() {
					decompressButton.Enable()
					decompressCancel = nil
					decompressCancelButton.Disable()
				})
				if err != nil {
					fyne.Do(func() {
						if errors.Is(err, context.Canceled) {
							decompressProgressLabel.SetText("Decompression canceled")
							return
						}
						decompressProgressLabel.SetText("Decompression failed")
						showError(err)
					})
					return
				}
				fyne.Do(func() {
					decompressProgressLabel.SetText("Decompression complete")
					decompressProgress.SetValue(1)
					showError(revealPath(output))
					sendAppNotification("NSZ decompression complete", fmt.Sprintf("Successfully decompressed file: %s", filepath.Base(path)))
				})
			}()
		})
	})
	decompressButton.Importance = widget.HighImportance

	splitPanel := container.NewVBox(
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
	)
	mergePanel := container.NewVBox(
		widget.NewLabelWithStyle("File Merger", fyne.TextAlignCenter, s),
		mergeDescription,
		container.NewHBox(
			mergeButton,
			mergeCancelButton,
			layout.NewSpacer(),
			mergeMakeCopy,
		),
		mergeProgressContainer,
	)
	compressPanel := container.NewVBox(
		widget.NewLabelWithStyle("NSZ Compressor", fyne.TextAlignCenter, s),
		compressDescription,
		container.NewHBox(
			compressButton,
			compressCancelButton,
			layout.NewSpacer(),
		),
		compressProgressContainer,
	)
	decompressPanel := container.NewVBox(
		widget.NewLabelWithStyle("NSZ Decompressor", fyne.TextAlignCenter, s),
		decompressDescription,
		container.NewHBox(
			decompressButton,
			decompressCancelButton,
			layout.NewSpacer(),
		),
		decompressProgressContainer,
	)

	return container.New(
		toolsGridLayout{},
		container.NewPadded(splitPanel),
		widget.NewSeparator(),
		container.NewPadded(mergePanel),
		widget.NewSeparator(),
		container.NewPadded(compressPanel),
		widget.NewSeparator(),
		container.NewPadded(decompressPanel),
		widget.NewSeparator(),
	)
}

type toolsGridLayout struct{}

func (toolsGridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 8 {
		return
	}

	if size.Width < toolsSingleColumnBreakpoint {
		layoutToolsSingleColumn(objects, size)
		return
	}

	separatorHeight := max(objects[3].MinSize().Height, toolsDividerThickness)
	separatorWidth := max(objects[7].MinSize().Width, toolsDividerThickness)
	columnWidth := max(float32(0), (size.Width-separatorWidth)/2)
	topHeight := max(objects[0].MinSize().Height, objects[2].MinSize().Height)
	if topHeight+separatorHeight > size.Height {
		topHeight = max(float32(0), size.Height-separatorHeight)
	}
	bottomHeight := max(float32(0), size.Height-topHeight-separatorHeight)

	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(columnWidth, topHeight))
	hideCanvasObject(objects[1])
	objects[2].Move(fyne.NewPos(columnWidth+separatorWidth, 0))
	objects[2].Resize(fyne.NewSize(columnWidth, topHeight))
	objects[3].Move(fyne.NewPos(0, topHeight))
	objects[3].Resize(fyne.NewSize(size.Width, separatorHeight))
	objects[4].Move(fyne.NewPos(0, topHeight+separatorHeight))
	objects[4].Resize(fyne.NewSize(columnWidth, bottomHeight))
	hideCanvasObject(objects[5])
	objects[6].Move(fyne.NewPos(columnWidth+separatorWidth, topHeight+separatorHeight))
	objects[6].Resize(fyne.NewSize(columnWidth, bottomHeight))
	objects[7].Move(fyne.NewPos(columnWidth, 0))
	objects[7].Resize(fyne.NewSize(separatorWidth, size.Height))
}

func (toolsGridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 8 {
		return fyne.NewSize(0, 0)
	}

	split := objects[0].MinSize()
	merge := objects[2].MinSize()
	separatorA := objects[1].MinSize()
	separatorB := objects[3].MinSize()
	compress := objects[4].MinSize()
	separatorC := objects[5].MinSize()
	decompress := objects[6].MinSize()

	return fyne.NewSize(
		max(split.Width, merge.Width, compress.Width, decompress.Width),
		split.Height+
			max(separatorA.Height, toolsDividerThickness)+
			merge.Height+
			max(separatorB.Height, toolsDividerThickness)+
			compress.Height+
			max(separatorC.Height, toolsDividerThickness)+
			decompress.Height,
	)
}

func layoutToolsSingleColumn(objects []fyne.CanvasObject, size fyne.Size) {
	hideCanvasObject(objects[7])

	y := float32(0)
	for _, index := range []int{0, 1, 2, 3, 4, 5, 6} {
		height := objects[index].MinSize().Height
		if index == 1 || index == 3 || index == 5 {
			height = max(height, toolsDividerThickness)
		}
		objects[index].Move(fyne.NewPos(0, y))
		objects[index].Resize(fyne.NewSize(size.Width, height))
		y += height
	}
}

func hideCanvasObject(object fyne.CanvasObject) {
	object.Move(fyne.NewPos(0, 0))
	object.Resize(fyne.NewSize(0, 0))
}
