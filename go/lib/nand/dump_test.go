package nand

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDumpBackend(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	tempDir := t.TempDir()

	// Create test files
	combinedFilePath := filepath.Join(tempDir, "rawnand.bin")
	splitBaseFilePath := filepath.Join(tempDir, "rawnand.bin.00")

	// Create a combined file
	combinedFile, err := os.Create(combinedFilePath)
	require.NoError(t, err)
	defer combinedFile.Close()
	_, err = combinedFile.Write(data)
	require.NoError(t, err)

	// Create split files
	for i := range 4 {
		splitFilePath := filepath.Join(tempDir, "rawnand.bin."+formatExtension(i))
		splitFile, err := os.Create(splitFilePath)
		require.NoError(t, err)
		_, err = splitFile.Write(data[i*4 : (i+1)*4])
		require.NoError(t, err)
		splitFile.Close()
	}

	// create a random file as well to test it is ignored
	_, err = os.Create(filepath.Join(tempDir, "random.bin"))
	require.NoError(t, err)

	t.Run("combined file", func(t *testing.T) {
		backend, err := NewDumpBackend(combinedFilePath, true)
		require.NoError(t, err)
		defer backend.Close()

		assert.NotNil(t, backend)
		stat, err := backend.Stat()
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), stat.Size())
	})

	t.Run("split files", func(t *testing.T) {
		backend, err := NewDumpBackend(splitBaseFilePath, true)
		require.NoError(t, err)
		assert.NotNil(t, backend)
		defer backend.Close()

		stat, err := backend.Stat()
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), stat.Size())

		buf := make([]byte, stat.Size())
		n, err := backend.ReadAt(buf, 0)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf)

		sb, ok := backend.(*SplitBackend)
		assert.True(t, ok)
		assert.Equal(t, 4, len(sb.files))
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := NewDumpBackend(filepath.Join(tempDir, "nonexistent.bin"), true)
		assert.Error(t, err)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := NewDumpBackend(filepath.Join(tempDir, "nonexistent", "rawnand.bin.00"), true)
		assert.Error(t, err)
	})

	t.Run("no matching split files", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.Mkdir(emptyDir, 0755)
		require.NoError(t, err)

		emptyFilePath := filepath.Join(emptyDir, "empty.00")
		emptyFile, err := os.Create(emptyFilePath)
		require.NoError(t, err)
		emptyFile.Close()

		backend, err := NewDumpBackend(emptyFilePath, true)
		assert.NoError(t, err)
		defer backend.Close()

		// Should fall back to combined backend since no matching split files found
		_, isSplitBackend := backend.(*SplitBackend)
		assert.False(t, isSplitBackend)
	})
}

func formatExtension(i int) string {
	return fmt.Sprintf("%02d", i)
}
