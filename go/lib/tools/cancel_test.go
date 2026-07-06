package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitWithProgressContextCanceled(t *testing.T) {
	dir := createTempDir(t)
	filePath := filepath.Join(dir, "file")
	if err := os.WriteFile(filePath, sequentialBuffer(100, 0), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := SplitWithProgressContext(ctx, filePath, false, false, 30, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SplitWithProgressContext error = %v, want context.Canceled", err)
	}
}

func TestMergeWithProgressContextCanceled(t *testing.T) {
	dir := createTempDir(t)
	basePath := filepath.Join(dir, "file")
	if err := os.WriteFile(basePath+".00", sequentialBuffer(30, 0), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(basePath+".01", sequentialBuffer(30, 30), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := MergeWithProgressContext(ctx, basePath+".00", false, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("MergeWithProgressContext error = %v, want context.Canceled", err)
	}
}
