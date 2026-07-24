//go:build linux

package tools

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	fatAttributeArchive   = 0x20
	fatIoctlGetAttributes = 0x80047210
	fatIoctlSetAttributes = 0x40047211
)

func platformSetArchiveBit(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()

	attributes, err := unix.IoctlGetInt(int(directory.Fd()), fatIoctlGetAttributes)
	if err != nil {
		return err
	}
	return unix.IoctlSetPointerInt(
		int(directory.Fd()),
		fatIoctlSetAttributes,
		attributes|fatAttributeArchive,
	)
}
