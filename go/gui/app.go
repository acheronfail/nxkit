package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var (
	nxkitApp   fyne.App
	mainWindow fyne.Window
)

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

	tabs := container.NewAppTabs(
		container.NewTabItem("Payload Injector", PayloadInjectorTab()),
		container.NewTabItem("NRO Forwarder", widget.NewLabel("TODO")),
		container.NewTabItem("NAND Explorer", container.NewVBox(
			widget.NewLabelWithData(str),
			widget.NewEntryWithData(str),
			widget.NewButton("Quit", func() {
				nxkitApp.Quit()
			}),
		)),
		container.NewTabItem("Tools", ToolsTab()),
		container.NewTabItem("Settings", widget.NewLabel("TODO")),
	)

	tabs.OnSelected = func(tab *container.TabItem) {
		fmt.Println(tab)
	}
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.SelectIndex(3)

	mainWindow.SetContent(tabs)

	mainWindow.ShowAndRun()
}
