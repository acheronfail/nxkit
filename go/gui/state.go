package gui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/acheronfail/nxkit/lib/keys"
)

type Options struct {
	RandomPSS bool
}

type appState struct {
	Options             Options
	PayloadDirectory    string
	ProdKeysSearchPaths []string
	KeysPath            string
	Keys                *keys.Keys
	ShowAdvanced        bool
	advancedListeners   []func(bool)
}

var (
	packagedBuild  = "false"
	packageVersion = "0.0.0"
)

func newAppState(options Options) (*appState, error) {
	payloadDir, err := payloadDirectory()
	if err != nil {
		return nil, err
	}
	state := &appState{
		Options:             options,
		PayloadDirectory:    payloadDir,
		ProdKeysSearchPaths: prodKeysSearchPaths(),
	}
	state.LoadDefaultKeys()
	return state, nil
}

func payloadDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".switch", "payloads")
	return path, os.MkdirAll(path, 0o755)
}

func prodKeysSearchPaths() []string {
	home, _ := os.UserHomeDir()
	var paths []string
	if home != "" {
		paths = append(paths, filepath.Join(home, ".switch", "prod.keys"))
	}
	if isDevMode() {
		if wd, err := os.Getwd(); err == nil {
			paths = append(paths, filepath.Join(wd, "prod.keys"))
		}
		if exe, err := os.Executable(); err == nil {
			paths = append(paths, filepath.Join(filepath.Dir(exe), "prod.keys"))
		}
	}
	return uniqueStrings(paths)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	var out []string
	for _, value := range values {
		clean := filepath.Clean(value)
		if !seen[clean] {
			seen[clean] = true
			out = append(out, clean)
		}
	}
	return out
}

func (s *appState) LoadDefaultKeys() {
	for _, path := range s.ProdKeysSearchPaths {
		if err := s.SetKeysPath(path); err == nil {
			return
		}
	}
}

func (s *appState) SetKeysPath(path string) error {
	parsed, err := keys.NewFromPath(path)
	if err != nil {
		return err
	}
	s.KeysPath = path
	s.Keys = parsed
	return nil
}

func (s *appState) ClearKeys() {
	s.KeysPath = ""
	s.Keys = nil
}

func (s *appState) SetShowAdvanced(show bool) {
	if s.ShowAdvanced == show {
		return
	}
	s.ShowAdvanced = show
	for _, listener := range s.advancedListeners {
		listener(show)
	}
}

func (s *appState) OnAdvancedSettingsChanged(listener func(bool)) {
	s.advancedListeners = append(s.advancedListeners, listener)
}

func showError(err error) {
	if err != nil {
		dialog.ShowError(err, mainWindow)
	}
}

func runAsync(setLoading func(bool), work func() error, done func()) {
	if setLoading != nil {
		setLoading(true)
	}
	go func() {
		err := work()
		fyne.Do(func() {
			if setLoading != nil {
				setLoading(false)
			}
			if err != nil {
				showError(err)
				return
			}
			if done != nil {
				done()
			}
		})
	}()
}

func openPath(path string) error {
	u := url.URL{Scheme: "file", Path: path}
	return nxkitApp.OpenURL(&u)
}

func revealPath(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	default:
		stat, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			path = filepath.Dir(path)
		}
		return openPath(path)
	}
}

func openExternalURL(link string) error {
	u, err := url.Parse(link)
	if err != nil {
		return err
	}
	return nxkitApp.OpenURL(u)
}

func sendAppNotification(title string, content string) {
	if runtime.GOOS == "darwin" && !isMacAppBundle() {
		return
	}
	nxkitApp.SendNotification(fyne.NewNotification(title, content))
}

func isMacAppBundle() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe = filepath.Clean(exe)
	return strings.Contains(exe, ".app"+string(filepath.Separator)+"Contents"+string(filepath.Separator)+"MacOS")
}

func isDevMode() bool {
	return packagedBuild != "true" && !isMacAppBundle()
}

func generateTitleID() (string, error) {
	randomBytes := make([]byte, 5)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return "01" + hex.EncodeToString(randomBytes) + "0000", nil
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func cleanSwitchPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	path = strings.TrimPrefix(path, "sdmc:")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
