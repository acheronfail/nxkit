package gui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/inject"
)

const payloadDirectoryWatchInterval = 1 * time.Second

var (
	//go:embed markdown/rcm-help.linux.md
	rcmHelpLinuxMarkdown string
	//go:embed markdown/rcm-help.windows.md
	rcmHelpWindowsMarkdown string
	//go:embed markdown/rcm-help.generic.md
	rcmHelpGenericMarkdown string
	//go:embed markdown/download-hekate.md
	downloadHekateMarkdown string
	//go:embed markdown/download-fusee.md
	downloadFuseeMarkdown string
)

type payloadDownloadOption struct {
	key         string
	displayName string
	link        string
	markdown    string
}

type payloadInjectorTab struct {
	content   fyne.CanvasObject
	reload    func()
	watchMu   sync.Mutex
	watchStop chan struct{}
}

var payloadDownloadOptions = []payloadDownloadOption{
	{
		key:         "hekate",
		displayName: "hekate_ctcaer.bin",
		link:        "https://github.com/CTCaer/hekate/releases/latest",
		markdown:    downloadHekateMarkdown,
	},
	{
		key:         "fusee",
		displayName: "fusee.bin",
		link:        "https://github.com/Atmosphere-NX/Atmosphere/releases/latest",
		markdown:    downloadFuseeMarkdown,
	},
	{
		key:         "tegraExplorer",
		displayName: "tegraexplorer.bin",
		link:        "https://github.com/suchmememanyskill/TegraExplorer/releases/latest",
	},
}

func getPayloads() ([]string, error) {
	if err := os.MkdirAll(state.PayloadDirectory, 0o755); err != nil {
		return nil, err
	}
	files, err := os.ReadDir(state.PayloadDirectory)
	if err != nil {
		return nil, err
	}

	var payloads []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".bin") {
			payloads = append(payloads, filepath.Join(state.PayloadDirectory, file.Name()))
		}
	}

	return payloads, nil
}

func PayloadInjectorTab() fyne.CanvasObject {
	return newPayloadInjectorTab().content
}

func newPayloadInjectorTab() *payloadInjectorTab {
	emptyWidget := widget.NewLabelWithStyle("No payloads found.", fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})
	output := newPayloadOutput()
	output.Hide()

	payloadRows := container.New(layout.NewCustomPaddedVBoxLayout(0))
	payloadList := container.NewVScroll(payloadRows)
	payloadListContent := container.NewStack(payloadList)
	payloadListPanel := newNandListPanel(payloadListContent)

	reloadPaths := func() {
		payloadPaths, err := getPayloads()
		if err != nil {
			showError(err)
			return
		}

		if len(payloadPaths) > 0 {
			rows := make([]fyne.CanvasObject, 0, len(payloadPaths))
			for i, payloadPath := range payloadPaths {
				rows = append(rows, newPayloadListRow(payloadPath, i, output))
			}
			payloadRows.Objects = rows
			payloadRows.Refresh()
			payloadListContent.Objects = []fyne.CanvasObject{payloadList}
		} else {
			payloadRows.Objects = nil
			payloadRows.Refresh()
			payloadListContent.Objects = []fyne.CanvasObject{container.NewCenter(emptyWidget)}
		}
		payloadListContent.Refresh()
	}

	reloadPaths()

	nxkitApp.Lifecycle().SetOnEnteredForeground(reloadPaths)

	payloadSelectOptions := []string{"select payload"}
	payloadByDisplayName := make(map[string]payloadDownloadOption, len(payloadDownloadOptions))
	for _, option := range payloadDownloadOptions {
		payloadSelectOptions = append(payloadSelectOptions, option.displayName)
		payloadByDisplayName[option.displayName] = option
	}

	downloadDetails := widget.NewRichTextFromMarkdown("Select a payload to see where to download it.")
	downloadDetails.Wrapping = fyne.TextWrapWord
	payloadSelect := widget.NewSelect(payloadSelectOptions, func(displayName string) {
		option, ok := payloadByDisplayName[displayName]
		if !ok {
			downloadDetails.ParseMarkdown("Select a payload to see where to download it.")
			return
		}
		content := fmt.Sprintf("Opens `%s`", option.link)
		if option.markdown != "" {
			content += "\n\n" + option.markdown
		}
		downloadDetails.ParseMarkdown(content)
	})
	payloadSelect.SetSelected("select payload")
	openPayloadLink := widget.NewButton("Open download page", func() {
		option, ok := payloadByDisplayName[payloadSelect.Selected]
		if !ok {
			return
		}
		showError(openExternalURL(option.link))
	})
	openPayloadLink.Importance = widget.HighImportance

	helpMarkdown := widget.NewRichTextFromMarkdown(payloadHelpMarkdown())
	helpMarkdown.Wrapping = fyne.TextWrapWord
	helpMarkdown.Hide()
	showHelp := false
	helpButton := widget.NewButton("Show help", nil)
	helpButton.OnTapped = func() {
		showHelp = !showHelp
		if showHelp {
			helpMarkdown.Show()
			helpButton.SetText("Hide help")
		} else {
			helpMarkdown.Hide()
			helpButton.SetText("Show help")
		}
	}

	copyIn := func() {
		chooseNativeFile("Choose payload", []nativeFileFilter{
			extensionFilter("Payload binaries", ".bin"),
		}, func(path string) {
			if !strings.HasSuffix(strings.ToLower(path), ".bin") {
				showError(fmt.Errorf("payload files must end in .bin"))
				return
			}
			target := filepath.Join(state.PayloadDirectory, filepath.Base(path))
			target = uniquePayloadTarget(target)
			runAsync(nil, func() error {
				input, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				return os.WriteFile(target, input, 0o644)
			}, reloadPaths)
		})
	}

	top := container.NewVBox(
		widget.NewRichTextFromMarkdown("Choose a payload to inject to a Switch in RCM mode."),
		container.NewHBox(
			widget.NewButton("Copy payload in", copyIn),
			widget.NewButton("Open Payload Folder", func() {
				showError(openPath(state.PayloadDirectory))
			}),
			layout.NewSpacer(),
		),
		widget.NewLabelWithStyle("Available Payloads", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	bottom := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewLabel("Go download a payload:"),
			payloadSelect,
			openPayloadLink,
			layout.NewSpacer(),
		),
		downloadDetails,
		output,
		widget.NewSeparator(),
		container.NewHBox(helpButton, layout.NewSpacer()),
		helpMarkdown,
	)

	return &payloadInjectorTab{
		content: container.NewBorder(top, bottom, nil, nil, payloadListPanel),
		reload:  reloadPaths,
	}
}

