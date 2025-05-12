package internal

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf16"
)

// calculates the number of directory entries required, including the normal entry
func (fs *FileSystem) numDirectoryEntriesRequired(name string) int {
	// start with the normal directory entry
	needed := 1

	// http://elm-chan.org/docs/fat_e.html#lfn_suppression
	sfn, _, ntRes := fs.createShortName(name, []Entry{})

	// if we set NTRes, then we've discovered a name that can be represented
	// with a single entry + the NTRes flag
	if ntRes != 0 {
		return needed
	}

	// if NTRes isn't set, and name is the same, then we only need a single entry
	if sfn == name {
		return needed
	}

	// otherwise we need to create LFN entries for this name
	runes := utf16.Encode([]rune(name))
	lfnCount := (len(runes) + 12) / 13 // round up
	needed += lfnCount

	return needed
}

// TODO: improve this - with tests - to:
//
//	(1) stop scanning when 0x00 is found, and
//	(2) handle issues when near end of `dirBytes`
func (fs *FileSystem) findAvailableDirectoryEntry(dirBytes []byte, numEntries int) (int, bool) {
	count := 0
	for i := 0; i < len(dirBytes); i += FatDirectoryEntrySize {
		// 0x00 == free, 0xE5 == deleted
		if dirBytes[i] == 0x00 || dirBytes[i] == 0xE5 {
			count++
		} else {
			count = 0
		}

		if count == numEntries {
			return i - (numEntries-1)*FatDirectoryEntrySize, true
		}
	}

	return 0, false
}

