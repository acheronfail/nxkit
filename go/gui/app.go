package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/inject"
)

var (
	nxkitApp   fyne.App
	mainWindow fyne.Window
)

func getPayloads() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	payloadDir := filepath.Join(homeDir, ".switch", "payloads")
	files, err := os.ReadDir(payloadDir)
	if err != nil {
		return nil, err
	}

	var payloads []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".bin") {
			payloads = append(payloads, filepath.Join(payloadDir, file.Name()))
		}
	}

	return payloads, nil
}

func payloadInjectorTab(payloads binding.StringList) fyne.CanvasObject {
	emptyWidget := widget.NewLabel("No payloads found; please add some to ~/.switch/payloads!")
	emptyWidget.Hide()

	payloadList := widget.NewListWithData(payloads,
		func() fyne.CanvasObject {
			return container.New(layout.NewHBoxLayout(),
				widget.NewLabel("<payload>"),
				widget.NewButton("Inject", func() {}),
			)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			payloadPath, err := i.(binding.String).Get()
			if err != nil {
				fmt.Printf("Error getting payload path: %v\n", err)
				return
			}

			basename := filepath.Base(payloadPath)
			container := o.(*fyne.Container)

			label := container.Objects[0].(*widget.Label)
			label.SetText(basename)

			button := container.Objects[1].(*widget.Button)
			button.OnTapped = func() {
				err := inject.Inject(payloadPath)
				if err != nil {
					dialog.ShowError(err, mainWindow)
					// nxkitApp.SendNotification(fyne.NewNotification("Error", fmt.Sprintf("Failed to inject payload: %v", err)))
				}
			}
		},
	)

	listContainer := container.New(layout.NewVBoxLayout(),
		widget.NewLabel("Available Payloads:"),
		payloadList,
	)

	reloadPaths := func() {
		payloadPaths, err := getPayloads()
		if err != nil {
			fmt.Printf("Error getting payloads: %v\n", err)
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

	tab := container.NewVBox(
		widget.NewButton("Refresh", reloadPaths),
		listContainer,
		emptyWidget,
	)

	return tab
}

func StartGuiApp() {
	nxkitApp = app.New()
	mainWindow = nxkitApp.NewWindow("NXKit")
	mainWindow.Resize(fyne.NewSize(640, 480))

	shortcutQuit := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	mainWindow.Canvas().AddShortcut(shortcutQuit, func(shortcut fyne.Shortcut) {
		nxkitApp.Quit()
	})

	str := binding.NewString()
	str.Set("Hi!")

	payloads := binding.NewStringList()

	tabs := container.NewAppTabs(
		container.NewTabItem("Payload Injector", payloadInjectorTab(payloads)),
		container.NewTabItem("NAND Explorer", widget.NewLabel("TODO")),
		container.NewTabItem("NRO Forwarder", container.NewVBox(
			widget.NewLabelWithData(str),
			widget.NewEntryWithData(str),
			widget.NewButton("Quit", func() {
				nxkitApp.Quit()
			}),
		)),
		container.NewTabItem("Tools", widget.NewLabel("TODO")),
		container.NewTabItem("Settings", widget.NewLabel("TODO")),
	)

	tabs.OnSelected = func(tab *container.TabItem) {
		fmt.Println(tab)
	}
	tabs.SetTabLocation(container.TabLocationTop)

	mainWindow.SetContent(tabs)

	mainWindow.ShowAndRun()
}
