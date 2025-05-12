package fat_auto

import (
	"fmt"

	"github.com/acheronfail/nxkit/lib/fat"
	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/fat/boot_sector"
	"github.com/acheronfail/nxkit/lib/fat/fat12"
	"github.com/acheronfail/nxkit/lib/fat/fat16"
	"github.com/acheronfail/nxkit/lib/fat/fat32"
	"github.com/acheronfail/nxkit/lib/fat/internal"
)

func Open(backend backend.Storage, offset int64) (fat.FileSystem, error) {
	bootSector, err := internal.ReadBootSector(backend, offset)
	if err != nil {
		return nil, err
	}

	switch bootSector.FatType() {
	case boot_sector.Fat12:
		return fat12.Open(backend, offset)
	case boot_sector.Fat16:
		return fat16.Open(backend, offset)
	case boot_sector.Fat32:
		return fat32.Open(backend, offset)
	default:
		return nil, fmt.Errorf("unsupported filesystem type")
	}
}
