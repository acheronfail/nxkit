package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// sequentialBuffer creates a byte slice filled with sequential values starting from start
func sequentialBuffer(size int, start byte) []byte {
	buf := make([]byte, size)
	for i := range size {
		buf[i] = start + byte(i)
	}
	return buf
}

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get test filename")
	}

	dir, err := os.MkdirTemp("", filepath.Base(file))
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}
