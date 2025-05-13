package ivfc

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// VerifyBlock verifies a specific 16KB block of data against the IVFC hash chain
// blockIndex is the 16KB block number to verify
// Returns true if verification succeeds
func VerifyBlock(dataPath string, blockIndex int64, ivfcLevelPaths []string, masterHash []byte) error {
	// Read the target block from the data file (Level 5)
	block, err := readBlock(dataPath, blockIndex)
	if err != nil {
		return fmt.Errorf("failed to read data block: %w", err)
	}

	// Calculate hash of the block
	currentHash := sha256.Sum256(block)

	// Verify hash chain up through each level
	for level := len(ivfcLevelPaths) - 1; level >= 0; level-- {
		// Calculate which hash entry we need from this level
		hashIndex := blockIndex

		// Read the expected hash from the level file
		expectedHash, err := readHashFromLevel(ivfcLevelPaths[level], hashIndex)
		if err != nil {
			return fmt.Errorf("failed to read hash from level %d: %w", level, err)
		}

		// Compare hashes
		if string(currentHash[:]) != string(expectedHash) {
			return fmt.Errorf("hash mismatch at level %d", level)
		}

		// For next iteration: calculate hash location in next level up
		blockIndex /= 512 // Since each level covers 512x more data
		currentHash = sha256.Sum256(expectedHash)
	}

	// Finally verify against master hash
	if string(currentHash[:]) != string(masterHash) {
		return fmt.Errorf("master hash verification failed")
	}

	return nil
}

// VerifyFile verifies an entire file against the IVFC hash chain
func VerifyFile(dataPath string, ivfcLevelPaths []string, masterHash []byte) error {
	// Get file size
	fileInfo, err := os.Stat(dataPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate number of blocks
	numBlocks := (fileInfo.Size() + IVFCHashBlockSize - 1) / IVFCHashBlockSize

	// Verify each block
	for i := int64(0); i < numBlocks; i++ {
		if err := VerifyBlock(dataPath, i, ivfcLevelPaths, masterHash); err != nil {
			return fmt.Errorf("block %d verification failed: %w", i, err)
		}
	}

	return nil
}

// Helper function to read a specific block from a file
func readBlock(filePath string, blockIndex int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Seek to the block position
	offset := blockIndex * IVFCHashBlockSize
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Read the block
	block := make([]byte, IVFCHashBlockSize)
	n, err := file.Read(block)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return block[:n], nil
}

// Helper function to read a specific hash from a level file
func readHashFromLevel(levelPath string, hashIndex int64) ([]byte, error) {
	file, err := os.Open(levelPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Seek to the hash position
	offset := hashIndex * 32 // Each hash is 32 bytes
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Read the hash
	hash := make([]byte, 32)
	if _, err := io.ReadFull(file, hash); err != nil {
		return nil, err
	}

	return hash, nil
}
