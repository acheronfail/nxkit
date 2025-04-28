package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/inject"
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

func PayloadInjectorTab() fyne.CanvasObject {
	emptyWidget := widget.NewLabel("No payloads found; please add some to ~/.switch/payloads!")
	emptyWidget.Hide()

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
				fmt.Printf("Error getting payload path: %v\n", err)
				return
			}

			basename := filepath.Base(payloadPath)
			container := o.(*fyne.Container)

			label := container.Objects[0].(*widget.Label)
			label.SetText(basename)

			button := container.Objects[2].(*widget.Button)
			button.OnTapped = func() {
				err := inject.Inject(payloadPath)
				if err != nil {
					dialog.ShowError(err, mainWindow)
					// nxkitApp.SendNotification(fyne.NewNotification("Error", fmt.Sprintf("Failed to inject payload: %v", err)))
				}
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

	nxkitApp.Lifecycle().SetOnEnteredForeground(reloadPaths)

	tab := container.NewVBox(
		widget.NewButton("Refresh", reloadPaths),
		listContainer,
		emptyWidget,
	)

	return tab
}
