//go:build windows

package tools

import "golang.org/x/sys/windows"

func platformSetArchiveBit(path string) error {
	pathUTF16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attributes, err := windows.GetFileAttributes(pathUTF16)
	if err != nil {
		return err
	}
	return windows.SetFileAttributes(pathUTF16, attributes|windows.FILE_ATTRIBUTE_ARCHIVE)
}
