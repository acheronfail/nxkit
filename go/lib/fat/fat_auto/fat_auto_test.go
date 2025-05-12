package fat_auto_test

import (
	"testing"

	"github.com/acheronfail/nxkit/lib/fat/backend/file"
	"github.com/acheronfail/nxkit/lib/fat/fat_auto"
	"github.com/acheronfail/nxkit/lib/fat/testdata"
	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	backend, err := file.OpenFromPath(testdata.GetFatDiskImagePath(), false)
	assert.Nil(t, err)

	fs, err := fat_auto.Open(backend, 0)
	assert.Nil(t, err)
	assert.Equal(t, testdata.TestFatType, fs.GetType())
}
