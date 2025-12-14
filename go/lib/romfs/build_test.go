package romfs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuild(t *testing.T) {
	// Create test data
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some files and directories
	createFile(t, filepath.Join(inputDir, "file1.txt"), "Hello World")
	createFile(t, filepath.Join(inputDir, "file2.txt"), "Another file")

	subDir := filepath.Join(inputDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	createFile(t, filepath.Join(subDir, "file3.txt"), "File in subdir")

	subSubDir := filepath.Join(subDir, "subsubdir")
	if err := os.Mkdir(subSubDir, 0755); err != nil {
		t.Fatal(err)
	}
	createFile(t, filepath.Join(subSubDir, "file4.txt"), "Deep file")

	// Paths for output
	cOutput := filepath.Join(tmpDir, "c_output.romfs")
	goOutput := filepath.Join(tmpDir, "go_output.romfs")

	// Run C program
	// Assuming romfs_gen is in testdata/romfs_gen
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	romfsGenPath := filepath.Join(cwd, "testdata", "romfs_gen")

	cmd := exec.Command(romfsGenPath, inputDir, cOutput)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("C program failed: %v\nOutput: %s", err, out)
	}

	// Run Go Build
	if err := Build(inputDir, goOutput); err != nil {
		t.Fatalf("Go Build failed: %v", err)
	}

	// Compare outputs
	cData, err := os.ReadFile(cOutput)
	if err != nil {
		t.Fatal(err)
	}
	goData, err := os.ReadFile(goOutput)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(cData, goData) {
		t.Errorf("Outputs do not match")
		// Optional: dump hex diff or save files for inspection
		t.Logf("C output size: %d", len(cData))
		t.Logf("Go output size: %d", len(goData))

		// Find first difference
		minLen := min(len(goData), len(cData))
		for i := range minLen {
			if cData[i] != goData[i] {
				t.Errorf("First mismatch at offset 0x%x: C=0x%02x, Go=0x%02x", i, cData[i], goData[i])
				break
			}
		}
	}
}

func createFile(t *testing.T, path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
