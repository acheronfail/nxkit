package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSplit(t *testing.T) {
	t.Run("copy", func(t *testing.T) {
		dir := createTempDir(t)
		filePath := filepath.Join(dir, "file")

		// Write test file
		err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Split the file
		_, err = Split(filePath, false, false, 30)
		if err != nil {
			t.Fatalf("Split failed: %v", err)
		}

		// Check directory contents
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		fileNames := make([]string, len(files))
		for i, file := range files {
			fileNames[i] = file.Name()
		}

		// Check if expected files exist
		expectedFiles := []string{"file", "file.00", "file.01", "file.02", "file.03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in directory", expected)
			}
		}

		// Check file contents
		content00, err := os.ReadFile(filepath.Join(dir, "file.00"))
		if err != nil {
			t.Fatalf("Failed to read file.00: %v", err)
		}
		content01, err := os.ReadFile(filepath.Join(dir, "file.01"))
		if err != nil {
			t.Fatalf("Failed to read file.01: %v", err)
		}
		content02, err := os.ReadFile(filepath.Join(dir, "file.02"))
		if err != nil {
			t.Fatalf("Failed to read file.02: %v", err)
		}
		content03, err := os.ReadFile(filepath.Join(dir, "file.03"))
		if err != nil {
			t.Fatalf("Failed to read file.03: %v", err)
		}

		if !bytes.Equal(content00, sequentialBuffer(30, 0)) {
			t.Errorf("Content of file.00 is incorrect")
		}
		if !bytes.Equal(content01, sequentialBuffer(30, 30)) {
			t.Errorf("Content of file.01 is incorrect")
		}
		if !bytes.Equal(content02, sequentialBuffer(30, 60)) {
			t.Errorf("Content of file.02 is incorrect")
		}
		if !bytes.Equal(content03, sequentialBuffer(10, 90)) {
			t.Errorf("Content of file.03 is incorrect")
		}
	})

	t.Run("in place", func(t *testing.T) {
		dir := createTempDir(t)
		filePath := filepath.Join(dir, "file")

		// Write test file
		err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Split the file in place
		_, err = Split(filePath, false, true, 30)
		if err != nil {
			t.Fatalf("Split failed: %v", err)
		}

		// Check directory contents - original file should be gone
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		fileNames := make([]string, len(files))
		for i, file := range files {
			fileNames[i] = file.Name()
		}

		// Check if expected files exist
		expectedFiles := []string{"file.00", "file.01", "file.02", "file.03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in directory", expected)
			}
		}

		// Original file should not exist
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("Original file should not exist after in-place split")
		}

		// Check file contents
		content00, err := os.ReadFile(filepath.Join(dir, "file.00"))
		if err != nil {
			t.Fatalf("Failed to read file.00: %v", err)
		}
		content01, err := os.ReadFile(filepath.Join(dir, "file.01"))
		if err != nil {
			t.Fatalf("Failed to read file.01: %v", err)
		}
		content02, err := os.ReadFile(filepath.Join(dir, "file.02"))
		if err != nil {
			t.Fatalf("Failed to read file.02: %v", err)
		}
		content03, err := os.ReadFile(filepath.Join(dir, "file.03"))
		if err != nil {
			t.Fatalf("Failed to read file.03: %v", err)
		}

		if !bytes.Equal(content00, sequentialBuffer(30, 0)) {
			t.Errorf("Content of file.00 is incorrect")
		}
		if !bytes.Equal(content01, sequentialBuffer(30, 30)) {
			t.Errorf("Content of file.01 is incorrect")
		}
		if !bytes.Equal(content02, sequentialBuffer(30, 60)) {
			t.Errorf("Content of file.02 is incorrect")
		}
		if !bytes.Equal(content03, sequentialBuffer(10, 90)) {
			t.Errorf("Content of file.03 is incorrect")
		}
	})

	t.Run("in place with extension", func(t *testing.T) {
		dir := createTempDir(t)
		filePath := filepath.Join(dir, "file.bin")

		// Write test file
		err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Split the file in place
		_, err = Split(filePath, false, true, 30)
		if err != nil {
			t.Fatalf("Split failed: %v", err)
		}

		// Check directory contents
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		fileNames := make([]string, len(files))
		for i, file := range files {
			fileNames[i] = file.Name()
		}

		// Check if expected files exist with extension
		expectedFiles := []string{"file.bin.00", "file.bin.01", "file.bin.02", "file.bin.03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in directory", expected)
			}
		}

		// Check file contents
		content00, err := os.ReadFile(filepath.Join(dir, "file.bin.00"))
		if err != nil {
			t.Fatalf("Failed to read file.bin.00: %v", err)
		}
		if !bytes.Equal(content00, sequentialBuffer(30, 0)) {
			t.Errorf("Content of file.bin.00 is incorrect")
		}
	})

	t.Run("as archive copy", func(t *testing.T) {
		dir := createTempDir(t)
		filePath := filepath.Join(dir, "file")

		// Write test file
		err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Split the file as archive
		_, err = Split(filePath, true, false, 30)
		if err != nil {
			t.Fatalf("Split failed: %v", err)
		}

		// Check directory structure
		dirFiles, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		foundSplitDir := false
		for _, file := range dirFiles {
			if file.Name() == "file_split" && file.IsDir() {
				foundSplitDir = true
				break
			}
		}
		if !foundSplitDir {
			t.Fatalf("Split directory not found")
		}

		// Check archive directory contents
		archiveFiles, err := os.ReadDir(filepath.Join(dir, "file_split"))
		if err != nil {
			t.Fatalf("Failed to read archive directory: %v", err)
		}

		fileNames := make([]string, len(archiveFiles))
		for i, file := range archiveFiles {
			fileNames[i] = file.Name()
		}

		// Check if expected files exist in archive
		expectedFiles := []string{"00", "01", "02", "03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in archive directory", expected)
			}
		}

		// Check file contents
		content00, err := os.ReadFile(filepath.Join(dir, "file_split", "00"))
		if err != nil {
			t.Fatalf("Failed to read 00: %v", err)
		}
		if !bytes.Equal(content00, sequentialBuffer(30, 0)) {
			t.Errorf("Content of 00 is incorrect")
		}

		// Original file should still exist
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Original file should still exist after archive copy split")
		}
	})

	t.Run("as archive in place", func(t *testing.T) {
		dir := createTempDir(t)
		filePath := filepath.Join(dir, "file")

		// Write test file
		err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Split the file as archive in place
		_, err = Split(filePath, true, true, 30)
		if err != nil {
			t.Fatalf("Split failed: %v", err)
		}

		// Check directory structure
		dirFiles, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		foundSplitDir := false
		for _, file := range dirFiles {
			if file.Name() == "file_split" && file.IsDir() {
				foundSplitDir = true
				break
			}
		}
		if !foundSplitDir {
			t.Fatalf("Split directory not found")
		}

		// Original file should not exist
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("Original file should not exist after in-place archive split")
		}

		// Check archive directory contents and file contents
		archiveFiles, err := os.ReadDir(filepath.Join(dir, "file_split"))
		if err != nil {
			t.Fatalf("Failed to read archive directory: %v", err)
		}

		fileNames := make([]string, len(archiveFiles))
		for i, file := range archiveFiles {
			fileNames[i] = file.Name()
		}

		// Check if expected files exist in archive
		expectedFiles := []string{"00", "01", "02", "03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in archive directory", expected)
			}
		}
	})

	t.Run("as archive in place with extension", func(t *testing.T) {
		dir := createTempDir(t)
		filePath := filepath.Join(dir, "file.nsp")

		// Write test file
		err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Split the file as archive in place
		_, err = Split(filePath, true, true, 30)
		if err != nil {
			t.Fatalf("Split failed: %v", err)
		}

		// Check directory structure
		dirFiles, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		foundSplitDir := false
		for _, file := range dirFiles {
			if file.Name() == "file_split.nsp" && file.IsDir() {
				foundSplitDir = true
				break
			}
		}
		if !foundSplitDir {
			t.Fatalf("Split directory not found")
		}

		// Original file should not exist
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("Original file should not exist after in-place archive split")
		}

		// Check archive directory contents
		archiveFiles, err := os.ReadDir(filepath.Join(dir, "file_split.nsp"))
		if err != nil {
			t.Fatalf("Failed to read archive directory: %v", err)
		}

		fileNames := make([]string, len(archiveFiles))
		for i, file := range archiveFiles {
			fileNames[i] = file.Name()
		}

		// Check if expected files exist in archive
		expectedFiles := []string{"00", "01", "02", "03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in archive directory", expected)
			}
		}

		// Check file contents
		content00, err := os.ReadFile(filepath.Join(dir, "file_split.nsp", "00"))
		if err != nil {
			t.Fatalf("Failed to read 00: %v", err)
		}
		if !bytes.Equal(content00, sequentialBuffer(30, 0)) {
			t.Errorf("Content of 00 is incorrect")
		}
	})
}
