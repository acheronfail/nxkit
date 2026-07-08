package gui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/sqweek/dialog"
)

type nativeFileFilter struct {
	name       string
	extensions []string
}

func chooseNativeFile(title string, filters []nativeFileFilter, onChosen func(string)) {
	go func() {
		path, ok, err := nativeOpenFile(title, filters)
		fyne.Do(func() {
			if err != nil {
				showError(err)
				return
			}
			if ok && onChosen != nil {
				onChosen(path)
			}
		})
	}()
}

func chooseNativeDirectory(title string, onChosen func(string)) {
	go func() {
		path, ok, err := nativeOpenDirectory(title)
		fyne.Do(func() {
			if err != nil {
				showError(err)
				return
			}
			if ok && onChosen != nil {
				onChosen(path)
			}
		})
	}()
}

func saveNativeFile(title string, defaultName string, filters []nativeFileFilter, onChosen func(string)) {
	go func() {
		path, ok, err := nativeSaveFile(title, defaultName, filters)
		fyne.Do(func() {
			if err != nil {
				showError(err)
				return
			}
			if ok && onChosen != nil {
				onChosen(path)
			}
		})
	}()
}

func nativeOpenFile(title string, filters []nativeFileFilter) (string, bool, error) {
	builder := nativeFileDialog(title, filters)
	path, err := builder.Load()
	return nativePathResult(path, err)
}

func nativeOpenDirectory(title string) (string, bool, error) {
	path, err := dialog.Directory().Title(title).Browse()
	return nativePathResult(path, err)
}

func nativeSaveFile(title string, defaultName string, filters []nativeFileFilter) (string, bool, error) {
	builder := nativeFileDialog(title, filters)
	if defaultName != "" {
		builder = builder.SetStartFile(defaultName)
	}
	path, err := builder.Save()
	return nativePathResult(path, err)
}

func nativeFileDialog(title string, filters []nativeFileFilter) *dialog.FileBuilder {
	builder := dialog.File().Title(title)
	for _, filter := range filters {
		builder = builder.Filter(filter.name, filter.extensions...)
	}
	return builder
}

func nativePathResult(path string, err error) (string, bool, error) {
	if errors.Is(err, dialog.ErrCancelled) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return path, path != "", nil
}

func extensionFilter(name string, extensions ...string) nativeFileFilter {
	cleaned := make([]string, len(extensions))
	for i, extension := range extensions {
		cleaned[i] = strings.TrimPrefix(extension, ".")
	}
	return nativeFileFilter{name: name, extensions: cleaned}
}
