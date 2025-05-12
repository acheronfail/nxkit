package nand

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/fat/backend/file"
)

func NewDumpBackend(path string, readOnly bool) (backend.Storage, error) {
	if strings.HasSuffix(path, ".00") {
		// Extract the base path without the .00 extension
		basePath := path[:len(path)-3]
		dirPath := filepath.Dir(basePath) + string(filepath.Separator)
		baseFileName := filepath.Base(basePath)

		// Read the directory to find all split files
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
		}

		// Collect all matching files
		var splitFilePaths []string
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				if strings.HasPrefix(name, baseFileName) {
					if _, err := fmt.Sscanf(name[len(baseFileName):], ".%02d", new(int)); err == nil {
						splitFilePaths = append(splitFilePaths, dirPath+name)
					}
				}
			}
		}

		// Sort files to ensure correct order
		sort.Strings(splitFilePaths)
		if len(splitFilePaths) > 0 {
			return NewSplitDumpBackend(splitFilePaths, readOnly)
		}
	}

	return NewCombinedDumpBackend(path, readOnly)
}

// CombinedBackend represents backend storage for a combined NAND dump (e.g., rawnand.bin)
type CombinedBackend struct {
	backend.Storage
}

func NewCombinedDumpBackend(path string, readOnly bool) (backend.Storage, error) {
	storage, err := file.OpenFromPath(path, readOnly)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

// SplitBackend represents backend storage for split NAND dumps (e.g., rawnand.bin.00, rawnand.bin.01, etc.)
type SplitBackend struct {
	files        []splitFile
	readOnly     bool
	seekLocation int64
	totalSize    int64
	fileMode     os.FileMode
	openFlags    int
	openMode     int
	writable     bool
}

type splitFile struct {
	path   string
	offset int64
	size   int64
	file   *os.File
}

// NewSplitDumpBackend creates a new backend from multiple split dump files
func NewSplitDumpBackend(paths []string, readOnly bool) (backend.Storage, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("NewSplitDumpBackend requires at least 1 file path")
	}

	if len(paths) == 1 {
		return NewCombinedDumpBackend(paths[0], readOnly)
	}

	backend := &SplitBackend{
		readOnly:     readOnly,
		fileMode:     0644,
		openMode:     0,
		seekLocation: 0,
		files:        make([]splitFile, len(paths)),
	}

	if readOnly {
		backend.openFlags = os.O_RDONLY
	} else {
		backend.openFlags = os.O_RDWR
		backend.writable = true
	}

	offset := int64(0)
	for i, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			backend.Close()
			return nil, err
		}

		file, err := os.OpenFile(path, backend.openFlags, backend.fileMode)
		if err != nil {
			backend.Close()
			return nil, err
		}

		size := info.Size()
		backend.files[i] = splitFile{
			path:   path,
			offset: offset,
			size:   size,
			file:   file,
		}

		offset += size
	}

	backend.totalSize = offset
	return backend, nil
}

// findFile finds the file containing the given offset
func (sb *SplitBackend) findFile(offset int64) (*splitFile, int64, error) {
	for i := range sb.files {
		file := &sb.files[i]
		if offset >= file.offset && offset < file.offset+file.size {
			return file, offset - file.offset, nil
		}
	}

	return nil, 0, fmt.Errorf("offset %d is out of bounds (total size: %d)", offset, sb.totalSize)
}

// Size returns the size of the virtual device
func (sb *SplitBackend) Size() int64 {
	return sb.totalSize
}

func (sb *SplitBackend) Read(p []byte) (int, error) {
	n, err := sb.ReadAt(p, sb.seekLocation)
	sb.seekLocation += int64(n)
	return n, err
}

func (sb *SplitBackend) Stat() (fs.FileInfo, error) {
	return &NandStats{
		name: sb.files[0].file.Name(),
		size: sb.totalSize,
	}, nil
}

func (sb *SplitBackend) Sys() (*os.File, error) {
	return nil, fmt.Errorf("sys not implemented for SplitBackend")
}

func (sb *SplitBackend) Writable() (backend.WritableFile, error) {
	return sb, nil
}

func (sb *SplitBackend) ReadAt(p []byte, offset int64) (int, error) {
	if offset >= sb.totalSize {
		return 0, io.EOF
	}

	totalRead := 0
	toRead := len(p)

	for totalRead < toRead {
		file, localOffset, err := sb.findFile(offset + int64(totalRead))
		if err != nil {
			// EOF or other error
			break
		}

		bytesLeftInFile := file.size - localOffset
		bytesToRead := int64(toRead - totalRead)
		if bytesToRead > bytesLeftInFile {
			bytesToRead = bytesLeftInFile
		}

		n, err := file.file.ReadAt(p[totalRead:totalRead+int(bytesToRead)], localOffset)
		totalRead += n

		if err != nil && err != io.EOF {
			return totalRead, err
		}

		if n < int(bytesToRead) || err == io.EOF {
			break
		}
	}

	return totalRead, nil
}

func (sb *SplitBackend) WriteAt(p []byte, offset int64) (int, error) {
	if sb.readOnly {
		return 0, fmt.Errorf("cannot write to read-only device")
	}

	if offset >= sb.totalSize {
		return 0, fmt.Errorf("offset %d is out of bounds (total size: %d)", offset, sb.totalSize)
	}

	totalWritten := 0
	toWrite := len(p)

	for totalWritten < toWrite {
		file, localOffset, err := sb.findFile(offset + int64(totalWritten))
		if err != nil {
			break
		}

		bytesLeftInFile := file.size - localOffset
		bytesToWrite := int64(toWrite - totalWritten)
		if bytesToWrite > bytesLeftInFile {
			bytesToWrite = bytesLeftInFile
		}

		n, err := file.file.WriteAt(p[totalWritten:totalWritten+int(bytesToWrite)], localOffset)
		totalWritten += n

		if err != nil {
			return totalWritten, err
		}

		if n < int(bytesToWrite) {
			break
		}
	}

	return totalWritten, nil
}

// Seek is not implemented and will return an error
func (sb *SplitBackend) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("seek not implemented for SplitBackend")
}

// Close closes all the files
func (sb *SplitBackend) Close() error {
	var lastErr error
	for i := range sb.files {
		if sb.files[i].file != nil {
			if err := sb.files[i].file.Close(); err != nil {
				lastErr = err
			}
			sb.files[i].file = nil
		}
	}
	return lastErr
}

// Flush syncs all files to disk
func (sb *SplitBackend) Flush() error {
	if sb.readOnly {
		return nil
	}

	var lastErr error
	for i := range sb.files {
		if sb.files[i].file != nil {
			if err := sb.files[i].file.Sync(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// Type returns the type of this implementation
func (sb *SplitBackend) Type() string {
	return "split"
}
