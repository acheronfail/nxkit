package gui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/tools"
)

func TestToolsTabChooseAndCancelButtonsToggleDuringOperations(t *testing.T) {
	test.NewTempApp(t)
	oldState := state
	state = &appState{Keys: &keys.Keys{}}
	t.Cleanup(func() {
		state = oldState
	})

	tests := []struct {
		name        string
		chooseText  string
		chooseTitle string
		path        string
		configure   func(*toolsTabDependencies, *blockingToolsOperation)
	}{
		{
			name:        "split",
			chooseText:  "Choose file to split",
			chooseTitle: "Choose file to split",
			path:        filepath.Join(t.TempDir(), "rawnand.bin"),
			configure: func(deps *toolsTabDependencies, op *blockingToolsOperation) {
				deps.split = func(ctx context.Context, _ string, _, _ bool, _ int64, _ tools.ProgressFunc) (string, error) {
					return op.run(ctx)
				}
			},
		},
		{
			name:        "merge",
			chooseText:  "Choose file to merge",
			chooseTitle: "Choose file to merge",
			path:        filepath.Join(t.TempDir(), "rawnand.bin.00"),
			configure: func(deps *toolsTabDependencies, op *blockingToolsOperation) {
				deps.merge = func(ctx context.Context, _ string, _ bool, _ tools.ProgressFunc) (string, error) {
					return op.run(ctx)
				}
			},
		},
		{
			name:        "compress",
			chooseText:  "Choose NSP to compress",
			chooseTitle: "Choose NSP to compress",
			path:        filepath.Join(t.TempDir(), "game.nsp"),
			configure: func(deps *toolsTabDependencies, op *blockingToolsOperation) {
				deps.compress = func(ctx context.Context, _ string, _ keys.Keys, _ tools.ProgressFunc) (string, error) {
					return op.run(ctx)
				}
			},
		},
		{
			name:        "decompress",
			chooseText:  "Choose NSZ to decompress",
			chooseTitle: "Choose NSZ to decompress",
			path:        filepath.Join(t.TempDir(), "game.nsz"),
			configure: func(deps *toolsTabDependencies, op *blockingToolsOperation) {
				deps.decompress = func(ctx context.Context, _ string, _ tools.ProgressFunc) (string, error) {
					return op.run(ctx)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(tt.path, nil, 0o644); err != nil {
				t.Fatal(err)
			}

			op := newBlockingToolsOperation()
			deps := inactiveToolsTabDependencies(t)
			deps.chooseFile = func(title string, _ []nativeFileFilter, onChosen func(string)) {
				if title != tt.chooseTitle {
					t.Errorf("choose file title = %q, want %q", title, tt.chooseTitle)
				}
				onChosen(tt.path)
			}
			tt.configure(&deps, op)

			tab := newToolsTab(deps)
			chooseButton := findButtonByText(t, tab, tt.chooseText)

			if !chooseButton.Visible() {
				t.Fatalf("%q hidden before operation starts", tt.chooseText)
			}
			if got := visibleButtonCountByText(tab, "Cancel"); got != 0 {
				t.Fatalf("visible cancel buttons before operation = %d, want 0", got)
			}

			test.Tap(chooseButton)
			waitForClosed(t, op.started, "operation to start")

			if chooseButton.Visible() {
				t.Fatalf("%q visible while operation is running", tt.chooseText)
			}
			cancelButton := findVisibleButtonByText(t, tab, "Cancel")
			test.Tap(cancelButton)
			waitForClosed(t, op.canceled, "operation to be canceled")

			waitUntil(t, func() bool {
				fyne.DoAndWait(func() {})
				return chooseButton.Visible() && visibleButtonCountByText(tab, "Cancel") == 0
			}, "choose button to return and cancel button to hide")
		})
	}
}

func TestToolActionSlotKeepsButtonPlacementStableWhenButtonsAreHidden(t *testing.T) {
	test.NewTempApp(t)

	choose := widget.NewButton("Choose a much longer file name", nil)
	cancel := widget.NewButton("Cancel", nil)
	slot := newToolActionSlot(choose, cancel)
	fullSize := slot.MinSize()

	choose.Hide()
	if got := slot.MinSize(); got != fullSize {
		t.Fatalf("slot minimum size with choose hidden = %v, want %v", got, fullSize)
	}

	choose.Show()
	cancel.Hide()
	if got := slot.MinSize(); got != fullSize {
		t.Fatalf("slot minimum size with cancel hidden = %v, want %v", got, fullSize)
	}
}

type blockingToolsOperation struct {
	started  chan struct{}
	canceled chan struct{}
}

func newBlockingToolsOperation() *blockingToolsOperation {
	return &blockingToolsOperation{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (o *blockingToolsOperation) run(ctx context.Context) (string, error) {
	close(o.started)
	<-ctx.Done()
	close(o.canceled)
	return "", ctx.Err()
}

func inactiveToolsTabDependencies(t *testing.T) toolsTabDependencies {
	t.Helper()

	inactive := func() (string, error) {
		t.Helper()
		t.Fatal("unexpected tool operation")
		return "", errors.New("unexpected tool operation")
	}

	return toolsTabDependencies{
		chooseFile: func(string, []nativeFileFilter, func(string)) {
			t.Fatal("unexpected file chooser")
		},
		split: func(context.Context, string, bool, bool, int64, tools.ProgressFunc) (string, error) {
			return inactive()
		},
		merge: func(context.Context, string, bool, tools.ProgressFunc) (string, error) {
			return inactive()
		},
		compress: func(context.Context, string, keys.Keys, tools.ProgressFunc) (string, error) {
			return inactive()
		},
		decompress: func(context.Context, string, tools.ProgressFunc) (string, error) {
			return inactive()
		},
	}
}

func findButtonByText(t *testing.T, object fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	for _, button := range findButtonsByText(object, text) {
		return button
	}
	t.Fatalf("button %q not found", text)
	return nil
}

func findVisibleButtonByText(t *testing.T, object fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	for _, button := range findButtonsByText(object, text) {
		if button.Visible() {
			return button
		}
	}
	t.Fatalf("visible button %q not found", text)
	return nil
}

func visibleButtonCountByText(object fyne.CanvasObject, text string) int {
	count := 0
	for _, button := range findButtonsByText(object, text) {
		if button.Visible() {
			count++
		}
	}
	return count
}

func findButtonsByText(object fyne.CanvasObject, text string) []*widget.Button {
	var buttons []*widget.Button
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		buttons = append(buttons, button)
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			buttons = append(buttons, findButtonsByText(child, text)...)
		}
	}
	return buttons
}

func waitForClosed(t *testing.T, done <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func waitUntil(t *testing.T, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}
