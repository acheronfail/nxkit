package gui

import (
	"fmt"

	"github.com/acheronfail/nxkit/resources"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
)

const tabContentPadding = 12

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
	payloadInjectorItem := newScrollableTabItem("Payload Injector", payloadInjector.content)
	nandExplorerItem := newScrollableTabItem("NAND Explorer", NandExplorerTab())
	toolsItem := newScrollableTabItem("Tools", ToolsTab())
	tabs := container.NewAppTabs(
		payloadInjectorItem,
		newScrollableTabItem("NRO Forwarder", NroForwarderTab()),
		nandExplorerItem,
		toolsItem,
		newScrollableTabItem("Settings", SettingsTab()),
	)

	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(item *container.TabItem) {
		switch item {
		case nandExplorerItem:
			setActiveWindowDropHandler(windowDropHandlerNAND)
		case toolsItem:
			setActiveWindowDropHandler(windowDropHandlerTools)
		default:
			setActiveWindowDropHandler("")
		}
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
	mainWindow.SetOnDropped(dispatchWindowDrop)
	payloadInjector.startWatching()

	mainWindow.ShowAndRun()
}

func newScrollableTabItem(text string, content fyne.CanvasObject) *container.TabItem {
	paddedContent := container.New(
		layout.NewCustomPaddedLayout(tabContentPadding, tabContentPadding, tabContentPadding, tabContentPadding),
		content,
	)
	return container.NewTabItem(text, container.NewVScroll(paddedContent))
}
