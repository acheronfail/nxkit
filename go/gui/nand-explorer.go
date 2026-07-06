package gui

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/fat"
	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/fat/fat_auto"
	"github.com/acheronfail/nxkit/lib/nand"
	"github.com/acheronfail/nxkit/lib/xtsn"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

type nxPartitionInfo struct {
	format      string
	bisKeyID    int
	hasBISKeyID bool
	magicOffset int64
	magicBytes  []byte
}

var nxPartitions = map[string]nxPartitionInfo{
	"98109e25-64e2-4c95-8a77-414916f5bceb": {format: "unknown", bisKeyID: 0, hasBISKeyID: true, magicOffset: 0, magicBytes: []byte("CAL0")},
	"f3056aec-5449-494c-9f2c-5fdcb75b6e6e": {format: "fat12", bisKeyID: 0, hasBISKeyID: true, magicOffset: 0x2b, magicBytes: []byte("NO NAME")},
	"5365de36-911b-4bb4-8ff9-aa1ebcd73990": {format: "unknown"},
	"8455717b-bd2b-4162-8454-91695218fc38": {format: "unknown"},
	"8ed6c9a6-9c48-490b-bbeb-001d17a4c0f7": {format: "unknown"},
	"5e99751c-56c9-47cc-aa30-b65039888917": {format: "unknown"},
	"c447d9a2-24b7-468a-98c8-595cd077165a": {format: "unknown"},
	"9586e1a1-3aa2-4c90-91b3-2f4a5195b4d2": {format: "unknown"},
	"a44f9f6b-4ed3-441f-a34a-56aaa136bc6a": {format: "fat32", bisKeyID: 1, hasBISKeyID: true, magicOffset: 0x47, magicBytes: []byte("NO NAME")},
	"acb0cdf0-4f72-432d-aa0d-5388c733b224": {format: "fat32", bisKeyID: 2, hasBISKeyID: true, magicOffset: 0x47, magicBytes: []byte("NO NAME")},
	"2b777f63-e842-47af-94c4-25a7f18b2280": {format: "fat32", bisKeyID: 3, hasBISKeyID: true, magicOffset: 0x47, magicBytes: []byte("NO NAME")},
}

var (
	nandListBackgroundColor = color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xff}
	nandListBorderColor     = color.NRGBA{R: 0x3e, G: 0x42, B: 0x4a, A: 0xff}
	nandListRowEvenColor    = color.NRGBA{R: 0x10, G: 0x11, B: 0x15, A: 0xff}
	nandListRowOddColor     = color.NRGBA{R: 0x18, G: 0x19, B: 0x1e, A: 0xff}
	nandListDragColor       = color.NRGBA{R: 0x24, G: 0x2d, B: 0x3a, A: 0xff}
	nandListDropTargetColor = color.NRGBA{R: 0x1f, G: 0x36, B: 0x2d, A: 0xff}
)

type nandExplorerState struct {
	path       string
	readOnly   bool
	raw        backend.Storage
	table      *gpt.Table
	fs         fat.FileSystem
	part       *gpt.Partition
	nodeCache  map[string][]nandNode
	nodeLookup map[string]nandNode
}

type nandNode struct {
	name     string
	path     string
	size     int64
	isDir    bool
	isParent bool
}

type visibleNandNode struct {
	node  nandNode
	index int
}

type nandFileListRow struct {
	widget.BaseWidget

	node        nandNode
	index       int
	selected    bool
	readOnly    bool
	dragging    bool
	dropTarget  bool
	lastDragPos fyne.Position
	hasDragged  bool
	onSelect    func(string)
	onOpenDir   func(string)
	onDragMove  func(string, fyne.Position)
	onDragEnd   func(string, fyne.Position)
}

type nandFileListRowRenderer struct {
	row        *nandFileListRow
	background *canvas.Rectangle
	typeIcon   *widget.Icon
	nameText   *canvas.Text
	sizeText   *canvas.Text
	objects    []fyne.CanvasObject
}

type nandCopyProgressFunc func(string, int64)

