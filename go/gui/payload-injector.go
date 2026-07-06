package gui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/inject"
)

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
	emptyWidget := widget.NewLabel("No payloads found.")
	emptyWidget.Hide()
	output := widget.NewMultiLineEntry()
	output.SetPlaceHolder("Payload injection log")
	output.Disable()
	output.Hide()

	payloads := binding.NewStringList()
	payloadList := widget.NewListWithData(payloads,
		func() fyne.CanvasObject {
			return container.New(layout.NewHBoxLayout(),
				widget.NewLabel("<payload>"),
				layout.NewSpacer(),
				widget.NewButton("Inject", func() {}),
			)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			payloadPath, err := i.(binding.String).Get()
			if err != nil {
				showError(err)
				return
			}

			basename := filepath.Base(payloadPath)
			container := o.(*fyne.Container)

			label := container.Objects[0].(*widget.Label)
			label.SetText(basename)

			button := container.Objects[2].(*widget.Button)
			button.OnTapped = func() {
				output.Show()
				output.SetText(fmt.Sprintf("Injecting %s...\n", basename))
				runAsync(nil, func() error {
					return inject.Inject(payloadPath)
				}, func() {
					output.SetText(output.Text + "Payload injection complete.\n")
				})
			}
		},
	)

	// prevent selections
	payloadList.OnSelected = func(_ widget.ListItemID) {
		payloadList.UnselectAll()
	}

	listContainer := container.New(layout.NewVBoxLayout(),
		widget.NewLabel("Available Payloads:"),
		payloadList,
	)

	reloadPaths := func() {
		payloadPaths, err := getPayloads()
		if err != nil {
			showError(err)
		} else {
			payloads.Set(payloadPaths)
		}

		if len(payloadPaths) > 0 {
			emptyWidget.Hide()
			payloadList.Show()
		} else {
			emptyWidget.Show()
			payloadList.Hide()
		}
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

	tab := container.NewVBox(
		widget.NewRichTextFromMarkdown("Choose a payload to inject to a Switch in RCM mode."),
		container.NewHBox(
			widget.NewButton("Refresh", reloadPaths),
			widget.NewButton("Copy payload in", copyIn),
			widget.NewButton("Open Payload Folder", func() {
				showError(openPath(state.PayloadDirectory))
			}),
			layout.NewSpacer(),
		),
		listContainer,
		emptyWidget,
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

	return tab
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
