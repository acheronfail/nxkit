package gui

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractNROIcon(t *testing.T) {
	icon := []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43}
	nroPath := writeTestNRO(t, icon)

	iconPath, cleanup, err := extractNROIcon(nroPath)
	if err != nil {
		t.Fatalf("extractNROIcon() error = %v", err)
	}
	defer cleanup()

	got, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", iconPath, err)
	}
	if !bytes.Equal(got, icon) {
		t.Fatalf("extracted icon = %x, want %x", got, icon)
	}
}

func TestExtractNROIconWithoutIcon(t *testing.T) {
	nroPath := writeTestNRO(t, nil)

	if _, _, err := extractNROIcon(nroPath); err == nil {
		t.Fatal("extractNROIcon() error = nil, want error")
	}
}

func TestExtractNROIconRejectsNonNRO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.jpg")
	if err := os.WriteFile(path, []byte{0xff, 0xd8, 0xff, 0xdb}, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := extractNROIcon(path); err == nil {
		t.Fatal("extractNROIcon() error = nil, want error")
	}
}

func writeTestNRO(t *testing.T, icon []byte) string {
	t.Helper()

	const nroSize = 0x40
	data := make([]byte, nroSize+nroAssetHeaderSize+len(icon))
	copy(data[nroMagicOffset:], "NRO0")
	binary.LittleEndian.PutUint32(data[nroSizeOffset:], nroSize)

	assetHeader := data[nroSize : nroSize+nroAssetHeaderSize]
	copy(assetHeader, nroAssetHeaderMagic)
	if len(icon) > 0 {
		binary.LittleEndian.PutUint32(assetHeader[nroIconSectionOffset:], nroAssetHeaderSize)
		binary.LittleEndian.PutUint32(assetHeader[nroIconSectionOffset+nroAssetSectionSizeSkip:], uint32(len(icon)))
		copy(data[nroSize+nroAssetHeaderSize:], icon)
	}

	path := filepath.Join(t.TempDir(), "test.nro")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
