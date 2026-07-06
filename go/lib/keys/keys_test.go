package keys_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	_, err := keys.NewFromPath("../../../.data/prod.keys")
	assert.Nil(t, err)
	// TODO: better tests
}

func TestPrefixedKeysUseEncodedIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prod.keys")
	require.NoError(t, os.WriteFile(path, []byte(`
header_key = 0000000000000000000000000000000000000000000000000000000000000000
key_area_key_application_02 = 02020202020202020202020202020202
key_area_key_application_05 = 05050505050505050505050505050505
`), 0o644))

	keyset, err := keys.NewFromPath(path)
	require.NoError(t, err)

	key, err := keyset.GetKeyAreaKey(0, 5)
	require.NoError(t, err)
	assert.Equal(t, []byte{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}, key)

	_, err = keyset.GetKeyAreaKey(0, 0)
	assert.Error(t, err)
}
