package romfs

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// TODO: build a romfs: see hacbrewpack's `build_romfs_into_file`

func Build(inDirectory string, outputPath string) error {
	filepath.WalkDir(inDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// TODO: build into romfs
		fmt.Println("walk dir", path, d.IsDir())
		return nil
	})

	panic("unimplemented")
}
