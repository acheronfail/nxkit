package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acheronfail/nxkit/lib/fat/backend"
	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/nand"
	"github.com/acheronfail/nxkit/lib/tools"
	"github.com/acheronfail/nxkit/lib/xtsn"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

const (
	sectorSize    = uint64(512)
	rawnandSize   = int64(31_268_536_320)
	numFATs       = uint32(2)
	fat32Reserved = uint32(32)
)

type fatFormat int

const (
	formatNone fatFormat = iota
	formatFAT12
	formatFAT32
)

type partitionSpec struct {
	name       string
	typeGUID   string
	partGUID   string
	firstLBA   uint64
	lastLBA    uint64
	attributes uint64
	format     fatFormat
	bisKeyID   int
	clusterSec uint32
}

var partitions = []partitionSpec{
	{name: "PRODINFO", typeGUID: "98109E25-64E2-4C95-8A77-414916F5BCEB", partGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", firstLBA: 34, lastLBA: 8191, attributes: 1, format: formatNone, bisKeyID: 0},
	{name: "PRODINFOF", typeGUID: "F3056AEC-5449-494C-9F2C-5FDCB75B6E6E", partGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", firstLBA: 8192, lastLBA: 16383, attributes: 1, format: formatFAT12, bisKeyID: 0, clusterSec: 2},
	{name: "BCPKG2-1-Normal-Main", typeGUID: "5365DE36-911B-4BB4-8FF9-AA1EBCD73990", partGUID: "755272B7-445C-46A3-987B-D40E5D25EB83", firstLBA: 16384, lastLBA: 32767, attributes: 1, format: formatNone},
	{name: "BCPKG2-2-Normal-Sub", typeGUID: "8455717B-BD2B-4162-8454-91695218FC38", partGUID: "EAD904D9-61A3-4DBA-BB11-6E516A1F4093", firstLBA: 32768, lastLBA: 49151, attributes: 1, format: formatNone},
	{name: "BCPKG2-3-SafeMode-Main", typeGUID: "8ED6C9A6-9C48-490B-BBEB-001D17A4C0F7", partGUID: "EF78007A-D02C-4BF8-9BEF-B5B5CB3F2B76", firstLBA: 49152, lastLBA: 65535, attributes: 1, format: formatNone},
	{name: "BCPKG2-4-SafeMode-Sub", typeGUID: "5E99751C-56C9-47CC-AA30-B65039888917", partGUID: "DACB7CD3-5624-41D9-85BF-DB61AE5A0096", firstLBA: 65536, lastLBA: 81919, attributes: 1, format: formatNone},
	{name: "BCPKG2-5-Repair-Main", typeGUID: "C447D9A2-24B7-468A-98C8-595CD077165A", partGUID: "1C58F253-945E-4F24-95F2-29091B775F56", firstLBA: 81920, lastLBA: 98303, attributes: 1, format: formatNone},
	{name: "BCPKG2-6-Repair-Sub", typeGUID: "9586E1A1-3AA2-4C90-91B3-2F4A5195B4D2", partGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", firstLBA: 98304, lastLBA: 114687, attributes: 1, format: formatNone},
	{name: "SAFE", typeGUID: "A44F9F6B-4ED3-441F-A34A-56AAA136BC6A", partGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", firstLBA: 114688, lastLBA: 245759, attributes: 1, format: formatFAT32, bisKeyID: 1, clusterSec: 1},
	{name: "SYSTEM", typeGUID: "ACB0CDF0-4F72-432D-AA0D-5388C733B224", partGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", firstLBA: 245760, lastLBA: 5488639, attributes: 1, format: formatFAT32, bisKeyID: 2, clusterSec: 32},
	{name: "USER", typeGUID: "2B777F63-E842-47AF-94C4-25A7F18B2280", partGUID: "5561E2D3-9B30-4D80-A546-10EB7C0151FC", firstLBA: 5488640, lastLBA: 60014591, attributes: 1, format: formatFAT32, bisKeyID: 3, clusterSec: 32},
}

func main() {
	keysPath := flag.String("keys", filepath.Join(".data", "prod.keys"), "path to prod.keys")
	outPath := flag.String("out", filepath.Join(".data", "rawnand.bin"), "path to write the NAND image")
	clear := flag.Bool("clear", false, "format partitions without BIS encryption")
	splitDump := flag.Bool("split", false, "split the generated dump in place")
	flag.Parse()

	if err := run(*keysPath, *outPath, *clear, *splitDump); err != nil {
		fmt.Fprintf(os.Stderr, "create nand: %v\n", err)
		os.Exit(1)
	}
}

func run(keysPath, outPath string, clear, splitDump bool) error {
	keyset, err := keys.NewFromPath(keysPath)
	if err != nil {
		return fmt.Errorf("failed to read keys from %s: %w", keysPath, err)
	}
	if !clear {
		for _, id := range []int{0, 1, 2, 3} {
			if id >= len(keyset.BisKey) || len(keyset.BisKey[id]) != 32 {
				return fmt.Errorf("bis_key_%02d must be present and 32 bytes", id)
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := writeSparseNAND(outPath); err != nil {
		return err
	}
	if err := writeGPT(outPath); err != nil {
		return err
	}
	if err := formatPartitions(outPath, keyset, clear); err != nil {
		return err
	}

	if splitDump {
		if _, err := tools.Split(outPath, false, true, 0); err != nil {
			return fmt.Errorf("failed to split %s: %w", outPath, err)
		}
	}
	return nil
}

func writeSparseNAND(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer file.Close()
	if err := file.Truncate(rawnandSize); err != nil {
		return fmt.Errorf("failed to size %s: %w", path, err)
	}
	return nil
}

func writeGPT(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open %s for GPT writing: %w", path, err)
	}
	defer file.Close()

	table := &gpt.Table{
		LogicalSectorSize:  512,
		PhysicalSectorSize: 512,
		GUID:               "EDD7049E-B2D3-4067-B3D9-E5A8F398258F",
		ProtectiveMBR:      true,
		Partitions:         make([]*gpt.Partition, 0, len(partitions)),
	}
	for _, spec := range partitions {
		table.Partitions = append(table.Partitions, &gpt.Partition{
			Start:      spec.firstLBA,
			End:        spec.lastLBA,
			Type:       gpt.Type(spec.typeGUID),
			Name:       spec.name,
			GUID:       spec.partGUID,
			Attributes: spec.attributes,
		})
	}
	if err := table.Write(file, rawnandSize); err != nil {
		return fmt.Errorf("failed to write GPT: %w", err)
	}
	return nil
}

func formatPartitions(path string, keyset *keys.Keys, clear bool) error {
	raw, err := nand.NewDumpBackend(path, false)
	if err != nil {
		return fmt.Errorf("failed to open %s for formatting: %w", path, err)
	}
	defer raw.Close()

	for _, spec := range partitions {
		if spec.format == formatNone {
			continue
		}
		fmt.Printf("Formatting %s...\n", spec.name)
		partStart := spec.firstLBA * sectorSize
		partEnd := (spec.lastLBA + 1) * sectorSize
		part := nand.NewNxPartBackend(raw, partStart, partEnd, 16, sectorSize)
		if !clear {
			bisKey := keyset.BisKey[spec.bisKeyID]
			crypto, err := xtsn.NewXtsnCipher(bisKey[:16], bisKey[16:], 16*1024)
			if err != nil {
				return fmt.Errorf("failed to create BIS crypto for %s: %w", spec.name, err)
			}
			part.SetCrypto(crypto)
		}

		switch spec.format {
		case formatFAT12:
			err = formatFAT12Partition(part, spec)
		case formatFAT32:
			err = formatFAT32Partition(part, spec)
		default:
			err = fmt.Errorf("unsupported format %d", spec.format)
		}
		if err != nil {
			return fmt.Errorf("failed to format %s: %w", spec.name, err)
		}
	}
	return nil
}

func formatFAT12Partition(part backend.WritableFile, spec partitionSpec) error {
	totalSectors := uint32(spec.lastLBA - spec.firstLBA + 1)
	rootEntries := uint16(512)
	rootDirSectors := uint32((uint32(rootEntries)*32 + uint32(sectorSize) - 1) / uint32(sectorSize))
	reservedSectors := uint32(1)
	fatSectors, clusterCount := fat12Sectors(totalSectors, reservedSectors, rootDirSectors, spec.clusterSec)
	if clusterCount > 4085 {
		return fmt.Errorf("FAT12 cluster count %d exceeds FAT12 limit", clusterCount)
	}

	boot := make([]byte, sectorSize)
	fillCommonBootSector(boot, spec.clusterSec, reservedSectors, rootEntries, totalSectors, fatSectors, 0)
	copy(boot[0:3], []byte{0xeb, 0x3c, 0x90})
	boot[36] = 0x80
	boot[38] = 0x29
	binary.LittleEndian.PutUint32(boot[39:43], 0)
	copy(boot[43:54], []byte("NO NAME    "))
	copy(boot[54:62], []byte("FAT12   "))
	binary.LittleEndian.PutUint16(boot[510:512], 0xaa55)
	if err := writeFull(part, int64(spec.firstLBA*sectorSize), boot); err != nil {
		return err
	}

	fat := make([]byte, uint64(fatSectors)*sectorSize)
	copy(fat[:3], []byte{0xf8, 0xff, 0xff})
	fatOffset := (spec.firstLBA + uint64(reservedSectors)) * sectorSize
	for i := uint32(0); i < numFATs; i++ {
		if err := writeFull(part, int64(fatOffset+uint64(i)*uint64(fatSectors)*sectorSize), fat); err != nil {
			return err
		}
	}

	rootOffset := (spec.firstLBA + uint64(reservedSectors+numFATs*fatSectors)) * sectorSize
	root := make([]byte, uint64(rootDirSectors)*sectorSize)
	return writeFull(part, int64(rootOffset), root)
}

func formatFAT32Partition(part backend.WritableFile, spec partitionSpec) error {
	totalSectors := uint32(spec.lastLBA - spec.firstLBA + 1)
	fatSectors, clusterCount := fat32Sectors(totalSectors, fat32Reserved, spec.clusterSec)
	if clusterCount <= 65525 {
		return fmt.Errorf("FAT32 cluster count %d is too small", clusterCount)
	}

	boot := make([]byte, sectorSize)
	fillCommonBootSector(boot, spec.clusterSec, fat32Reserved, 0, totalSectors, 0, fatSectors)
	copy(boot[0:3], []byte{0xeb, 0x58, 0x90})
	binary.LittleEndian.PutUint16(boot[40:42], 0)
	binary.LittleEndian.PutUint16(boot[42:44], 0)
	binary.LittleEndian.PutUint32(boot[44:48], 2)
	binary.LittleEndian.PutUint16(boot[48:50], 1)
	binary.LittleEndian.PutUint16(boot[50:52], 6)
	boot[64] = 0x80
	boot[66] = 0x29
	binary.LittleEndian.PutUint32(boot[67:71], 0)
	copy(boot[71:82], []byte("NO NAME    "))
	copy(boot[82:90], []byte("FAT32   "))
	binary.LittleEndian.PutUint16(boot[510:512], 0xaa55)

	fsinfo := makeFSInfo(clusterCount - 1)
	bootOffset := spec.firstLBA * sectorSize
	for _, sector := range []uint64{0, 6} {
		if err := writeFull(part, int64(bootOffset+sector*sectorSize), boot); err != nil {
			return err
		}
	}
	for _, sector := range []uint64{1, 7} {
		if err := writeFull(part, int64(bootOffset+sector*sectorSize), fsinfo); err != nil {
			return err
		}
	}

	fat := make([]byte, uint64(fatSectors)*sectorSize)
	binary.LittleEndian.PutUint32(fat[0:4], 0x0ffffff8)
	binary.LittleEndian.PutUint32(fat[4:8], 0xffffffff)
	binary.LittleEndian.PutUint32(fat[8:12], 0x0fffffff)
	fatOffset := (spec.firstLBA + uint64(fat32Reserved)) * sectorSize
	for i := uint32(0); i < numFATs; i++ {
		if err := writeFull(part, int64(fatOffset+uint64(i)*uint64(fatSectors)*sectorSize), fat); err != nil {
			return err
		}
	}

	dataStart := spec.firstLBA + uint64(fat32Reserved+numFATs*fatSectors)
	root := make([]byte, uint64(spec.clusterSec)*sectorSize)
	return writeFull(part, int64(dataStart*sectorSize), root)
}

func fillCommonBootSector(boot []byte, sectorsPerCluster, reservedSectors uint32, rootEntries uint16, totalSectors, fatSectors16, fatSectors32 uint32) {
	copy(boot[3:11], []byte("MSWIN4.1"))
	binary.LittleEndian.PutUint16(boot[11:13], uint16(sectorSize))
	boot[13] = byte(sectorsPerCluster)
	binary.LittleEndian.PutUint16(boot[14:16], uint16(reservedSectors))
	boot[16] = byte(numFATs)
	binary.LittleEndian.PutUint16(boot[17:19], rootEntries)
	if totalSectors < 0x10000 {
		binary.LittleEndian.PutUint16(boot[19:21], uint16(totalSectors))
	} else {
		binary.LittleEndian.PutUint32(boot[32:36], totalSectors)
	}
	boot[21] = 0xf8
	binary.LittleEndian.PutUint16(boot[22:24], uint16(fatSectors16))
	binary.LittleEndian.PutUint16(boot[24:26], 63)
	binary.LittleEndian.PutUint16(boot[26:28], 255)
	binary.LittleEndian.PutUint32(boot[36:40], fatSectors32)
}

func makeFSInfo(freeClusters uint32) []byte {
	fsinfo := make([]byte, sectorSize)
	copy(fsinfo[0:4], []byte{0x52, 0x52, 0x61, 0x41})
	copy(fsinfo[484:488], []byte{0x72, 0x72, 0x41, 0x61})
	binary.LittleEndian.PutUint32(fsinfo[488:492], freeClusters)
	binary.LittleEndian.PutUint32(fsinfo[492:496], 2)
	copy(fsinfo[508:512], []byte{0x00, 0x00, 0x55, 0xaa})
	return fsinfo
}

func fat12Sectors(totalSectors, reservedSectors, rootDirSectors, sectorsPerCluster uint32) (uint32, uint32) {
	fatSectors := uint32(1)
	var clusters uint32
	for {
		dataSectors := totalSectors - reservedSectors - rootDirSectors - numFATs*fatSectors
		clusters = dataSectors / sectorsPerCluster
		bytesPerFAT := ((clusters+2)*3 + 1) / 2
		next := divRoundUp(bytesPerFAT, uint32(sectorSize))
		if next <= fatSectors {
			return fatSectors, clusters
		}
		fatSectors = next
	}
}

func fat32Sectors(totalSectors, reservedSectors, sectorsPerCluster uint32) (uint32, uint32) {
	fatSectors := uint32(1)
	var clusters uint32
	for {
		dataSectors := totalSectors - reservedSectors - numFATs*fatSectors
		clusters = dataSectors / sectorsPerCluster
		next := divRoundUp((clusters+2)*4, uint32(sectorSize))
		if next <= fatSectors {
			return fatSectors, clusters
		}
		fatSectors = next
	}
}

func divRoundUp(n, d uint32) uint32 {
	return (n + d - 1) / d
}

func writeFull(w backend.WritableFile, offset int64, data []byte) error {
	n, err := w.WriteAt(data, offset)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("short write at %d: wrote %d bytes, expected %d", offset, n, len(data))
	}
	return nil
}
