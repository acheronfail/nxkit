package nand

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/stretchr/testify/require"
)

func TestBuildSwitchGPTTablePreservesExistingGUIDs(t *testing.T) {
	const diskGUID = "11111111-2222-3333-4444-555555555555"
	const userGUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	existing := &gpt.Table{
		GUID: diskGUID,
		Partitions: []*gpt.Partition{
			{
				Name: "USER",
				Type: gpt.Type("2B777F63-E842-47AF-94C4-25A7F18B2280"),
				GUID: userGUID,
			},
		},
	}

	table := BuildSwitchGPTTable(existing)
	require.Equal(t, diskGUID, table.GUID)

	var user *gpt.Partition
	for _, part := range table.Partitions {
		if part.Name == "USER" {
			user = part
			break
		}
	}
	require.NotNil(t, user)
	require.Equal(t, userGUID, user.GUID)
	require.Equal(t, uint64(5488640), user.Start)
	require.Equal(t, uint64(60014591), user.End)
}

func TestBuildSwitchGPTTableWritesVerifiableGPT(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rawnand.bin")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	require.NoError(t, err)
	defer file.Close()
	require.NoError(t, file.Truncate(RawNANDSize))

	table := BuildSwitchGPTTable(nil)
	require.NoError(t, table.Write(file, RawNANDSize))

	readBack, err := gpt.Read(file, int(SectorSize), int(SectorSize))
	require.NoError(t, err)
	require.NoError(t, readBack.Verify(file, uint64(RawNANDSize)))
	require.Len(t, readBack.Partitions, len(SwitchPartitionSpecs()))
}
