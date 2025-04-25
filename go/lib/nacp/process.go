package nacp

import (
	"fmt"
	"os"
	"path/filepath"
)

func Process(controlDir string, titleId int64) error {
	nacpPath := filepath.Join(controlDir, "control.nacp")
	nacpFile, err := os.OpenFile(nacpPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer nacpFile.Close()

	// Read the NACP file
	nacpData, err := os.ReadFile(nacpPath)
	if err != nil {
		return err
	}

	nacp := NewNacp(nacpData)

	// TODO: these are required
	if nacp.GetTitle() == "" {
		return fmt.Errorf("invalid title name in control.nacp")
	}
	if nacp.GetAuthor() == "" {
		return fmt.Errorf("invalid publisher/author in control.nacp")
	}

	// TODO: these are optional
	nacp.SetLogoHandling(0)

	if titleId != 0 {
		nacp.SetID(titleId)
	}

	// Write the modified NACP data back to the file
	_, err = nacpFile.Write(nacp.Buffer())
	if err != nil {
		return err
	}

	return nil
}
