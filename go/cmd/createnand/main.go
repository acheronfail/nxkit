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
)

const (
	numFATs       = uint32(2)
	fat32Reserved = uint32(32)
)

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
	if err := file.Truncate(nand.RawNANDSize); err != nil {
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

	table := nand.BuildSwitchGPTTable(nil)
	if err := table.Write(file, nand.RawNANDSize); err != nil {
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

	for _, spec := range nand.SwitchPartitionSpecs() {
		if spec.Format == nand.FormatNone {
			continue
		}
		fmt.Printf("Formatting %s...\n", spec.Name)
		partStart := spec.FirstLBA * nand.SectorSize
		partEnd := (spec.LastLBA + 1) * nand.SectorSize
		part := nand.NewNxPartBackend(raw, partStart, partEnd, 16, nand.SectorSize)
		if !clear {
			bisKey := keyset.BisKey[spec.BISKeyID]
			crypto, err := xtsn.NewXtsnCipher(bisKey[:16], bisKey[16:], 16*1024)
			if err != nil {
				return fmt.Errorf("failed to create BIS crypto for %s: %w", spec.Name, err)
			}
			part.SetCrypto(crypto)
		}

		switch spec.Format {
		case nand.FormatFAT12:
			err = formatFAT12Partition(part, spec)
		case nand.FormatFAT32:
			err = formatFAT32Partition(part, spec)
		default:
			err = fmt.Errorf("unsupported format %d", spec.Format)
		}
		if err != nil {
			return fmt.Errorf("failed to format %s: %w", spec.Name, err)
		}
	}
	return nil
}

func formatFAT12Partition(part backend.WritableFile, spec nand.PartitionSpec) error {
	totalSectors := uint32(spec.LastLBA - spec.FirstLBA + 1)
	rootEntries := uint16(512)
	rootDirSectors := uint32((uint32(rootEntries)*32 + uint32(nand.SectorSize) - 1) / uint32(nand.SectorSize))
	reservedSectors := uint32(1)
	fatSectors, clusterCount := fat12Sectors(totalSectors, reservedSectors, rootDirSectors, spec.ClusterSec)
	if clusterCount > 4085 {
		return fmt.Errorf("FAT12 cluster count %d exceeds FAT12 limit", clusterCount)
	}

	boot := make([]byte, nand.SectorSize)
	fillCommonBootSector(boot, spec.ClusterSec, reservedSectors, rootEntries, totalSectors, fatSectors, 0)
	copy(boot[0:3], []byte{0xeb, 0x3c, 0x90})
	boot[36] = 0x80
	boot[38] = 0x29
	binary.LittleEndian.PutUint32(boot[39:43], 0)
	copy(boot[43:54], []byte("NO NAME    "))
	copy(boot[54:62], []byte("FAT12   "))
	binary.LittleEndian.PutUint16(boot[510:512], 0xaa55)
	if err := writeFull(part, int64(spec.FirstLBA*nand.SectorSize), boot); err != nil {
		return err
	}

	fat := make([]byte, uint64(fatSectors)*nand.SectorSize)
	copy(fat[:3], []byte{0xf8, 0xff, 0xff})
	fatOffset := (spec.FirstLBA + uint64(reservedSectors)) * nand.SectorSize
	for i := uint32(0); i < numFATs; i++ {
		if err := writeFull(part, int64(fatOffset+uint64(i)*uint64(fatSectors)*nand.SectorSize), fat); err != nil {
			return err
		}
	}

	rootOffset := (spec.FirstLBA + uint64(reservedSectors+numFATs*fatSectors)) * nand.SectorSize
	root := make([]byte, uint64(rootDirSectors)*nand.SectorSize)
	return writeFull(part, int64(rootOffset), root)
}

func formatFAT32Partition(part backend.WritableFile, spec nand.PartitionSpec) error {
	totalSectors := uint32(spec.LastLBA - spec.FirstLBA + 1)
	fatSectors, clusterCount := fat32Sectors(totalSectors, fat32Reserved, spec.ClusterSec)
	if clusterCount <= 65525 {
		return fmt.Errorf("FAT32 cluster count %d is too small", clusterCount)
	}

	boot := make([]byte, nand.SectorSize)
	fillCommonBootSector(boot, spec.ClusterSec, fat32Reserved, 0, totalSectors, 0, fatSectors)
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
	bootOffset := spec.FirstLBA * nand.SectorSize
	for _, sector := range []uint64{0, 6} {
		if err := writeFull(part, int64(bootOffset+sector*nand.SectorSize), boot); err != nil {
			return err
		}
	}
	for _, sector := range []uint64{1, 7} {
		if err := writeFull(part, int64(bootOffset+sector*nand.SectorSize), fsinfo); err != nil {
			return err
		}
	}

	fat := make([]byte, uint64(fatSectors)*nand.SectorSize)
	binary.LittleEndian.PutUint32(fat[0:4], 0x0ffffff8)
	binary.LittleEndian.PutUint32(fat[4:8], 0xffffffff)
	binary.LittleEndian.PutUint32(fat[8:12], 0x0fffffff)
	fatOffset := (spec.FirstLBA + uint64(fat32Reserved)) * nand.SectorSize
	for i := uint32(0); i < numFATs; i++ {
		if err := writeFull(part, int64(fatOffset+uint64(i)*uint64(fatSectors)*nand.SectorSize), fat); err != nil {
			return err
		}
	}

	dataStart := spec.FirstLBA + uint64(fat32Reserved+numFATs*fatSectors)
	root := make([]byte, uint64(spec.ClusterSec)*nand.SectorSize)
	return writeFull(part, int64(dataStart*nand.SectorSize), root)
}

func fillCommonBootSector(boot []byte, sectorsPerCluster, reservedSectors uint32, rootEntries uint16, totalSectors, fatSectors16, fatSectors32 uint32) {
	copy(boot[3:11], []byte("MSWIN4.1"))
	binary.LittleEndian.PutUint16(boot[11:13], uint16(nand.SectorSize))
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
	fsinfo := make([]byte, nand.SectorSize)
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
		next := divRoundUp(bytesPerFAT, uint32(nand.SectorSize))
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
		next := divRoundUp((clusters+2)*4, uint32(nand.SectorSize))
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
