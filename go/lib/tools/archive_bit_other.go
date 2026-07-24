//go:build !darwin && !linux && !windows

package tools

import "fmt"

func platformSetArchiveBit(string) error {
	return fmt.Errorf("archive-bit updates are not supported on this platform")
}