func (t *payloadInjectorTab) startWatching() {
	t.watchMu.Lock()
	defer t.watchMu.Unlock()

	if t.watchStop != nil {
		return
	}
	t.reload()
	t.watchStop = make(chan struct{})
	go watchPayloadDirectory(t.watchStop, func() {
		fyne.Do(t.reload)
	})
}

func (t *payloadInjectorTab) stopWatching() {
	t.watchMu.Lock()
	defer t.watchMu.Unlock()

	if t.watchStop == nil {
		return
	}
	close(t.watchStop)
	t.watchStop = nil
}

func watchPayloadDirectory(stop <-chan struct{}, onChanged func()) {
	lastSnapshot, err := payloadDirectorySnapshot()
	if err != nil {
		lastSnapshot = ""
	}

	ticker := time.NewTicker(payloadDirectoryWatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			nextSnapshot, err := payloadDirectorySnapshot()
			if err != nil {
				continue
			}
			if nextSnapshot == lastSnapshot {
				continue
			}
			lastSnapshot = nextSnapshot
			onChanged()
		}
	}
}

func payloadDirectorySnapshot() (string, error) {
	if err := os.MkdirAll(state.PayloadDirectory, 0o755); err != nil {
		return "", err
	}

	files, err := os.ReadDir(state.PayloadDirectory)
	if err != nil {
		return "", err
	}

	var snapshot strings.Builder
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".bin") {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		fmt.Fprintf(&snapshot, "%s\x00%d\x00%d\n", file.Name(), info.Size(), info.ModTime().UnixNano())
	}

	return snapshot.String(), nil
}

func newPayloadListRow(payloadPath string, index int, output *payloadOutput) fyne.CanvasObject {
	return newActionListRow(filepath.Base(payloadPath), index, theme.FileIcon(), "Inject", false, widget.HighImportance, func() {
		basename := filepath.Base(payloadPath)
		output.Show()
		output.SetText(fmt.Sprintf("Injecting %s...\n", basename))
		runAsync(nil, func() error {
			return inject.Inject(payloadPath)
		}, func() {
			output.SetText(output.Text + "Payload injection complete.\n")
		})
	})
}

func payloadHelpMarkdown() string {
	switch runtime.GOOS {
	case "windows":
		return rcmHelpWindowsMarkdown
	case "linux":
		return rcmHelpLinuxMarkdown
	default:
		return rcmHelpGenericMarkdown
	}
}

func uniquePayloadTarget(target string) string {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target
	}
	ext := filepath.Ext(target)
	base := strings.TrimSuffix(target, ext)
	for i := 1; ; i++ {
		next := fmt.Sprintf("%s.copy_%d%s", base, i, ext)
		if _, err := os.Stat(next); os.IsNotExist(err) {
			return next
		}
	}
}
