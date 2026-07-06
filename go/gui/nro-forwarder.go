package gui

import (
	"fmt"
	"image"
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
	"fyne.io/fyne/v2/widget"
	"github.com/acheronfail/nxkit/lib/hacbrewpack"
)

func NroForwarderTab() fyne.CanvasObject {
	mode := widget.NewSelect([]string{"Application", "RetroArch ROM"}, nil)
	mode.SetSelected("Application")

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
	imagePreview := canvas.NewImageFromFile("")
	imagePreview.FillMode = canvas.ImageFillContain
	imagePreview.SetMinSize(fyne.NewSize(180, 180))
	imagePreviewContainer := container.NewVBox(widget.NewLabel("Image preview"), imagePreview)
	imagePreviewContainer.Hide()
	status := widget.NewMultiLineEntry()
	status.Disable()
	status.Hide()
	romPathRow := container.NewBorder(nil, nil, widget.NewLabel("ROM Path"), nil, romPath)
	romPathRow.Hide()

	mode.OnChanged = func(value string) {
		if value == "RetroArch ROM" {
			title.SetPlaceHolder("Kirby's Adventure")
			publisher.SetPlaceHolder("Nintendo")
			nroPath.SetPlaceHolder("/retroarch/cores/nestopia_libretro_libnx.nro")
			romPathRow.Show()
		} else {
			title.SetPlaceHolder("NX Shell")
			publisher.SetPlaceHolder("joel16")
			nroPath.SetPlaceHolder("/switch/NX-Shell.nro")
			romPathRow.Hide()
		}
	}

	chooseImage := widget.NewButton("Choose image", func() {
		chooseNativeFile("Choose NSP image", []nativeFileFilter{
			extensionFilter("Images", ".jpg", ".jpeg", ".png"),
		}, func(path string) {
			imagePath.SetText(path)
			imagePreview.File = path
			imagePreview.Refresh()
			imagePreviewContainer.Show()
		})
	})

	regenerate := widget.NewButton("Regenerate ID", func() {
		generated, err := generateTitleID()
		if err != nil {
			showError(err)
			return
		}
		id.SetText(generated)
	})

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
		if mode.Selected == "RetroArch ROM" && strings.TrimSpace(romPath.Text) != "" {
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

	form := container.NewVBox(
		widget.NewRichTextFromMarkdown("Create a Nintendo Switch NSP that forwards to an NRO on the SD card."),
		container.NewGridWithColumns(2,
			widget.NewLabel("Mode"), mode,
			widget.NewLabel("App ID"), container.NewBorder(nil, nil, nil, regenerate, id),
			widget.NewLabel("Title"), title,
			widget.NewLabel("Publisher"), publisher,
			widget.NewLabel("NRO Path"), nroPath,
		),
		romPathRow,
		container.NewBorder(nil, nil, widget.NewLabel("Image"), chooseImage, imagePath),
		imagePreviewContainer,
		container.NewHBox(layout.NewSpacer(), generate),
		status,
	)

	return form
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
