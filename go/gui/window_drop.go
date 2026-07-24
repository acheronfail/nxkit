package gui

import (
	"sync"

	"fyne.io/fyne/v2"
)

const (
	windowDropHandlerNAND  = "nand"
	windowDropHandlerTools = "tools"
)

var windowDropHandlers = struct {
	sync.RWMutex
	active   string
	handlers map[string]func(fyne.Position, []fyne.URI)
}{
	handlers: make(map[string]func(fyne.Position, []fyne.URI)),
}

func registerWindowDropHandler(name string, handler func(fyne.Position, []fyne.URI)) {
	windowDropHandlers.Lock()
	defer windowDropHandlers.Unlock()
	if handler == nil {
		delete(windowDropHandlers.handlers, name)
		return
	}
	windowDropHandlers.handlers[name] = handler
}

func setActiveWindowDropHandler(name string) {
	windowDropHandlers.Lock()
	windowDropHandlers.active = name
	windowDropHandlers.Unlock()
}

func dispatchWindowDrop(position fyne.Position, uris []fyne.URI) {
	windowDropHandlers.RLock()
	handler := windowDropHandlers.handlers[windowDropHandlers.active]
	windowDropHandlers.RUnlock()
	if handler != nil {
		handler(position, uris)
	}
}
