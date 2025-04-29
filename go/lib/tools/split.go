package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Split splits a file into multiple parts
// If asArchive is true, creates a directory structure, otherwise splits into files
// If inPlace is true, modifies the original file
// If splitSize is 0, it will be set automatically
func Split(filePath string, asArchive, inPlace bool, splitSize int64) (string, error) {
	if splitSize == 0 {
		if asArchive {
			splitSize = 0xffff0000
		} else {
			splitSize = 0x80000000
		}
	}

	file, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := fileInfo.Size()

	offset := fileSize - (fileSize % splitSize)
	count := fileSize / splitSize

	getSplitPath := func() string {
		countSuffix := fmt.Sprintf("%02d", count)

		if asArchive {
			dir := filepath.Dir(filePath)
			ext := filepath.Ext(filePath)
			base := filepath.Base(filePath)

			if ext != "" {
				baseName := strings.TrimSuffix(base, ext)
				return filepath.Join(dir, baseName+"_split"+ext, countSuffix)
			} else {
				return filepath.Join(dir, base+"_split", countSuffix)
			}
		}

		return filePath + "." + countSuffix
	}

	// If not in-place, process all chunks including the first one
	limit := int64(0)
	if !inPlace {
		limit = -1
	}

	// Process chunks from the end of the file towards the beginning
	for offset > limit {
		splitPath := getSplitPath()

		if asArchive {
			if err := os.MkdirAll(filepath.Dir(splitPath), 0755); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
		}

		start := offset
		end := min(offset+splitSize, fileSize)

		splitFile, err := os.Create(splitPath)
		if err != nil {
			return "", fmt.Errorf("failed to create split file: %w", err)
		}

		if _, err := file.Seek(start, 0); err != nil {
			splitFile.Close()
			return "", fmt.Errorf("failed to seek in source file: %w", err)
		}

		bytesToCopy := end - start
		if _, err := io.CopyN(splitFile, file, bytesToCopy); err != nil {
			splitFile.Close()
			return "", fmt.Errorf("failed to copy data: %w", err)
		}

		splitFile.Close()

		if inPlace {
			if err := os.Truncate(filePath, offset); err != nil {
				return "", fmt.Errorf("failed to truncate file: %w", err)
			}
		}

		offset -= splitSize
		count--
	}

	// Close the file before renaming
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("failed to close file: %w", err)
	}

	// If in-place mode, rename the original file to the first split part
	if inPlace {
		splitPath := getSplitPath()

		if err := os.Rename(filePath, splitPath); err != nil {
			return "", fmt.Errorf("failed to rename file: %w", err)
		}
	}

	outputPath := filepath.Dir(getSplitPath())
	return outputPath, nil
}
