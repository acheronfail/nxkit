package nand

import (
	"strings"

	"github.com/diskfs/go-diskfs/partition/gpt"
)

const (
	SectorSize  = uint64(512)
	RawNANDSize = int64(31_268_536_320)
)

type FATFormat int

const (
	FormatNone FATFormat = iota
	FormatFAT12
	FormatFAT32
)

type PartitionSpec struct {
	Name       string
	TypeGUID   string
	PartGUID   string
	FirstLBA   uint64
	LastLBA    uint64
	Attributes uint64
	Format     FATFormat
	BISKeyID   int
	ClusterSec uint32
}

var switchPartitionSpecs = []PartitionSpec{
	{Name: "PRODINFO", TypeGUID: "98109E25-64E2-4C95-8A77-414916F5BCEB", PartGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", FirstLBA: 34, LastLBA: 8191, Attributes: 1, Format: FormatNone, BISKeyID: 0},
	{Name: "PRODINFOF", TypeGUID: "F3056AEC-5449-494C-9F2C-5FDCB75B6E6E", PartGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", FirstLBA: 8192, LastLBA: 16383, Attributes: 1, Format: FormatFAT12, BISKeyID: 0, ClusterSec: 2},
	{Name: "BCPKG2-1-Normal-Main", TypeGUID: "5365DE36-911B-4BB4-8FF9-AA1EBCD73990", PartGUID: "755272B7-445C-46A3-987B-D40E5D25EB83", FirstLBA: 16384, LastLBA: 32767, Attributes: 1, Format: FormatNone},
	{Name: "BCPKG2-2-Normal-Sub", TypeGUID: "8455717B-BD2B-4162-8454-91695218FC38", PartGUID: "EAD904D9-61A3-4DBA-BB11-6E516A1F4093", FirstLBA: 32768, LastLBA: 49151, Attributes: 1, Format: FormatNone},
	{Name: "BCPKG2-3-SafeMode-Main", TypeGUID: "8ED6C9A6-9C48-490B-BBEB-001D17A4C0F7", PartGUID: "EF78007A-D02C-4BF8-9BEF-B5B5CB3F2B76", FirstLBA: 49152, LastLBA: 65535, Attributes: 1, Format: FormatNone},
	{Name: "BCPKG2-4-SafeMode-Sub", TypeGUID: "5E99751C-56C9-47CC-AA30-B65039888917", PartGUID: "DACB7CD3-5624-41D9-85BF-DB61AE5A0096", FirstLBA: 65536, LastLBA: 81919, Attributes: 1, Format: FormatNone},
	{Name: "BCPKG2-5-Repair-Main", TypeGUID: "C447D9A2-24B7-468A-98C8-595CD077165A", PartGUID: "1C58F253-945E-4F24-95F2-29091B775F56", FirstLBA: 81920, LastLBA: 98303, Attributes: 1, Format: FormatNone},
	{Name: "BCPKG2-6-Repair-Sub", TypeGUID: "9586E1A1-3AA2-4C90-91B3-2F4A5195B4D2", PartGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", FirstLBA: 98304, LastLBA: 114687, Attributes: 1, Format: FormatNone},
	{Name: "SAFE", TypeGUID: "A44F9F6B-4ED3-441F-A34A-56AAA136BC6A", PartGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", FirstLBA: 114688, LastLBA: 245759, Attributes: 1, Format: FormatFAT32, BISKeyID: 1, ClusterSec: 1},
	{Name: "SYSTEM", TypeGUID: "ACB0CDF0-4F72-432D-AA0D-5388C733B224", PartGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", FirstLBA: 245760, LastLBA: 5488639, Attributes: 1, Format: FormatFAT32, BISKeyID: 2, ClusterSec: 32},
	{Name: "USER", TypeGUID: "2B777F63-E842-47AF-94C4-25A7F18B2280", PartGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", FirstLBA: 5488640, LastLBA: 60014591, Attributes: 1, Format: FormatFAT32, BISKeyID: 3, ClusterSec: 32},
}

func SwitchPartitionSpecs() []PartitionSpec {
	specs := make([]PartitionSpec, len(switchPartitionSpecs))
	copy(specs, switchPartitionSpecs)
	return specs
}

func MinimumSwitchGPTSize() int64 {
	var lastLBA uint64
	for _, spec := range switchPartitionSpecs {
		if spec.LastLBA > lastLBA {
			lastLBA = spec.LastLBA
		}
	}
	return int64(lastLBA+34) * int64(SectorSize)
}

// BuildSwitchGPTTable uses the canonical Switch GPP layout from createnand.
// NxNandManager repairs a missing backup GPT by rebuilding it from the primary
// header and partition entries; diskfs writes both GPT copies from this table.
func BuildSwitchGPTTable(existing *gpt.Table) *gpt.Table {
	diskGUID := "EDD7049E-B2D3-4067-B3D9-E5A8F398258F"
	existingPartitionGUIDs := map[string]string{}
	if existing != nil {
		if existing.GUID != "" {
			diskGUID = existing.GUID
		}
		for _, part := range existing.Partitions {
			if part == nil || part.GUID == "" {
				continue
			}
			for _, spec := range switchPartitionSpecs {
				if part.Name == spec.Name && strings.EqualFold(string(part.Type), spec.TypeGUID) {
					existingPartitionGUIDs[spec.Name] = part.GUID
					break
				}
			}
		}
	}

	table := &gpt.Table{
		LogicalSectorSize:  int(SectorSize),
		PhysicalSectorSize: int(SectorSize),
		GUID:               diskGUID,
		ProtectiveMBR:      true,
		Partitions:         make([]*gpt.Partition, 0, len(switchPartitionSpecs)),
	}
	for i, spec := range switchPartitionSpecs {
		partGUID := spec.PartGUID
		if existingGUID, ok := existingPartitionGUIDs[spec.Name]; ok {
			partGUID = existingGUID
		}
		table.Partitions = append(table.Partitions, &gpt.Partition{
			Index:      i + 1,
			Start:      spec.FirstLBA,
			End:        spec.LastLBA,
			Type:       gpt.Type(spec.TypeGUID),
			Name:       spec.Name,
			GUID:       partGUID,
			Attributes: spec.Attributes,
		})
	}
	return table
}
