package nand

import (
	"fmt"

	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/partition/gpt"
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

// TODO: return err don't panic
func Open(path string) {
	backend, err := NewCombinedDumpBackend(path)
	if err != nil {
		panic(err)
	}
	defer backend.Close()

	// TODO: verify block sizes here
	gptTable, err := gpt.Read(backend, 512, 512)
	if err != nil {
		panic(err)
	}

	var userPartition *gpt.Partition
	for i, part := range gptTable.Partitions {
		if part.Name == "USER" {
			userPartition = gptTable.Partitions[i]
			break
		}
	}

	if userPartition == nil {
		panic("USER partition not found")
	}

	// TODO: read encrypted partitions
	fmt.Printf("Found USER partition: %+v\n", userPartition)
}
