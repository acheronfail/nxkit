//go:build darwin

package tools

import "golang.org/x/sys/unix"

func platformSetArchiveBit(path string) error {
	var stat unix.Stat_t
	if err := unix.Stat(path, &stat); err != nil {
		return err
	}

	// SF_ARCHIVED means "has been archived", which is the inverse of the
	// DOS/FAT archive bit ("needs to be archived"). Clearing it therefore sets
	// the FAT archive bit when path is on a FAT volume.
	return unix.Chflags(path, int(stat.Flags&^unix.SF_ARCHIVED))
}
