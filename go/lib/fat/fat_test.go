package fat_test

import (
	"testing"

	"github.com/acheronfail/nxkit/lib/fat/fat16"
	"github.com/acheronfail/nxkit/lib/fat/testdata"
	"github.com/stretchr/testify/assert"
)

func TestThings(t *testing.T) {
	fs, err := fat16.NewFromPath(testdata.Fat16DiskImagePath)
	assert.Nil(t, err)
	defer fs.Close()
}
