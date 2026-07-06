package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func SettingsTab() fyne.CanvasObject {
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapBreak
	searchPaths := widget.NewRichTextFromMarkdown("By default NXKit searches:\n\n- `" + strings.Join(state.ProdKeysSearchPaths, "`\n- `") + "`")
	searchPaths.Wrapping = fyne.TextWrapBreak
	description := widget.NewRichTextFromMarkdown("Prod keys are required for creating NSPs with the NRO Forwarder and for reading encrypted NAND partitions.")
	description.Wrapping = fyne.TextWrapWord
	advanced := widget.NewCheck("Show Advanced Settings", func(checked bool) {
		state.ShowAdvanced = checked
	})
	advanced.SetChecked(state.ShowAdvanced)

	var clearKeys *widget.Button
	refreshStatus := func() {
		noKeysConfigured := state.KeysPath == ""
		if noKeysConfigured {
			status.SetText("No prod.keys configured.")
		} else {
			status.SetText(fmt.Sprintf("Configured keys: %s", state.KeysPath))
		}
		if clearKeys != nil {
			if noKeysConfigured {
				clearKeys.Disable()
			} else {
				clearKeys.Enable()
			}
		}
	}
	refreshStatus()

	chooseKeys := widget.NewButton("Manually select keys", func() {
		chooseNativeFile("Choose prod.keys", []nativeFileFilter{
			{Name: "prod.keys", Patterns: []string{"prod.keys", "*.keys"}, CaseFold: true},
		}, func(path string) {
			if err := state.SetKeysPath(path); err != nil {
				showError(err)
				return
			}
			refreshStatus()
		})
	})

	clearKeys = widget.NewButton("Clear selected keys", func() {
		state.ClearKeys()
		refreshStatus()
	})
	clearKeys.Importance = widget.DangerImportance
	refreshStatus()

	reloadKeys := widget.NewButton("Search default paths", func() {
		state.ClearKeys()
		state.LoadDefaultKeys()
		refreshStatus()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Select Prod Keys", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		status,
		container.NewHBox(chooseKeys, clearKeys, reloadKeys, layout.NewSpacer()),
		description,
		searchPaths,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Other toggles", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		advanced,
	)
}
