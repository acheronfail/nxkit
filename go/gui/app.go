package gui

import (
	"fmt"

	"github.com/acheronfail/nxkit/resources"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
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
	nxkitApp.SetIcon(resources.NXKitIcon)
	nxkitApp.Settings().SetTheme(newNXKitTheme())
	mainWindow = nxkitApp.NewWindow("NXKit")
	mainWindow.SetIcon(resources.NXKitIcon)
	mainWindow.Resize(fyne.NewSize(960, 720))

	shortcutQuit := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	mainWindow.Canvas().AddShortcut(shortcutQuit, func(shortcut fyne.Shortcut) {
		nxkitApp.Quit()
	})

	payloadInjector := newPayloadInjectorTab()
	payloadInjectorItem := container.NewTabItem("Payload Injector", payloadInjector.content)
	tabs := container.NewAppTabs(
		payloadInjectorItem,
		container.NewTabItem("NRO Forwarder", NroForwarderTab()),
		container.NewTabItem("NAND Explorer", NandExplorerTab()),
		container.NewTabItem("Tools", ToolsTab()),
		container.NewTabItem("Settings", SettingsTab()),
	)

	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(item *container.TabItem) {
		if item == payloadInjectorItem {
			payloadInjector.startWatching()
		}
	}
	tabs.OnUnselected = func(item *container.TabItem) {
		if item == payloadInjectorItem {
			payloadInjector.stopWatching()
		}
	}
	nxkitApp.Lifecycle().SetOnStopped(payloadInjector.stopWatching)

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

	mainWindow.SetContent(tabs)
	payloadInjector.startWatching()

	mainWindow.ShowAndRun()
}
