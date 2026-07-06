package gui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"github.com/ncruces/zenity"
)

type nativeFileFilter = zenity.FileFilter

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
	opts := nativeFileOptions(title, filters)
	path, err := zenity.SelectFile(opts...)
	return nativePathResult(path, err)
}

func nativeOpenDirectory(title string) (string, bool, error) {
	opts := nativeFileOptions(title, nil)
	opts = append(opts, zenity.Directory())
	path, err := zenity.SelectFile(opts...)
	return nativePathResult(path, err)
}

func nativeSaveFile(title string, defaultName string, filters []nativeFileFilter) (string, bool, error) {
	opts := nativeFileOptions(title, filters)
	if defaultName != "" {
		opts = append(opts, zenity.Filename(defaultName))
	}
	opts = append(opts, zenity.ConfirmOverwrite())
	path, err := zenity.SelectFileSave(opts...)
	return nativePathResult(path, err)
}

func nativeFileOptions(title string, filters []nativeFileFilter) []zenity.Option {
	opts := []zenity.Option{
		zenity.Title(title),
		zenity.Modal(),
	}
	if icon, ok := nativeDialogWindowIcon(); ok {
		opts = append(opts, zenity.WindowIcon(icon))
	}
	if attach, ok := nativeDialogAttachOption(); ok {
		opts = append(opts, attach)
	}
	for _, filter := range filters {
		opts = append(opts, filter)
	}
	return opts
}

func nativeDialogWindowIcon() (string, bool) {
	if runtime.GOOS != "darwin" {
		return "", false
	}

	candidates := []string{
		filepath.Join("go", "resources", "nxkit-icon-subtle.png"),
		filepath.Join("resources", "nxkit-icon-subtle.png"),
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append([]string{
			filepath.Join(exeDir, "..", "Resources", "nxkit-icon-subtle.png"),
			filepath.Join(exeDir, "..", "Resources", "nxkit-icon-subtle-256.png"),
		}, candidates...)
	}

	for _, candidate := range candidates {
		path, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func nativeDialogAttachOption() (zenity.Option, bool) {
	if runtime.GOOS == "darwin" {
		return nil, false
	}

	nativeWindow, ok := mainWindow.(driver.NativeWindow)
	if !ok {
		return nil, false
	}

	var attach any
	nativeWindow.RunNative(func(context any) {
		switch ctx := context.(type) {
		case driver.WindowsWindowContext:
			if ctx.HWND != 0 {
				attach = ctx.HWND
			}
		case driver.X11WindowContext:
			if ctx.WindowHandle != 0 {
				attach = int(ctx.WindowHandle)
			}
		}
	})
	if attach == nil {
		return nil, false
	}

	return zenity.Attach(attach), true
}

func nativePathResult(path string, err error) (string, bool, error) {
	if errors.Is(err, zenity.ErrCanceled) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return path, path != "", nil
}

func extensionFilter(name string, extensions ...string) nativeFileFilter {
	patterns := make([]string, len(extensions))
	for i, extension := range extensions {
		patterns[i] = "*" + extension
	}
	return nativeFileFilter{Name: name, Patterns: patterns, CaseFold: true}
}
