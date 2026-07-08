package gui

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/hacbrewpack"
)

const (
	forwarderApplicationMode = "Application"
	forwarderRetroArchMode   = "RetroArch ROM"
	forwarderPreviewSize     = 260
)

var (
	forwarderPreviewBackgroundColor = color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xff}
	forwarderPreviewBorderColor     = color.NRGBA{R: 0x3e, G: 0x42, B: 0x4a, A: 0xff}
)

func NroForwarderTab() fyne.CanvasObject {
	selectedMode := forwarderApplicationMode
	id := widget.NewEntry()
	id.SetPlaceHolder("01..........0000")
	if generated, err := generateTitleID(); err == nil {
		id.SetText(generated)
	}
	title := widget.NewEntry()
	title.SetPlaceHolder("NX Shell")
	publisher := widget.NewEntry()
	publisher.SetPlaceHolder("joel16")
	nroPath := widget.NewEntry()
	nroPath.SetPlaceHolder("/switch/NX-Shell.nro")
	romPath := widget.NewEntry()
	romPath.SetPlaceHolder("/roms/nes/Kirby's Adventure.zip")
	imagePath := widget.NewEntry()
	imagePath.Disable()

	var chooseForwarderImage func()
	emptyPreview := widget.NewLabelWithStyle("Please select an NRO file or\nan image", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	imagePreview := canvas.NewImageFromFile("")
	imagePreview.FillMode = canvas.ImageFillContain
	imagePreview.SetMinSize(fyne.NewSquareSize(forwarderPreviewSize))
	imagePreview.Hide()
	previewContent := newForwarderPreviewButton(container.NewStack(container.NewCenter(emptyPreview), imagePreview), func() {
		if chooseForwarderImage != nil {
			chooseForwarderImage()
		}
	})
	preview := container.NewCenter(container.NewGridWrap(
		fyne.NewSquareSize(forwarderPreviewSize),
		newForwarderPreviewPanel(previewContent),
	))

	status := widget.NewMultiLineEntry()
	status.Disable()
	status.Hide()

	romPathLabel := widget.NewLabel("ROM Path:")
	romPathLabel.Hide()
	romPath.Hide()

	var clearImage *widget.Button
	var applicationMode *widget.Button
	var retroArchMode *widget.Button
	var imageCleanup func()

	setMode := func(value string) {
		selectedMode = value
		if value == forwarderRetroArchMode {
			title.SetPlaceHolder("Kirby's Adventure")
			publisher.SetPlaceHolder("Nintendo")
			nroPath.SetPlaceHolder("/retroarch/cores/nestopia_libretro_libnx.nro")
			romPathLabel.Show()
			romPath.Show()
		} else {
			title.SetPlaceHolder("NX Shell")
			publisher.SetPlaceHolder("joel16")
			nroPath.SetPlaceHolder("/switch/NX-Shell.nro")
			romPathLabel.Hide()
			romPath.Hide()
		}
		if applicationMode != nil && retroArchMode != nil {
			applicationMode.Importance = widget.MediumImportance
			retroArchMode.Importance = widget.MediumImportance
			if value == forwarderRetroArchMode {
				retroArchMode.Importance = widget.HighImportance
			} else {
				applicationMode.Importance = widget.HighImportance
			}
			applicationMode.Refresh()
			retroArchMode.Refresh()
		}
	}
	applicationMode = widget.NewButton(forwarderApplicationMode, func() {
		setMode(forwarderApplicationMode)
	})
	retroArchMode = widget.NewButton(forwarderRetroArchMode, func() {
		setMode(forwarderRetroArchMode)
	})
	modeSelector := container.NewGridWithColumns(2, applicationMode, retroArchMode)

	clearForwarderImage := func() {
		if imageCleanup != nil {
			imageCleanup()
			imageCleanup = nil
		}
		imagePath.SetText("")
		imagePreview.File = ""
		imagePreview.Hide()
		emptyPreview.Show()
		imagePreview.Refresh()
		clearImage.Disable()
	}

	setForwarderImage := func(path string, cleanup func()) {
		if imageCleanup != nil {
			imageCleanup()
		}
		imageCleanup = cleanup
		imagePath.SetText(path)
		imagePreview.File = path
		emptyPreview.Hide()
		imagePreview.Show()
		imagePreview.Refresh()
		clearImage.Enable()
	}

	chooseForwarderImage = func() {
		chooseNativeFile("Choose NRO or image", []nativeFileFilter{
			extensionFilter("NRO files and images", ".nro", ".jpg", ".jpeg", ".png"),
			extensionFilter("NRO files", ".nro"),
			extensionFilter("Images", ".jpg", ".jpeg", ".png"),
		}, func(path string) {
			if strings.EqualFold(filepath.Ext(path), ".nro") {
				iconPath, cleanup, err := extractNROIcon(path)
				if err != nil {
					showError(err)
					return
				}
				setForwarderImage(iconPath, cleanup)
				return
			}
			setForwarderImage(path, nil)
		})
	}
	chooseImage := widget.NewButton("Choose image or NRO", chooseForwarderImage)
	clearImage = widget.NewButton("Clear image", clearForwarderImage)
	clearImage.Importance = widget.DangerImportance
	clearImage.Disable()

	regenerate := widget.NewButton("Regenerate ID", func() {
		generated, err := generateTitleID()
		if err != nil {
			showError(err)
			return
		}
		id.SetText(generated)
	})
	regenerate.SetIcon(theme.ViewRefreshIcon())

	build := func(outPath string) error {
		if state.Keys == nil || state.KeysPath == "" {
			return fmt.Errorf("prod.keys are required; configure them in Settings")
		}
		if imagePath.Text == "" {
			return fmt.Errorf("an NSP image is required")
		}
		titleID, err := strconv.ParseUint(strings.TrimSpace(id.Text), 16, 64)
		if err != nil {
			return fmt.Errorf("invalid app id: %w", err)
		}
		iconPath, cleanup, err := prepareForwarderIcon(imagePath.Text)
		if err != nil {
			return err
		}
		defer cleanup()

		nro := "sdmc:" + cleanSwitchPath(nroPath.Text)
		var argv []string
		if selectedMode == forwarderRetroArchMode && strings.TrimSpace(romPath.Text) != "" {
			argv = append(argv, "sdmc:"+cleanSwitchPath(romPath.Text))
		}

		return hacbrewpack.BuildForwarderNSP(hacbrewpack.Options{
			OutPath:         outPath,
			KeysPath:        state.KeysPath,
			IconPath:        iconPath,
			TitleID:         titleID,
			TitleName:       strings.TrimSpace(title.Text),
			TitlePublisher:  strings.TrimSpace(publisher.Text),
			NROPath:         nro,
			NROArgv:         argv,
			NoPatchNACPLogo: true,
			RandomPSS:       state.Options.RandomPSS,
		})
	}

	generate := widget.NewButton("Generate NSP", func() {
		filename := strings.TrimSpace(title.Text)
		if filename == "" {
			filename = strings.TrimSpace(id.Text)
		}
		filename = strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(filename) + ".nsp"
		saveNativeFile("Save NSP", filename, []nativeFileFilter{
			extensionFilter("Nintendo Submission Package", ".nsp"),
		}, func(outPath string) {
			status.Show()
			status.SetText("Generating NSP...\n")
			runAsync(nil, func() error {
				return build(outPath)
			}, func() {
				status.SetText(status.Text + "Generated: " + outPath + "\n")
				sendAppNotification("NSP generated", filepath.Base(outPath))
			})
		})
	})
	generate.Importance = widget.HighImportance

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("App ID:"), container.NewBorder(nil, nil, regenerate, nil, id),
		widget.NewLabel("App Title:"), title,
		widget.NewLabel("App Publisher:"), publisher,
		widget.NewLabel("NRO Path:"), nroPath,
		romPathLabel, romPath,
	)

	setMode(selectedMode)

	return container.NewVBox(
		modeSelector,
		container.NewCenter(container.NewVBox(
			preview,
			container.NewCenter(container.NewHBox(chooseImage, clearImage)),
		)),
		form,
		generate,
		status,
	)
}

