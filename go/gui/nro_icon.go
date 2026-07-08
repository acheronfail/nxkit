package gui

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	nroMagicOffset          = 0x10
	nroSizeOffset           = 0x18
	nroAssetHeaderSize      = 0x38
	nroAssetHeaderMagic     = "ASET"
	nroIconSectionOffset    = 0x8
	nroAssetSectionSizeSkip = 0x8
)

func extractNROIcon(path string) (string, func(), error) {
	input, err := os.Open(path)
	if err != nil {
		return "", func() {}, err
	}
	defer input.Close()

	isNRO, err := readNROMagic(input)
	if err != nil {
		return "", func() {}, err
	}
	if !isNRO {
		return "", func() {}, fmt.Errorf("not an NRO file")
	}

	nroSize, err := readUint32At(input, nroSizeOffset)
	if err != nil {
		return "", func() {}, err
	}

	assetHeader := make([]byte, nroAssetHeaderSize)
	if _, err := input.ReadAt(assetHeader, int64(nroSize)); err != nil {
		return "", func() {}, fmt.Errorf("failed to read NRO asset header: %w", err)
	}
	if string(assetHeader[:4]) != nroAssetHeaderMagic {
		return "", func() {}, fmt.Errorf("failed to find NRO asset header")
	}

	iconOffset := binary.LittleEndian.Uint32(assetHeader[nroIconSectionOffset:])
	iconSize := binary.LittleEndian.Uint32(assetHeader[nroIconSectionOffset+nroAssetSectionSizeSkip:])
	if iconSize == 0 {
		return "", func() {}, fmt.Errorf("NRO does not contain an icon")
	}

	info, err := input.Stat()
	if err != nil {
		return "", func() {}, err
	}
	iconStart := uint64(nroSize) + uint64(iconOffset)
	iconEnd := iconStart + uint64(iconSize)
	if iconEnd > uint64(info.Size()) {
		return "", func() {}, fmt.Errorf("NRO icon section extends past end of file")
	}

	output, err := os.CreateTemp("", "nxkit-nro-icon-*.jpg")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.Remove(output.Name()) }
	defer output.Close()

	if _, err := io.Copy(output, io.NewSectionReader(input, int64(iconStart), int64(iconSize))); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return output.Name(), cleanup, nil
}

func readNROMagic(input io.ReaderAt) (bool, error) {
	magic := make([]byte, 4)
	if _, err := input.ReadAt(magic, nroMagicOffset); err != nil {
		return false, err
	}
	return string(magic) == "NRO0", nil
}

func readUint32At(input io.ReaderAt, offset int64) (uint32, error) {
	buf := make([]byte, 4)
	if _, err := input.ReadAt(buf, offset); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}
