package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMerge(t *testing.T) {
	t.Run("copy", func(t *testing.T) {
		dir := createTempDir(t)
		basePath := filepath.Join(dir, "file")

		// Create split files
		err := os.WriteFile(basePath+".00", sequentialBuffer(30, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".01", sequentialBuffer(30, 30), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".02", sequentialBuffer(30, 60), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".03", sequentialBuffer(10, 90), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Merge the files
		outputPath, err := Merge(basePath+".00", false)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
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

		// Check if original files and merged file exist
		expectedFiles := []string{"file", "file.00", "file.01", "file.02", "file.03"}
		for _, expected := range expectedFiles {
			found := slices.Contains(fileNames, expected)
			if !found {
				t.Errorf("Expected file %s not found in directory", expected)
			}
		}

		// Check merged file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read merged file: %v", err)
		}

		if !bytes.Equal(content, sequentialBuffer(100, 0)) {
			t.Errorf("Content of merged file is incorrect")
		}
	})

	t.Run("in place", func(t *testing.T) {
		dir := createTempDir(t)
		basePath := filepath.Join(dir, "file")

		// Create split files
		err := os.WriteFile(basePath+".00", sequentialBuffer(30, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".01", sequentialBuffer(30, 30), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".02", sequentialBuffer(30, 60), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".03", sequentialBuffer(10, 90), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Merge the files in place
		outputPath, err := Merge(basePath+".00", true)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		// Check directory contents
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// Should only have the merged file
		if len(files) != 1 {
			t.Errorf("Expected only one file in directory, got %d", len(files))
		}

		if len(files) > 0 && files[0].Name() != "file" {
			t.Errorf("Expected file name to be 'file', got %s", files[0].Name())
		}

		// Check merged file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read merged file: %v", err)
		}

		if !bytes.Equal(content, sequentialBuffer(100, 0)) {
			t.Errorf("Content of merged file is incorrect")
		}
	})

	t.Run("in place with extension", func(t *testing.T) {
		dir := createTempDir(t)
		basePath := filepath.Join(dir, "file.bin")

		// Create split files
		err := os.WriteFile(basePath+".00", sequentialBuffer(30, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".01", sequentialBuffer(30, 30), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".02", sequentialBuffer(30, 60), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(basePath+".03", sequentialBuffer(10, 90), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Merge the files in place
		outputPath, err := Merge(basePath+".00", true)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		// Check directory contents
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// Should only have the merged file
		if len(files) != 1 {
			t.Errorf("Expected only one file in directory, got %d", len(files))
		}

		if len(files) > 0 && files[0].Name() != "file.bin" {
			t.Errorf("Expected file name to be 'file.bin', got %s", files[0].Name())
		}

		// Check merged file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read merged file: %v", err)
		}

		if !bytes.Equal(content, sequentialBuffer(100, 0)) {
			t.Errorf("Content of merged file is incorrect")
		}
	})

	t.Run("archive copy", func(t *testing.T) {
		dir := createTempDir(t)
		splitDir := filepath.Join(dir, "file")

		// Create directory structure
		err := os.MkdirAll(splitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create split files in archive format
		err = os.WriteFile(filepath.Join(splitDir, "00"), sequentialBuffer(30, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "01"), sequentialBuffer(30, 30), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "02"), sequentialBuffer(30, 60), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "03"), sequentialBuffer(10, 90), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Merge the files
		outputPath, err := Merge(filepath.Join(splitDir, "00"), false)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
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

		// Check if merged file and split directory exist
		foundSplitDir := false
		foundMergedFile := false
		for _, name := range fileNames {
			if name == "file" && !filepath.IsAbs(name) {
				foundSplitDir = true
			}
			if name == "file_merged" {
				foundMergedFile = true
			}
		}

		if !foundSplitDir {
			t.Errorf("Expected split directory not found")
		}
		if !foundMergedFile {
			t.Errorf("Expected merged file not found")
		}

		// Check merged file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read merged file: %v", err)
		}

		if !bytes.Equal(content, sequentialBuffer(100, 0)) {
			t.Errorf("Content of merged file is incorrect")
		}
	})

	t.Run("archive in place", func(t *testing.T) {
		dir := createTempDir(t)
		splitDir := filepath.Join(dir, "file")

		// Create directory structure
		err := os.MkdirAll(splitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create split files in archive format
		err = os.WriteFile(filepath.Join(splitDir, "00"), sequentialBuffer(30, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "01"), sequentialBuffer(30, 30), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "02"), sequentialBuffer(30, 60), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "03"), sequentialBuffer(10, 90), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Merge the files in place
		outputPath, err := Merge(filepath.Join(splitDir, "00"), true)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		// Check directory contents
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// Should only have the merged file
		if len(files) != 1 {
			t.Errorf("Expected only one file in directory, got %d", len(files))
		}

		if len(files) > 0 && files[0].Name() != "file_merged" {
			t.Errorf("Expected file name to be 'file_merged', got %s", files[0].Name())
		}

		// Split directory should not exist
		if _, err := os.Stat(splitDir); !os.IsNotExist(err) {
			t.Errorf("Split directory should not exist after in-place merge")
		}

		// Check merged file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read merged file: %v", err)
		}

		if !bytes.Equal(content, sequentialBuffer(100, 0)) {
			t.Errorf("Content of merged file is incorrect")
		}
	})

	t.Run("archive copy with extension", func(t *testing.T) {
		dir := createTempDir(t)
		splitDir := filepath.Join(dir, "file.nsp")

		// Create directory structure
		err := os.MkdirAll(splitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create split files in archive format
		err = os.WriteFile(filepath.Join(splitDir, "00"), sequentialBuffer(30, 0), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "01"), sequentialBuffer(30, 30), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "02"), sequentialBuffer(30, 60), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		err = os.WriteFile(filepath.Join(splitDir, "03"), sequentialBuffer(10, 90), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Merge the files
		outputPath, err := Merge(filepath.Join(splitDir, "00"), false)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		// Check directory contents
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// Check if merged file exists with correct extension
		expectedMergedFile := "file.nsp"
		found := false
		for _, file := range files {
			if file.Name() == expectedMergedFile {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected merged file %s not found", expectedMergedFile)
		}

		// Check merged file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read merged file: %v", err)
		}

		if !bytes.Equal(content, sequentialBuffer(100, 0)) {
			t.Errorf("Content of merged file is incorrect")
		}
	})
}
