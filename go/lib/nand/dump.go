package nand

import (
	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/backend/file"
)

// TODO: support split dumps
type CombinedBackend struct {
	backend.Storage
}

func NewCombinedDumpBackend(path string) (backend.Storage, error) {
	// TODO: support writing
	readOnly := true
	storage, err := file.OpenFromPath(path, readOnly)
	if err != nil {
		return nil, err
	}

	return storage, nil
}
