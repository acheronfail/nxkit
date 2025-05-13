package ivfc

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

const (
	IVFCHashBlockSize = 0x4000 // Based on the original code's value
)

// CreateLevel creates a hash level file from a source level file
func CreateLevel(dstLevelPath, srcLevelPath string) (int64, error) {
	// Open source file for reading
	srcFile, err := os.Open(srcLevelPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dstLevelPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Get source file size
	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get source file info: %w", err)
	}
	srcFileSize := srcFileInfo.Size()

	// Create buffer for reading blocks
	buf := make([]byte, IVFCHashBlockSize)

	// Process file in blocks
	var offset int64 = 0
	for offset < srcFileSize {
		readSize := IVFCHashBlockSize
		if offset+int64(readSize) >= srcFileSize {
			readSize = int(srcFileSize - offset)
		}

		// Read block
		n, err := srcFile.Read(buf[:readSize])
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("failed to read source file: %w", err)
		}
		if n != readSize {
			return 0, fmt.Errorf("short read: got %d bytes, expected %d", n, readSize)
		}

		// Calculate hash of block
		hash := sha256.Sum256(buf[:readSize])

		// Write hash to destination file
		if _, err := dstFile.Write(hash[:]); err != nil {
			return 0, fmt.Errorf("failed to write hash: %w", err)
		}

		offset += int64(readSize)
	}

	// Add padding if needed
	currentOffset, err := dstFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("failed to get current file position: %w", err)
	}

	paddingSize := IVFCHashBlockSize - (currentOffset % IVFCHashBlockSize)
	if paddingSize != IVFCHashBlockSize {
		padding := make([]byte, paddingSize)
		if _, err := dstFile.Write(padding); err != nil {
			return 0, fmt.Errorf("failed to write padding: %w", err)
		}
	}

	// Get final size
	finalSize, err := dstFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("failed to get final file size: %w", err)
	}

	return finalSize, nil
}

// CalculateMasterHash calculates the master hash from the level 1 file
func CalculateMasterHash(ivfcLevel1Path string) ([]byte, error) {
	// Open level 1 file
	file, err := os.Open(ivfcLevel1Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open level 1 file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	size := fileInfo.Size()

	// Read entire file
	buf := make([]byte, size)
	if _, err := io.ReadFull(file, buf); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate hash
	hash := sha256.Sum256(buf)
	return hash[:], nil
}
