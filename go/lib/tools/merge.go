package tools

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func Merge(path string, inPlace bool) (string, error) {
	firstFileName := filepath.Base(path)
	matched, err := regexp.MatchString(`^00$|.*\.00$`, firstFileName)
	if err != nil {
		return "", err
	}
	if !matched {
		return "", errors.New("the selected file doesn't appear to be the first part of a split! Select 00 or myfile.00")
	}

	isArchive := firstFileName == "00"
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	reArchivePart := regexp.MustCompile(`^\d\d$`)
	reFilePart := regexp.MustCompile(`\.\d\d$`)

	var splitFiles []string
	for _, entry := range entries {
		name := entry.Name()
		pattern := reFilePart
		if isArchive {
			pattern = reArchivePart
		}
		if pattern.MatchString(name) {
			splitFiles = append(splitFiles, filepath.Join(dir, name))
		}
	}

	sort.Strings(splitFiles)

	var mergedFileName string
	if isArchive {
		parsed := filepath.Base(dir)
		ext := filepath.Ext(parsed)
		if ext != "" {
			mergedFileName = fmt.Sprintf("%s_merged%s", strings.TrimSuffix(parsed, ext), ext)
		} else {
			mergedFileName = fmt.Sprintf("%s_merged", parsed)
		}
	} else {
		mergedFileName = strings.TrimSuffix(firstFileName, ".00")
	}

	var mergedFilePath string
	if isArchive {
		mergedFilePath = filepath.Join(filepath.Dir(dir), mergedFileName)
	} else {
		mergedFilePath = filepath.Join(dir, mergedFileName)
	}

	mergedFile, err := os.Create(mergedFilePath)
	if err != nil {
		return "", err
	}
	defer mergedFile.Close()

	for _, splitFile := range splitFiles {
		fmt.Printf("Merging %s...\n", splitFile)
		splitFileHandle, err := os.Open(splitFile)
		if err != nil {
			return "", err
		}

		_, err = io.Copy(mergedFile, splitFileHandle)
		splitFileHandle.Close()
		if err != nil {
			return "", err
		}

		if inPlace {
			if err := os.Remove(splitFile); err != nil {
				return "", err
			}
		}
	}

	if inPlace && isArchive {
		if err := os.Remove(dir); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return mergedFilePath, nil
}