func NandExplorerTab() fyne.CanvasObject {
	model := &nandExplorerState{
		readOnly:   true,
		nodeCache:  map[string][]nandNode{},
		nodeLookup: map[string]nandNode{"/": {name: "/", path: "/", isDir: true}},
	}
	status := widget.NewLabel("Choose your rawnand.bin or rawnand.bin.00 file to begin.")
	status.Wrapping = fyne.TextWrapWord
	readOnly := widget.NewCheck("Read-Only", func(checked bool) {
		model.readOnly = checked
	})
	readOnly.SetChecked(true)

	content := container.NewStack(widget.NewLabel("No NAND is open."))
	setContent := func(object fyne.CanvasObject) {
		content.Objects = []fyne.CanvasObject{object}
		content.Refresh()
	}
	selectedPath := widget.NewLabel("No entry selected")

	var rebuildPartitions func()
	var rebuildMounted func()
	var chooseNand *widget.Button
	var closeButton *widget.Button
	var topControls *fyne.Container
	var choosePartitionButton *widget.Button
	copyInProgress := false
	refreshTopControls := func() {
		if topControls == nil || chooseNand == nil || closeButton == nil {
			return
		}
		controls := make([]fyne.CanvasObject, 0, 3)
		if model.raw == nil {
			controls = append(controls, chooseNand)
		} else {
			controls = append(controls, closeButton)
		}
		controls = append(controls, layout.NewSpacer(), readOnly)
		topControls.Objects = controls
		topControls.Refresh()
	}
	updateCopyControls := func() {
		if closeButton != nil {
			if copyInProgress {
				closeButton.Disable()
			} else if model.raw != nil {
				closeButton.Enable()
			}
		}
		if choosePartitionButton != nil {
			if copyInProgress {
				choosePartitionButton.Disable()
			} else {
				choosePartitionButton.Enable()
			}
		}
	}
	closeNand := func() {
		if model.raw != nil {
			_ = model.raw.Close()
		}
		model.raw = nil
		model.table = nil
		model.fs = nil
		model.part = nil
		model.path = ""
		model.nodeCache = map[string][]nandNode{}
		model.nodeLookup = map[string]nandNode{"/": {name: "/", path: "/", isDir: true}}
		readOnly.Enable()
		status.SetText("Choose your rawnand.bin or rawnand.bin.00 file to begin.")
		choosePartitionButton = nil
		setContent(widget.NewLabel("No NAND is open."))
		refreshTopControls()
		updateCopyControls()
	}

	openDump := func(path string) error {
		if model.raw != nil {
			_ = model.raw.Close()
		}
		raw, err := nand.NewDumpBackend(path, model.readOnly)
		if err != nil {
			return err
		}
		table, err := gpt.Read(raw, 512, 512)
		if err != nil {
			_ = raw.Close()
			return err
		}
		model.raw = raw
		model.table = table
		model.path = path
		model.fs = nil
		model.part = nil
		model.nodeCache = map[string][]nandNode{}
		model.nodeLookup = map[string]nandNode{"/": {name: "/", path: "/", isDir: true}}
		return nil
	}

	verifyPartitionTable := func() {
		if model.raw == nil || model.table == nil {
			showError(fmt.Errorf("no NAND is open"))
			return
		}
		stat, err := model.raw.Stat()
		if err != nil {
			showError(err)
			return
		}
		if err := model.table.Verify(model.raw, uint64(stat.Size())); err != nil {
			showError(err)
			return
		}
		dialog.ShowInformation("Partition table", "Partition table is valid.", mainWindow)
	}

	mountPartition := func(part *gpt.Partition) error {
		info := nxPartitionInfoFor(part)
		if !isMountablePartition(info) {
			return fmt.Errorf("unsupported partition format for %s", part.Name)
		}
		partBackend := nand.NewNxPartBackend(model.raw, part.Start*512, (part.End+1)*512, 16, 512)
		cleartext := partitionHasMagic(model.raw, part, info)
		if !cleartext && info.hasBISKeyID {
			if state.Keys == nil {
				return fmt.Errorf("prod.keys are required to decrypt %s", part.Name)
			}
			if info.bisKeyID >= len(state.Keys.BisKey) || len(state.Keys.BisKey[info.bisKeyID]) != 32 {
				return fmt.Errorf("bis_key_%02d must be present and 32 bytes", info.bisKeyID)
			}
			bisKey := state.Keys.BisKey[info.bisKeyID]
			crypto, err := xtsn.NewXtsnCipher(bisKey[:16], bisKey[16:], 16384)
			if err != nil {
				return err
			}
			partBackend.SetCrypto(crypto)
			if !partitionBackendHasMagic(partBackend, part, info) {
				return fmt.Errorf("failed to decrypt %s; check that the selected prod.keys are correct", part.Name)
			}
		}
		fs, err := fat_auto.Open(partBackend, int64(part.Start)*512)
		if err != nil {
			return err
		}
		model.fs = fs
		model.part = part
		model.nodeCache = map[string][]nandNode{}
		model.nodeLookup = map[string]nandNode{"/": {name: "/", path: "/", isDir: true}}
		return nil
	}

	rebuildPartitions = func() {
		if model.table == nil {
			setContent(widget.NewLabel("No NAND is open."))
			return
		}
		choosePartitionButton = nil
		partitions := container.NewVBox()
		partitionsList := container.NewBorder(
			widget.NewLabelWithStyle("Choose a partition to explore", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil,
			nil,
			nil,
			container.NewBorder(
				container.NewHBox(widget.NewButton("Verify partition table", verifyPartitionTable), layout.NewSpacer()),
				nil,
				nil,
				nil,
				newNandListPanel(container.NewPadded(partitions)),
			),
		)
		for _, part := range model.table.Partitions {
			if part == nil {
				continue
			}
			info := nxPartitionInfoFor(part)
			mountable := isMountablePartition(info)
			size := formatBytes(int64(part.Size))
			button := widget.NewButton("Mount", func(p *gpt.Partition) func() {
				return func() {
					runAsync(nil, func() error {
						return mountPartition(p)
					}, func() {
						status.SetText(fmt.Sprintf("Mounted %s from %s", p.Name, model.path))
						rebuildMounted()
					})
				}
			}(part))
			if !mountable {
				button.Disable()
			}
			row := container.NewBorder(nil, nil, widget.NewLabel(fmt.Sprintf("%s (%s, %s)", part.Name, strings.ToUpper(info.format), size)), button)
			partitions.Add(row)
		}
		setContent(partitionsList)
	}

	rebuildMounted = func() {
		currentDir := "/"
		currentDirLabel := widget.NewLabel("Current directory: /")
		currentDirLabel.TextStyle = fyne.TextStyle{Monospace: true}
		listRows := container.New(layout.NewCustomPaddedVBoxLayout(0))
		fileList := container.NewVScroll(listRows)
		var renderedRows []*nandFileListRow
		var refreshFileList func()
		var dragPopup *widget.PopUp
		var dragPopupLabel *widget.Label
		var dragTargetPath string

		selectedDirPath := func() string {
			if currentDir == "" || currentDir == "." {
				return "/"
			}
			return currentDir
		}

		rowForPath := func(path string) (*nandFileListRow, bool) {
			for _, row := range renderedRows {
				if row.node.path == path {
					return row, true
				}
			}
			return nil, false
		}

		selectPath := func(path string) {
			previousPath := selectedPath.Text
			if previousPath == path {
				return
			}
			selectedPath.SetText(path)
			if previousRow, ok := rowForPath(previousPath); ok && previousRow.selected {
				previousRow.selected = false
				previousRow.Refresh()
			}
			if nextRow, ok := rowForPath(path); ok && !nextRow.selected {
				nextRow.selected = true
				nextRow.Refresh()
			}
		}

		navigateToDir := func(path string) {
			if path == "" || path == "." {
				path = "/"
			}
			currentDir = path
			currentDirLabel.SetText("Current directory: " + currentDir)
			selectedPath.SetText("No entry selected")
			refreshFileList()
		}

		rowAtAbsolute := func(position fyne.Position, sourcePath string) (nandNode, bool) {
			driver := fyne.CurrentApp().Driver()
			for _, row := range renderedRows {
				if row.node.path == sourcePath || !row.node.isDir || row.node.isParent {
					continue
				}
				if sourcePath != "" && isSameOrDescendantNandPath(row.node.path, sourcePath) {
					continue
				}
				rowPos := driver.AbsolutePositionForObject(row)
				rowSize := row.Size()
				if position.X >= rowPos.X &&
					position.X <= rowPos.X+rowSize.Width &&
					position.Y >= rowPos.Y &&
					position.Y <= rowPos.Y+rowSize.Height {
					return row.node, true
				}
			}
			return nandNode{}, false
		}

		moveEntryToDropTarget := func(sourcePath string, position fyne.Position) {
			if model.readOnly || sourcePath == "/" {
				return
			}
			targetPath := currentDir
			if target, ok := rowAtAbsolute(position, sourcePath); ok {
				targetPath = target.path
			}
			runAsync(nil, func() error {
				return model.moveEntry(sourcePath, targetPath)
			}, func() {
				model.invalidateAll()
				selectedPath.SetText(nandChildPath(targetPath, filepath.Base(sourcePath)))
				refreshFileList()
			})
		}

		setRowDragState := func(path string, dragging bool) {
			for _, row := range renderedRows {
				if row.node.path != path || row.dragging == dragging {
					continue
				}
				row.dragging = dragging
				row.Refresh()
				return
			}
		}

		setDropTarget := func(path string) {
			if path == dragTargetPath {
				return
			}
			for _, row := range renderedRows {
				shouldTarget := row.node.path == path && path != ""
				if row.dropTarget == shouldTarget {
					continue
				}
				row.dropTarget = shouldTarget
				row.Refresh()
			}
			dragTargetPath = path
		}

		hideDragPopup := func() {
			if dragPopup != nil {
				dragPopup.Hide()
			}
		}

		updateDragIndicator := func(sourcePath string, position fyne.Position) {
			setRowDragState(sourcePath, true)
			target, ok := rowAtAbsolute(position, sourcePath)
			if ok {
				setDropTarget(target.path)
			} else {
				setDropTarget("")
			}

			if dragPopupLabel == nil {
				dragPopupLabel = widget.NewLabel("")
			}
			dragPopupLabel.SetText("Move " + filepath.Base(sourcePath))
			if dragPopup == nil {
				canvas := fyne.CurrentApp().Driver().CanvasForObject(fileList)
				if canvas == nil {
					return
				}
				dragPopup = widget.NewPopUp(newNandDragPreview(dragPopupLabel), canvas)
			}
			dragPopup.ShowAtPosition(position.Add(fyne.NewPos(14, 14)))
		}

		finishDrag := func(sourcePath string, position fyne.Position) {
			hideDragPopup()
			setDropTarget("")
			setRowDragState(sourcePath, false)
			moveEntryToDropTarget(sourcePath, position)
		}

		refreshFileList = func() {
			children, err := model.children(currentDir)
			if err != nil {
				status.SetText(err.Error())
				return
			}
			nodes := make([]visibleNandNode, 0, len(children)+1)
			if currentDir != "/" {
				parent := filepath.Dir(currentDir)
				if parent == "." || parent == "" {
					parent = "/"
				}
				nodes = append(nodes, visibleNandNode{
					node:  nandNode{name: "..", path: parent, isDir: true, isParent: true},
					index: len(nodes),
				})
			}
			for _, child := range children {
				nodes = append(nodes, visibleNandNode{node: child, index: len(nodes)})
			}
			renderedRows = make([]*nandFileListRow, 0, len(nodes))
			objects := make([]fyne.CanvasObject, 0, len(nodes))
			for _, visible := range nodes {
				row := newNandFileListRow(
					visible.node,
					visible.index,
					visible.node.path == selectedPath.Text,
					model.readOnly,
					func(path string) {
						selectPath(path)
					},
					func(path string) {
						navigateToDir(path)
					},
					updateDragIndicator,
					finishDrag,
				)
				renderedRows = append(renderedRows, row)
				objects = append(objects, row)
			}
			listRows.Objects = objects
			listRows.Refresh()
		}
		refreshFileList()

		copyProgressLabel := widget.NewLabel("")
		copyProgressLabel.Hide()
		copyProgress := widget.NewProgressBar()
		copyProgress.Hide()
		copyProgressContainer := container.NewVBox(copyProgressLabel, copyProgress)
		copyProgressContainer.Hide()

		copyHostPathsIn := func(dirPath string, hostPaths []string) {
			if copyInProgress {
				return
			}
			totalBytes := int64(1)
			copiedBytes := int64(0)
			copyProgress.Min = 0
			copyProgress.Max = 1
			copyProgress.TextFormatter = func() string {
				return fmt.Sprintf("%s / %s", formatBytes(copiedBytes), formatBytes(totalBytes))
			}
			copyProgress.SetValue(0)
			copyProgressLabel.SetText("Preparing copy into " + dirPath)
			copyProgressLabel.Show()
			copyProgress.Show()
			copyProgressContainer.Show()
			copyInProgress = true
			updateCopyControls()
			go func() {
				countedBytes, countErr := countHostCopyBytes(hostPaths)
				if countErr != nil {
					fyne.Do(func() {
						copyInProgress = false
						updateCopyControls()
						copyProgressContainer.Hide()
						copyProgress.Hide()
						copyProgressLabel.Hide()
						showError(countErr)
					})
					return
				}
				if countedBytes > 0 {
					fyne.DoAndWait(func() {
						totalBytes = countedBytes
					})
				}
				fyne.DoAndWait(func() {
					copiedBytes = 0
					copyProgressLabel.SetText("Copying into " + dirPath)
					copyProgress.SetValue(0)
				})
				progress := func(hostPath string, copied int64) {
					fyne.Do(func() {
						copiedBytes += copied
						copyProgressLabel.SetText("Copying " + filepath.Base(hostPath))
						if totalBytes <= 0 {
							copyProgress.SetValue(1)
						} else {
							copyProgress.SetValue(float64(copiedBytes) / float64(totalBytes))
						}
					})
				}
				var copyErr error
				for _, hostPath := range hostPaths {
					if err := model.copyPathInWithProgress(dirPath, hostPath, progress); err != nil {
						copyErr = err
						break
					}
				}
				fyne.Do(func() {
					copyInProgress = false
					updateCopyControls()
					copyProgressContainer.Hide()
					copyProgress.Hide()
					copyProgressLabel.Hide()
					if copyErr != nil {
						showError(copyErr)
						return
					}
					copyProgress.SetValue(1)
					model.invalidateAll()
					refreshFileList()
				})
			}()
		}

		importButton := widget.NewButton("Copy file in", func() {
			dirPath := selectedDirPath()
			chooseNativeFile("Choose file to copy into NAND", nil, func(hostPath string) {
				copyHostPathsIn(dirPath, []string{hostPath})
			})
		})
		newFolderButton := widget.NewButton("New folder", func() {
			nameEntry := widget.NewEntry()
			nameEntry.SetPlaceHolder("Folder name")
			dialog.ShowForm("New folder", "Create", "Cancel", []*widget.FormItem{
				widget.NewFormItem("Name", nameEntry),
			}, func(ok bool) {
				if !ok {
					return
				}
				name := strings.TrimSpace(nameEntry.Text)
				if err := validateNandChildName(name); err != nil {
					showError(err)
					return
				}
				dirPath := selectedDirPath()
				runAsync(nil, func() error {
					return model.createDirectory(dirPath, name)
				}, func() {
					model.invalidate(dirPath)
					refreshFileList()
				})
			}, mainWindow)
		})
		deleteButton := widget.NewButton("Delete", func() {
			path := selectedPath.Text
			if path == "" || path == "/" || path == "No entry selected" {
				return
			}
			dialog.ShowConfirm("Delete entry", "Delete "+path+" from NAND?", func(ok bool) {
				if !ok {
					return
				}
				runAsync(nil, func() error {
					return model.delete(path)
				}, func() {
					parent := filepath.Dir(path)
					if parent == "." {
						parent = "/"
					}
					model.invalidate(parent)
					selectedPath.SetText("No entry selected")
					refreshFileList()
				})
			}, mainWindow)
		})
		exportButton := widget.NewButton("Copy file out", func() {
			path := selectedPath.Text
			node, ok := model.nodeLookup[path]
			if !ok || node.isDir {
				return
			}
			saveNativeFile("Copy file out", node.name, nil, func(destPath string) {
				runAsync(nil, func() error {
					return model.copyFileOut(path, destPath)
				}, func() {
					sendAppNotification("File copied out", filepath.Base(destPath))
				})
			})
		})
		if model.readOnly {
			importButton.Disable()
			newFolderButton.Disable()
			deleteButton.Disable()
		}
		mainWindow.SetOnDropped(func(position fyne.Position, uris []fyne.URI) {
			if model.fs == nil || model.part == nil || model.readOnly {
				return
			}
			hostPaths := make([]string, 0, len(uris))
			for _, uri := range uris {
				if uri == nil {
					continue
				}
				if uri.Scheme() != "file" {
					showError(fmt.Errorf("cannot copy non-file URI into NAND: %s", uri.String()))
					return
				}
				hostPaths = append(hostPaths, uri.Path())
			}
			if len(hostPaths) == 0 {
				return
			}
			dirPath := selectedDirPath()
			if target, ok := rowAtAbsolute(position, ""); ok {
				dirPath = target.path
			}
			copyHostPathsIn(dirPath, hostPaths)
		})

		choosePartitionButton = widget.NewButton("Choose another partition", rebuildPartitions)
		updateCopyControls()
		top := container.NewVBox(
			widget.NewLabelWithStyle("Currently exploring "+model.part.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			currentDirLabel,
			container.NewHBox(choosePartitionButton, layout.NewSpacer()),
		)
		bottom := container.NewVBox(selectedPath, container.NewHBox(exportButton, importButton, newFolderButton, deleteButton), copyProgressContainer)
		setContent(container.NewBorder(top, bottom, nil, nil, newNandListPanel(fileList)))
	}

	chooseNand = widget.NewButton("Choose NAND", func() {
		chooseNativeFile("Choose NAND dump", nil, func(path string) {
			runAsync(nil, func() error {
				return openDump(path)
			}, func() {
				readOnly.Disable()
				status.SetText("Opened " + path)
				refreshTopControls()
				updateCopyControls()
				rebuildPartitions()
			})
		})
	})
	closeButton = widget.NewButton("Close NAND", closeNand)
	topControls = container.NewHBox()
	refreshTopControls()
	updateCopyControls()

	return container.NewBorder(
		container.NewVBox(status, topControls),
		nil,
		nil,
		nil,
		content,
	)
}

func nxPartitionInfoFor(part *gpt.Partition) nxPartitionInfo {
	if part == nil {
		return nxPartitionInfo{format: "unknown"}
	}
	if info, ok := nxPartitions[strings.ToLower(string(part.Type))]; ok {
		return info
	}
	return nxPartitionInfo{format: "unknown"}
}

func isMountablePartition(info nxPartitionInfo) bool {
	return info.format == "fat12" || info.format == "fat32"
}

func newNandListPanel(content fyne.CanvasObject) fyne.CanvasObject {
	background := canvas.NewRectangle(nandListBackgroundColor)
	border := canvas.NewRectangle(color.NRGBA{A: 0})
	border.StrokeColor = nandListBorderColor
	border.StrokeWidth = 1
	return container.NewMax(background, content, border)
}

func newNandDragPreview(label *widget.Label) fyne.CanvasObject {
	background := canvas.NewRectangle(nandListDragColor)
	background.StrokeColor = nandListBorderColor
	background.StrokeWidth = 1
	return container.NewMax(background, container.NewPadded(label))
}

func countHostCopyBytes(hostPaths []string) (int64, error) {
	total := int64(0)
	for _, hostPath := range hostPaths {
		count, err := countHostCopyPathBytes(hostPath, true)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func countHostCopyPathBytes(hostPath string, topLevel bool) (int64, error) {
	info, err := os.Stat(hostPath)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			if topLevel {
				return 0, fmt.Errorf("cannot copy non-regular file into NAND: %s", hostPath)
			}
			return 0, nil
		}
		return info.Size(), nil
	}

	total := int64(0)
	entries, err := os.ReadDir(hostPath)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		childCount, err := countHostCopyPathBytes(filepath.Join(hostPath, entry.Name()), false)
		if err != nil {
			return 0, err
		}
		total += childCount
	}
	return total, nil
}

func partitionHasMagic(raw backend.Storage, part *gpt.Partition, info nxPartitionInfo) bool {
	if len(info.magicBytes) == 0 {
		return false
	}
	data := make([]byte, len(info.magicBytes))
	_, err := raw.ReadAt(data, int64(part.Start*512)+info.magicOffset)
	return err == nil && bytes.Equal(data, info.magicBytes)
}

func partitionBackendHasMagic(partBackend backend.Storage, part *gpt.Partition, info nxPartitionInfo) bool {
	if len(info.magicBytes) == 0 {
		return true
	}
	data := make([]byte, len(info.magicBytes))
	_, err := partBackend.ReadAt(data, int64(part.Start*512)+info.magicOffset)
	return err == nil && bytes.Equal(data, info.magicBytes)
}

func newNandFileListRow(
	node nandNode,
	index int,
	selected bool,
	readOnly bool,
	onSelect func(string),
	onOpenDir func(string),
	onDragMove func(string, fyne.Position),
	onDragEnd func(string, fyne.Position),
) *nandFileListRow {
	row := &nandFileListRow{
		node:       node,
		index:      index,
		selected:   selected,
		readOnly:   readOnly,
		onSelect:   onSelect,
		onOpenDir:  onOpenDir,
		onDragMove: onDragMove,
		onDragEnd:  onDragEnd,
	}
	row.ExtendBaseWidget(row)
	return row
}

func (r *nandFileListRow) Tapped(_ *fyne.PointEvent) {
	if r.onSelect != nil {
		r.onSelect(r.node.path)
	}
	if r.node.isDir {
		if r.onOpenDir != nil {
			r.onOpenDir(r.node.path)
		}
	}
}

func (r *nandFileListRow) Dragged(event *fyne.DragEvent) {
	if r.readOnly || r.node.path == "/" || r.node.isParent {
		return
	}
	r.lastDragPos = event.AbsolutePosition
	r.hasDragged = true
	if r.onDragMove != nil {
		r.onDragMove(r.node.path, event.AbsolutePosition)
	}
}

func (r *nandFileListRow) DragEnd() {
	if !r.hasDragged {
		return
	}
	r.hasDragged = false
	if r.onDragEnd != nil {
		r.onDragEnd(r.node.path, r.lastDragPos)
	}
}

func (r *nandFileListRow) Cursor() desktop.Cursor {
	if r.node.isDir || (!r.readOnly && r.node.path != "/") {
		return desktop.PointerCursor
	}
	if r.readOnly || r.node.path == "/" {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

func (r *nandFileListRow) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(nandListRowEvenColor)
	typeIcon := widget.NewIcon(nil)
	nameText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	nameText.TextStyle = fyne.TextStyle{Monospace: true}
	nameText.TextSize = theme.Size(theme.SizeNameText)
	sizeText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	sizeText.Alignment = fyne.TextAlignTrailing
	sizeText.TextStyle = fyne.TextStyle{Monospace: true}
	sizeText.TextSize = theme.Size(theme.SizeNameText)
	renderer := &nandFileListRowRenderer{
		row:        r,
		background: background,
		typeIcon:   typeIcon,
		nameText:   nameText,
		sizeText:   sizeText,
		objects:    []fyne.CanvasObject{background, typeIcon, nameText, sizeText},
	}
	renderer.Refresh()
	return renderer
}

func (r *nandFileListRowRenderer) Layout(size fyne.Size) {
	r.background.Move(fyne.NewPos(0, 0))
	r.background.Resize(size)

	iconSize := float32(16)
	iconY := (size.Height - iconSize) / 2
	typeIconX := float32(6)
	r.typeIcon.Move(fyne.NewPos(typeIconX, iconY))
	r.typeIcon.Resize(fyne.NewSize(iconSize, iconSize))

	nameX := typeIconX + iconSize + 6
	if nameX > size.Width {
		nameX = 0
	}
	sizeWidth := float32(0)
	if !r.row.node.isDir {
		sizeWidth = r.sizeText.MinSize().Width + 12
	}
	sizeX := size.Width - sizeWidth - 12
	if sizeX < nameX {
		sizeX = nameX
	}
	nameWidth := sizeX - nameX - 8
	if nameWidth < 0 {
		nameWidth = 0
	}

	displayName := truncateTextToWidth(r.row.node.name, nameWidth, r.nameText.TextSize, r.nameText.TextStyle)
	if r.nameText.Text != displayName {
		r.nameText.Text = displayName
		r.nameText.Refresh()
	}

	nameHeight := r.nameText.MinSize().Height
	nameY := (size.Height - nameHeight) / 2
	if nameY < 0 {
		nameY = 0
		nameHeight = size.Height
	}
	r.nameText.Move(fyne.NewPos(nameX, nameY))
	r.nameText.Resize(fyne.NewSize(nameWidth, nameHeight))

	sizeHeight := r.sizeText.MinSize().Height
	sizeY := (size.Height - sizeHeight) / 2
	if sizeY < 0 {
		sizeY = 0
		sizeHeight = size.Height
	}
	r.sizeText.Move(fyne.NewPos(sizeX, sizeY))
	r.sizeText.Resize(fyne.NewSize(sizeWidth, sizeHeight))
}

func (r *nandFileListRowRenderer) MinSize() fyne.Size {
	return fyne.NewSize(320, 28)
}

func (r *nandFileListRowRenderer) Refresh() {
	r.nameText.Color = theme.Color(theme.ColorNameForeground)
	r.nameText.TextSize = theme.Size(theme.SizeNameText)
	r.sizeText.Color = theme.Color(theme.ColorNameForeground)
	r.sizeText.TextSize = theme.Size(theme.SizeNameText)
	if r.row.dropTarget {
		r.background.FillColor = nandListDropTargetColor
	} else if r.row.dragging {
		r.background.FillColor = nandListDragColor
	} else if r.row.selected {
		r.background.FillColor = theme.Color(theme.ColorNameSelection)
	} else if r.row.index%2 == 0 {
		r.background.FillColor = nandListRowEvenColor
	} else {
		r.background.FillColor = nandListRowOddColor
	}
	r.background.Refresh()

	if r.row.node.isDir {
		if r.row.node.isParent {
			r.typeIcon.SetResource(theme.NavigateBackIcon())
		} else {
			r.typeIcon.SetResource(theme.FolderIcon())
		}
		r.sizeText.Text = ""
	} else {
		r.typeIcon.SetResource(theme.FileIcon())
		r.sizeText.Text = formatBytes(r.row.node.size)
	}
	r.nameText.Refresh()
	r.sizeText.Refresh()
}

func (r *nandFileListRowRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *nandFileListRowRenderer) Destroy() {}

func (n *nandExplorerState) children(path string) ([]nandNode, error) {
	if children, ok := n.nodeCache[path]; ok {
		return children, nil
	}
	entries, err := n.fs.ReadDir(path)
	if err != nil {
		return nil, err
	}
	children := make([]nandNode, 0, len(entries))
	for _, entry := range entries {
		name := entry.LongName()
		if name == "." || name == ".." {
			continue
		}
		childPath := filepath.Join(path, name)
		if path == "/" {
			childPath = "/" + name
		}
		node := nandNode{name: name, path: childPath, size: entry.Size(), isDir: entry.IsDir()}
		children = append(children, node)
		n.nodeLookup[childPath] = node
	}
	sort.Slice(children, func(i, j int) bool {
		if children[i].isDir != children[j].isDir {
			return children[i].isDir
		}
		return strings.ToLower(children[i].name) < strings.ToLower(children[j].name)
	})
	n.nodeCache[path] = children
	return children, nil
}

func (n *nandExplorerState) invalidate(path string) {
	delete(n.nodeCache, path)
}

func (n *nandExplorerState) invalidateAll() {
	n.nodeCache = map[string][]nandNode{}
	n.nodeLookup = map[string]nandNode{"/": {name: "/", path: "/", isDir: true}}
}

func (n *nandExplorerState) copyFileOut(pathInNand string, destPath string) error {
	file, err := n.fs.OpenFile(pathInNand, os.O_RDONLY)
	if err != nil {
		return err
	}
	defer file.Close()
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 1024*1024)
	var off int64
	for {
		read, err := file.ReadAt(buf, off)
		if read > 0 {
			if _, writeErr := out.Write(buf[:read]); writeErr != nil {
				return writeErr
			}
			off += int64(read)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (n *nandExplorerState) copyPathIn(dirPathInNand string, hostPath string) error {
	return n.copyPathInWithProgress(dirPathInNand, hostPath, nil)
}

func (n *nandExplorerState) copyPathInWithProgress(dirPathInNand string, hostPath string, progress nandCopyProgressFunc) error {
	inputInfo, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if inputInfo.IsDir() {
		return n.copyDirectoryInWithProgress(dirPathInNand, hostPath, progress)
	}
	if !inputInfo.Mode().IsRegular() {
		return fmt.Errorf("cannot copy non-regular file into NAND: %s", hostPath)
	}
	return n.copyFileInWithProgress(dirPathInNand, hostPath, progress)
}

func (n *nandExplorerState) copyDirectoryIn(dirPathInNand string, hostPath string) error {
	return n.copyDirectoryInWithProgress(dirPathInNand, hostPath, nil)
}

func (n *nandExplorerState) copyDirectoryInWithProgress(dirPathInNand string, hostPath string, progress nandCopyProgressFunc) error {
	name := filepath.Base(hostPath)
	if err := validateNandChildName(name); err != nil {
		return err
	}
	targetPath := nandChildPath(dirPathInNand, name)
	if err := n.ensureDirectory(targetPath); err != nil {
		return err
	}
	entries, err := os.ReadDir(hostPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		childHostPath := filepath.Join(hostPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := n.copyDirectoryInWithProgress(targetPath, childHostPath, progress); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := n.copyFileInWithProgress(targetPath, childHostPath, progress); err != nil {
			return err
		}
	}
	return nil
}

func (n *nandExplorerState) copyFileIn(dirPathInNand string, hostPath string) error {
	return n.copyFileInWithProgress(dirPathInNand, hostPath, nil)
}

func (n *nandExplorerState) copyFileInWithProgress(dirPathInNand string, hostPath string, progress nandCopyProgressFunc) error {
	input, err := os.Open(hostPath)
	if err != nil {
		return err
	}
	defer input.Close()
	inputInfo, err := input.Stat()
	if err != nil {
		return err
	}
	if inputInfo.IsDir() {
		return fmt.Errorf("cannot copy directory into NAND: %s", hostPath)
	}
	targetPath := nandChildPath(dirPathInNand, filepath.Base(hostPath))
	file, err := n.fs.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR)
	if err != nil {
		return err
	}
	defer file.Close()
	buf := make([]byte, 1024*1024)
	var off int64
	for {
		read, readErr := input.Read(buf)
		if read > 0 {
			chunk := buf[:read]
			for len(chunk) > 0 {
				written, writeErr := file.WriteAt(chunk, off)
				if written > 0 {
					off += int64(written)
					chunk = chunk[written:]
					if progress != nil {
						progress(hostPath, int64(written))
					}
				}
				if writeErr != nil {
					return writeErr
				}
				if written == 0 {
					return io.ErrUnexpectedEOF
				}
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (n *nandExplorerState) createDirectory(dirPathInNand string, name string) error {
	if err := validateNandChildName(name); err != nil {
		return err
	}
	return n.ensureDirectory(nandChildPath(dirPathInNand, name))
}

func (n *nandExplorerState) moveEntry(sourcePath string, targetDirPath string) error {
	if sourcePath == "" || sourcePath == "/" {
		return fmt.Errorf("cannot move root directory")
	}
	source, err := n.fs.Stat(sourcePath)
	if err != nil {
		return err
	}
	if targetDirPath != "/" {
		target, err := n.fs.Stat(targetDirPath)
		if err != nil {
			return err
		}
		if !target.IsDir() {
			return fmt.Errorf("drop target is not a directory: %s", targetDirPath)
		}
	}
	if source.IsDir() && isSameOrDescendantNandPath(targetDirPath, sourcePath) {
		return fmt.Errorf("cannot move a directory into itself")
	}

	targetPath := nandChildPath(targetDirPath, filepath.Base(sourcePath))
	if targetPath == sourcePath {
		return nil
	}
	if _, err := n.fs.Stat(targetPath); err == nil {
		return fmt.Errorf("destination already exists: %s", targetPath)
	}
	return n.fs.Rename(sourcePath, targetPath)
}

func (n *nandExplorerState) ensureDirectory(pathInNand string) error {
	if err := n.fs.Mkdir(pathInNand); err != nil {
		stat, statErr := n.fs.Stat(pathInNand)
		if statErr == nil && stat.IsDir() {
			return nil
		}
		return err
	}
	return nil
}

func validateNandChildName(name string) error {
	if name == "" {
		return fmt.Errorf("folder name is required")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("folder name cannot contain path separators")
	}
	return nil
}

func nandChildPath(dirPathInNand string, name string) string {
	if dirPathInNand == "" || dirPathInNand == "." || dirPathInNand == "/" {
		return "/" + name
	}
	return filepath.Join(dirPathInNand, name)
}

func isSameOrDescendantNandPath(path string, parent string) bool {
	if parent == "" || parent == "." {
		parent = "/"
	}
	if path == parent {
		return true
	}
	if parent == "/" {
		return strings.HasPrefix(path, "/")
	}
	return strings.HasPrefix(path, parent+"/")
}

func truncateTextToWidth(text string, maxWidth float32, textSize float32, textStyle fyne.TextStyle) string {
	if maxWidth <= 0 {
		return ""
	}
	if fyne.MeasureText(text, textSize, textStyle).Width <= maxWidth {
		return text
	}
	suffix := "..."
	if fyne.MeasureText(suffix, textSize, textStyle).Width > maxWidth {
		return ""
	}
	runes := []rune(text)
	low := 0
	high := len(runes)
	for low < high {
		mid := (low + high + 1) / 2
		candidate := string(runes[:mid]) + suffix
		if fyne.MeasureText(candidate, textSize, textStyle).Width <= maxWidth {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return string(runes[:low]) + suffix
}

func (n *nandExplorerState) delete(pathInNand string) error {
	node, ok := n.nodeLookup[pathInNand]
	if !ok {
		return fmt.Errorf("no selected NAND entry")
	}
	if node.isDir {
		return n.fs.Rmdir(pathInNand)
	}
	return n.fs.Unlink(pathInNand)
}
