package keys_test

import (
	"testing"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/stretchr/testify/assert"
)

func TestKeys(t *testing.T) {
	_, err := keys.NewFromPath("../../../.data/prod.keys")
	assert.Nil(t, err)
	// TODO: better tests
}