func readDirectoryEntries(dirBytes []byte) ([]Entry, error) {
	entries := make([]Entry, 0)

	var longFileNameSum *uint8
	longFileName := ""
	for i := 0; i < len(dirBytes); i += FatDirectoryEntrySize {
		if dirBytes[i] == 0x00 {
			break
		}

		if dirBytes[i] == 0xE5 {
			continue
		}

		if dirBytes[i+11] == 0x0F {
			lfn := fatLongFileNameFromBytes(dirBytes[i:])

			// Extract the actual name parts and prepend them (since LFN entries are stored in reverse order)
			namePart := lfn.extractNamePart()

			// this was the first lfn entry, reset lfn
			if lfn.LDIR_Ord&0x40 != 0 {
				longFileName = ""
				longFileNameSum = nil
			}

			// if the checksums didn't match, then discard them
			if longFileNameSum != nil && *longFileNameSum != lfn.LDIR_Chksum {
				continue
			}

			longFileName = namePart + longFileName
			longFileNameSum = &lfn.LDIR_Chksum
			continue
		}

		fatEntry := fatDirectoryEntryFromBytes(dirBytes[i : i+32])
		entry := Entry{fatDirectoryEntry: fatEntry}

		// check lfn checksum before applying it
		if longFileNameSum != nil && calculateShortNameChecksum(fatEntry.DIR_Name[:]) == *longFileNameSum {
			entry.longFileName = longFileName
		}

		// reset lfn
		longFileName = ""
		longFileNameSum = nil

		if !entry.IsVolumeId() {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (fs *FileSystem) findIndexInParentBytes(ent *Entry, parentDirCluster *uint32) (i int, parentDirBytes []byte, err error) {
	if parentDirCluster == nil {
		parentDirBytes, err = fs.getRootDirectoryBytes(fs)
	} else {
		parentDirBytes, err = fs.GetClusterChainBytes(*parentDirCluster)
	}

	if err != nil {
		return -1, nil, err
	}

	for i := 0; i < len(parentDirBytes); i += FatDirectoryEntrySize {
		if parentDirBytes[i] == 0x00 {
			break
		}

		// skip deleted and lfn entries
		if parentDirBytes[i] == 0xE5 || parentDirBytes[i+11] == 0x0F {
			continue
		}

		// check if this is the entry we are looking for
		if bytes.Equal(parentDirBytes[i:i+11], ent.DIR_Name[:]) {
			// check it's not an entry with the same name as the volume id
			if parentDirBytes[i+11]&0x08 == 0x08 {
				continue
			}

			return i, parentDirBytes, nil
		}
	}

	return -1, nil, fmt.Errorf("entry not found in parent directory")
}

func (fs *FileSystem) createNewEntry(
	newEntName string,
	newEntTime *fatTime,
	newEntAttr uint8,
	newEntCluster uint32,
	parentDirEntries []Entry,
) *Entry {
	shortNameBytes, ntRes := fs.createShortNameBytes(newEntName, parentDirEntries)
	newFatDirEntry := fatDirectoryEntry{
		DIR_Name:         shortNameBytes,
		DIR_Attr:         newEntAttr,
		DIR_NTRes:        ntRes,
		DIR_CrtTimeTenth: newEntTime.tenth,
		DIR_CrtTime:      newEntTime.time,
		DIR_CrtDate:      newEntTime.date,
		DIR_LstAccDate:   newEntTime.date,
		DIR_WrtTime:      newEntTime.time,
		DIR_WrtDate:      newEntTime.date,
		DIR_FileSize:     0,
	}
	newFatDirEntry.setCluster(newEntCluster)

	return &Entry{fatDirectoryEntry: newFatDirEntry, longFileName: newEntName}
}

func (fs *FileSystem) writeEntryWithLfnToParent(
	entLongName string,
	ent *Entry,
	parentDirCluster *uint32,
	parentDirBytes []byte,
	atParentByteIndex int,
) error {
	items := []to32Bytes{}
	for _, lfnEntry := range fs.createLongFileNameEntries(entLongName, calculateShortNameChecksum(ent.fatDirectoryEntry.DIR_Name[:])) {
		items = append(items, &lfnEntry)
	}
	items = append(items, &ent.fatDirectoryEntry)

	return fs.writeEntriesToParent(items, parentDirCluster, parentDirBytes, atParentByteIndex)
}

type to32Bytes interface {
	toBytes() [32]byte
}

func (fs *FileSystem) writeEntriesToParent(
	items []to32Bytes,
	parentDirCluster *uint32,
	parentDirBytes []byte,
	atParentByteIndex int,
) error {
	// write new directory into parent's directory bytes
	for i, item := range items {
		start := atParentByteIndex + (FatDirectoryEntrySize * i)
		end := start + FatDirectoryEntrySize
		bytesToWrite := item.toBytes()
		copy(parentDirBytes[start:end], bytesToWrite[:])
	}

	// write back to disk
	if parentDirCluster == nil {
		// if root, just write it all back since it's in the dedicated root directory area
		_, err := fs.BackendWriter.WriteAt(parentDirBytes, int64(fs.RootDirectorySectorStart*fs.BytesPerSector))
		if err != nil {
			return err
		}
	} else {
		// if not root, then write the entire parent directory back to disk, along its cluster chain
		err := fs.writeClusterChain(*parentDirCluster, parentDirBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) writeDirectoryEntry(
	newDirName string,
	newDirCluster uint32,
	parentDirBytes []byte,
	parentDirEntries []Entry,
	parentDirCluster *uint32,
	atParentByteIndex int,
) ([]byte, error) {
	t := asFatTime(time.Now())
	newDirEntry := fs.createNewEntry(newDirName, t, 0x10, newDirCluster, parentDirEntries)

	err := fs.writeEntryWithLfnToParent(newDirName, newDirEntry, parentDirCluster, parentDirBytes, atParentByteIndex)
	if err != nil {
		return nil, err
	}

	// create special directory entries
	dotDirEntry := fatDirectoryEntry{
		DIR_Name:         [11]byte{'.', 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20},
		DIR_Attr:         0x10,
		DIR_CrtTimeTenth: t.tenth,
		DIR_CrtTime:      t.time,
		DIR_CrtDate:      t.date,
		DIR_LstAccDate:   t.date,
		DIR_WrtTime:      t.time,
		DIR_WrtDate:      t.date,
		DIR_FileSize:     0,
	}
	dotDirEntry.setCluster(newDirCluster)

	dotDotDirEntry := fatDirectoryEntry{
		DIR_Name:         [11]byte{'.', '.', 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20},
		DIR_Attr:         0x10,
		DIR_CrtTimeTenth: t.tenth,
		DIR_CrtTime:      t.time,
		DIR_CrtDate:      t.date,
		DIR_LstAccDate:   t.date,
		DIR_WrtTime:      t.time,
		DIR_WrtDate:      t.date,
		DIR_FileSize:     0,
	}
	if parentDirCluster != nil {
		dotDotDirEntry.setCluster(*parentDirCluster)
	}

	// create directory bytes for the new directory
	newDirectoryDataBytes := make([]byte, fs.BytesPerCluster)
	dotBytes := dotDirEntry.toBytes()
	dotDotBytes := dotDotDirEntry.toBytes()
	copy(newDirectoryDataBytes[:FatDirectoryEntrySize], dotBytes[:])
	copy(newDirectoryDataBytes[FatDirectoryEntrySize:], dotDotBytes[:])

	// write . and .. into new cluster in data region
	_, err = fs.BackendWriter.WriteAt(newDirectoryDataBytes, int64(fs.clusterToSector(newDirCluster)*fs.BytesPerSector))
	if err != nil {
		return nil, err
	}

	return newDirectoryDataBytes, nil
}

func (fs *FileSystem) getAvailableDirectoryEntry(
	nRequired int,
	parentDirCluster *uint32,
	parentDirBytes []byte,
) (int, []byte, error) {
	freeIndex, ok := fs.findAvailableDirectoryEntry(parentDirBytes, nRequired)
	if !ok {
		// if we're in the root directory area we can't expand on fat16
		if parentDirCluster == nil {
			return 0, nil, fmt.Errorf("no available space for creating root directory entry")
		}

		// expand the current directory's cluster chain since we're out of space
		extraCluster, err := fs.allocateClusterChain(int64(nRequired * FatDirectoryEntrySize))
		if err != nil {
			return 0, nil, err
		}

		err = fs.writeClusterToFats(*parentDirCluster, extraCluster)
		if err != nil {
			return 0, nil, err
		}

		parentDirBytes, err = fs.GetClusterChainBytes(*parentDirCluster)
		if err != nil {
			return 0, nil, fmt.Errorf("could not read directory bytes: %w", err)
		}

		// TODONICE: can optimise and return start of new cluster
		freeIndex, ok = fs.findAvailableDirectoryEntry(parentDirBytes, nRequired)
		if !ok {
			return 0, nil, fmt.Errorf("no available space for creating directory entry")
		}
	}

	return freeIndex, parentDirBytes, nil
}

func (fs *FileSystem) findEntry(path string) (*Entry, *readDirResult, bool, error) {
	dirPath := filepath.Dir(path)
	baseName := filepath.Base(path)
	parent, err := fs.readDir(dirPath, false)
	if err != nil {
		return nil, nil, false, err
	}

	for _, existing := range parent.entries {
		if slices.ContainsFunc(existing.names(), func(name string) bool { return strings.EqualFold(name, baseName) }) {
			return &existing, parent, true, nil
		}
	}

	return nil, parent, false, nil
}

func (fs *FileSystem) removeEntryFromParent(entry *Entry, parentDirCluster *uint32) error {
	entryIndex, parentDirBytes, err := fs.findIndexInParentBytes(entry, parentDirCluster)
	if err != nil {
		return err
	}

	// find all associated lfn entries
	sfnChecksum := calculateShortNameChecksum(entry.DIR_Name[:])
	lfnIndex := entryIndex - FatDirectoryEntrySize
	for lfnIndex >= 0 && parentDirBytes[lfnIndex+11] == 0x0F && parentDirBytes[lfnIndex+13] == sfnChecksum {
		parentDirBytes[lfnIndex] = 0xE5
		lfnIndex -= FatDirectoryEntrySize
	}

	// set first byte of entry to 0xE5 to mark as deleted
	parentDirBytes[entryIndex] = 0xE5

	// write back to disk
	err = fs.writeEntriesToParent(
		[]to32Bytes{},
		parentDirCluster,
		parentDirBytes,
		entryIndex,
	)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) removeEntryFromParentWithCluster(entry *Entry, parentDirCluster *uint32) error {
	err := fs.removeEntryFromParent(entry, parentDirCluster)
	if err != nil {
		return err
	}

	// remove any allocated clusters from FATs
	cluster := entry.clusterNumber()
	if cluster == 0 {
		return nil
	}

	err = fs.deleteClusterChain(cluster)
	return err
}
