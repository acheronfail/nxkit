package gui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
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
	name  string
	path  string
	size  int64
	isDir bool
}

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

	content := container.NewVBox()
	selectedPath := widget.NewLabel("No entry selected")

	var rebuildPartitions func()
	var rebuildMounted func()
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
		content.Objects = nil
		content.Add(widget.NewLabel("No NAND is open."))
		content.Refresh()
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
		readOnly.Disable()
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
		content.Objects = nil
		if model.table == nil {
			content.Add(widget.NewLabel("No NAND is open."))
			content.Refresh()
			return
		}
		content.Add(widget.NewLabelWithStyle("Choose a partition to explore", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		content.Add(container.NewHBox(widget.NewButton("Verify partition table", verifyPartitionTable), layout.NewSpacer()))
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
			content.Add(row)
		}
		content.Refresh()
	}

	rebuildMounted = func() {
		content.Objects = nil
		tree := widget.NewTree(
			func(uid widget.TreeNodeID) []widget.TreeNodeID {
				children, err := model.children(uid)
				if err != nil {
					status.SetText(err.Error())
					return nil
				}
				ids := make([]widget.TreeNodeID, len(children))
				for i, child := range children {
					ids[i] = child.path
				}
				return ids
			},
			func(uid widget.TreeNodeID) bool {
				node, ok := model.nodeLookup[uid]
				return uid == "/" || (ok && node.isDir)
			},
			func(branch bool) fyne.CanvasObject {
				return widget.NewLabel("entry")
			},
			func(uid widget.TreeNodeID, branch bool, object fyne.CanvasObject) {
				node := model.nodeLookup[uid]
				if uid == "/" {
					node = nandNode{name: "/", path: "/", isDir: true}
				}
				prefix := "  "
				if branch {
					prefix = "[DIR] "
				}
				label := object.(*widget.Label)
				if node.isDir {
					label.SetText(prefix + node.name)
				} else {
					label.SetText(fmt.Sprintf("%s%s (%s)", prefix, node.name, formatBytes(node.size)))
				}
			},
		)
		tree.Root = "/"
		tree.HideSeparators = true
		tree.OnSelected = func(uid widget.TreeNodeID) {
			selectedPath.SetText(uid)
		}
		tree.OpenBranch("/")

		importButton := widget.NewButton("Copy file in", func() {
			dirPath := selectedPath.Text
			if node, ok := model.nodeLookup[dirPath]; ok && !node.isDir {
				dirPath = filepath.Dir(dirPath)
			}
			chooseNativeFile("Choose file to copy into NAND", nil, func(hostPath string) {
				runAsync(nil, func() error {
					return model.copyFileIn(dirPath, hostPath)
				}, func() {
					model.invalidate(dirPath)
					tree.Refresh()
				})
			})
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
					tree.Refresh()
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
			deleteButton.Disable()
		}

		content.Add(widget.NewLabelWithStyle("Currently exploring "+model.part.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		content.Add(container.NewHBox(widget.NewButton("Choose another partition", rebuildPartitions), layout.NewSpacer()))
		content.Add(container.NewBorder(nil, container.NewVBox(selectedPath, container.NewHBox(exportButton, importButton, deleteButton)), nil, nil, tree))
		content.Refresh()
	}

	chooseNand := widget.NewButton("Choose your rawnand.bin", func() {
		chooseNativeFile("Choose NAND dump", nil, func(path string) {
			runAsync(nil, func() error {
				return openDump(path)
			}, func() {
				status.SetText("Opened " + path)
				rebuildPartitions()
			})
		})
	})
	closeButton := widget.NewButton("Close NAND", closeNand)

	content.Add(widget.NewLabel("No NAND is open."))
	return container.NewBorder(
		container.NewVBox(status, container.NewHBox(chooseNand, closeButton, layout.NewSpacer(), readOnly)),
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

func (n *nandExplorerState) copyFileIn(dirPathInNand string, hostPath string) error {
	input, err := os.Open(hostPath)
	if err != nil {
		return err
	}
	defer input.Close()
	targetPath := filepath.Join(dirPathInNand, filepath.Base(hostPath))
	if dirPathInNand == "/" {
		targetPath = "/" + filepath.Base(hostPath)
	}
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
			written, writeErr := file.WriteAt(buf[:read], off)
			if writeErr != nil {
				return writeErr
			}
			off += int64(written)
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
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