func newForwarderPreviewPanel(content fyne.CanvasObject) fyne.CanvasObject {
	background := canvas.NewRectangle(forwarderPreviewBackgroundColor)
	border := canvas.NewRectangle(color.NRGBA{A: 0})
	border.StrokeColor = forwarderPreviewBorderColor
	border.StrokeWidth = 1
	return container.NewMax(background, content, border)
}

type forwarderPreviewButton struct {
	widget.BaseWidget
	content fyne.CanvasObject
	tapped  func()
}

func newForwarderPreviewButton(content fyne.CanvasObject, tapped func()) *forwarderPreviewButton {
	button := &forwarderPreviewButton{
		content: content,
		tapped:  tapped,
	}
	button.ExtendBaseWidget(button)
	return button
}

func (b *forwarderPreviewButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.content)
}

func (b *forwarderPreviewButton) Tapped(*fyne.PointEvent) {
	if b.tapped != nil {
		b.tapped()
	}
}

func prepareForwarderIcon(path string) (string, func(), error) {
	input, err := os.Open(path)
	if err != nil {
		return "", func() {}, err
	}
	defer input.Close()
	img, _, err := image.Decode(input)
	if err != nil {
		return "", func() {}, err
	}
	tmp, err := os.CreateTemp("", "nxkit-icon-*.jpg")
	if err != nil {
		return "", func() {}, err
	}
	defer tmp.Close()
	if err := jpeg.Encode(tmp, img, &jpeg.Options{Quality: 90}); err != nil {
		os.Remove(tmp.Name())
		return "", func() {}, err
	}
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}
