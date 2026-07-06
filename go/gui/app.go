package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	nxkitApp   fyne.App
	mainWindow fyne.Window
	state      *appState
)

func StartGuiApp(options Options) {
	var err error
	state, err = newAppState(options)
	if err != nil {
		panic(err)
	}

	nxkitApp = app.NewWithID("fail.acheron.nxkit")
	mainWindow = nxkitApp.NewWindow("NXKit")
	mainWindow.Resize(fyne.NewSize(960, 720))

	shortcutQuit := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	mainWindow.Canvas().AddShortcut(shortcutQuit, func(shortcut fyne.Shortcut) {
		nxkitApp.Quit()
	})

	tabs := container.NewAppTabs(
		container.NewTabItem("NRO Forwarder", NroForwarderTab()),
		container.NewTabItem("Payload Injector", PayloadInjectorTab()),
		container.NewTabItem("NAND Explorer", NandExplorerTab()),
		container.NewTabItem("Tools", ToolsTab()),
		container.NewTabItem("Settings", SettingsTab()),
	)

	tabs.SetTabLocation(container.TabLocationTop)

	for i := 1; i <= 5; i++ {
		index := i - 1
		key := fyne.KeyName(fmt.Sprintf("%d", i))
		mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) {
			tabs.SelectIndex(index)
		})
		mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierSuper}, func(fyne.Shortcut) {
			tabs.SelectIndex(index)
		})
	}

	header := widget.NewLabelWithStyle("NXKit", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	header.TextStyle.Monospace = true
	mainWindow.SetContent(container.NewBorder(
		container.NewVBox(layout.NewSpacer(), header),
		nil,
		nil,
		nil,
		tabs,
	))

	mainWindow.ShowAndRun()
}
