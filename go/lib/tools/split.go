package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ProgressFunc func(done, total int64)

// Split splits a file into multiple parts
// If asArchive is true, creates a directory structure, otherwise splits into files
// If inPlace is true, modifies the original file
// If splitSize is 0, it will be set automatically
func Split(filePath string, asArchive, inPlace bool, splitSize int64) (string, error) {
	return SplitWithProgress(filePath, asArchive, inPlace, splitSize, nil)
}

func SplitWithProgress(filePath string, asArchive, inPlace bool, splitSize int64, progress ProgressFunc) (string, error) {
	return SplitWithProgressContext(context.Background(), filePath, asArchive, inPlace, splitSize, progress)
}

func SplitWithProgressContext(ctx context.Context, filePath string, asArchive, inPlace bool, splitSize int64, progress ProgressFunc) (string, error) {
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

	totalBytes := fileSize
	if inPlace {
		totalBytes = max(fileSize-splitSize, 0)
	}
	var copiedBytes int64
	if progress != nil {
		progress(0, totalBytes)
	}

	// Process chunks from the end of the file towards the beginning
	for offset > limit {
		if err := ctx.Err(); err != nil {
			return "", err
		}

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
		if err := copyNWithProgress(ctx, splitFile, file, bytesToCopy, &copiedBytes, totalBytes, progress); err != nil {
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

	if progress != nil {
		progress(totalBytes, totalBytes)
	}

	outputPath := filepath.Dir(getSplitPath())
	if asArchive {
		// The Switch treats a directory containing 00, 01, ... as one
		// concatenated file only when the directory has the FAT archive bit.
		// This is best-effort because some host filesystems do not expose that
		// bit and a later copy may not preserve it.
		if err := setArchiveBit(outputPath); err != nil {
			log.Printf("warning: could not set archive bit on %s: %v", outputPath, err)
		}
	}
	return outputPath, nil
}

func copyNWithProgress(ctx context.Context, dst io.Writer, src io.Reader, bytesToCopy int64, copiedBytes *int64, totalBytes int64, progress ProgressFunc) error {
	if bytesToCopy == 0 {
		return nil
	}

	buf := make([]byte, 1024*1024)
	remaining := bytesToCopy
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		readSize := int64(len(buf))
		if remaining < readSize {
			readSize = remaining
		}

		nr, readErr := src.Read(buf[:readSize])
		if nr > 0 {
			nw, writeErr := dst.Write(buf[:nr])
			if writeErr != nil {
				return writeErr
			}
			if nw != nr {
				return io.ErrShortWrite
			}
			remaining -= int64(nw)
			*copiedBytes += int64(nw)
			if progress != nil {
				progress(*copiedBytes, totalBytes)
			}
		}
		if readErr != nil {
			if readErr == io.EOF && remaining == 0 {
				return nil
			}
			return readErr
		}
	}

	return nil
}
