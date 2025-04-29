package boot_sector

import (
	"encoding/binary"
	"fmt"
)

type FatType int

const (
	Fat12 FatType = 12
	Fat16 FatType = 16
	Fat32 FatType = 32
)

// http://elm-chan.org/docs/fat_e.html
type BootSectorCommon struct {
	// Jump instruction to the bootstrap code (x86 instruction) used by OS boot process.
	// There are two type of formats for this field and the former format is prefered.
	// 0xEB, 0x??, 0x90 (Short jump + NOP)
	// 0xE9, 0x??, 0x?? (Near jump)
	// ?? is the arbitrary value depends on where to jump is.
	// In case of any format out of these formats, the volume will not be recognized by Windows.
	BS_JmpBoot [3]byte
	// "MSWIN 4.1" is recommended and "MSDOS 5.0" is often used. There are many misconceptions about this field.
	// This is only a name. Microsoft's OS does not pay any attention to this field, but some FAT drivers do some reference.
	// This string is recommended because it is considered to minimize compatibility problems.
	// You can set something else, but some FAT drivers may not recognize that volume.
	// This field usually indicates name of the system created the volume.
	BS_OEMName [8]byte
	// Sector size in unit of byte. Valid values for this field are 512, 1024, 2048 or 4096.
	// Microsoft's OS properly supports these sector sizes. Some FAT drivers assume the sector size to be 512 and do not
	// check this field. For this reason, 512 should be used for maximum compatibility. However, you should not
	// misunderstand that it is only related to compatibility. This value must be the same as the sector size of the
	// storage contains the FAT volume.
	BPB_BytsPerSec uint16
	// Number of sectors per allocation unit. In the FAT file system, the allocation unit is called Cluster.
	// This is a block of one or more consecutive sectors and the data area is managed in this unit.
	// The number of sectors per cluster must be a power of 2. Therefore, valid values are 1, 2, 4,... and 128.
	// However, any value whose cluster size (BPB_BytsPerSec * BPB_SecPerClus) exceeds 32 KB should not be used.
	// Recent systems, such as Windows, supprts cluster size larger than 32 KB, such as 64 KB, 128 KB, and 256 KB, but
	// such volumes will not be recognized correctly by MS-DOS or old disk utilities.
	BPB_SecPerClus uint8
	// Number of sectors in reserved area. This field must not be 0 because there is the boot sector itself contains
	// this BPB in the reserved area. To avoid compatibility problems, it should be 1 on FAT12/16 volume.
	// This is because some old FAT drivers ignore this field and assume that the size of reserved area is 1.
	// On the FAT32 volume, it is typically 32. Microsoft's OS properly supports any value of 1 or larger.
	BPB_RsvdSecCnt uint16
	// Number of FATs. The value of this field should always be 2.
	// Also any value eaual to or greater than 1 is valid, but it is strongly recommended not to use values other than 2
	// to avoid compatibility problem. Microsoft's FAT driver properly supports the values other than 2.
	// Some tools and FAT drivers ignore this field and assume the number of FAT to be 2.
	// The standard value for this field 2 is to provide redudancy for the FAT data.
	// The value of FAT entry is typically read from the first FAT and changes to the FAT entry are refrected to FAT copies.
	// If any sector in the FAT area is damaged, the data will not be lost because it is duplicated in another FAT.
	// Therefore it can minimize risk of data loss. On the non-disk based storages, such as memory card, such redundancy
	// is a useless feature, so that tne number of FAT may be 1 to save the disk space. However, some FAT driver might
	// not recognize such volume properly.
	BPB_NumFATs uint8
	// On the FAT12/16 volumes, this field indicates number of 32-byte directory entries in the root directory.
	// The value should be set a value that the size of root directory is aligned to the 2-sector boundary,
	// BPB_RootEntCnt * 32 becomes even multiple of BPB_BytsPerSec. For maximum compatibility, this field should be set
	// to 512 on the FAT16 volume. For FAT32 volumes, this field must be 0.
	BPB_RootEntCnt uint16
	// Total number of sectors of the volume in old 16-bit field. This value is the number of sectors including all
	// four areas of the volume. When the number of sectors of the FAT12/16 volumes is 0x10000 or larger, an invalid
	// value 0 is set in this field, and the true value is set to BPB_TotSec32. For FAT32 volumes, this field must always be 0.
	BPB_TotSec16 uint16
	// The valid values for this field is 0xF0, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE and 0xFF. 0xF8 is the standard
	// value for non-removable disks and 0xF0 is often used for non partitioned removable disks. Other important point
	// is that the same value must be put in the lower 8-bits of FAT[0]. This comes from the media determination of
	// MS-DOS Ver.1 and never used for any purpose any longer.
	BPB_Media uint8
	// Number of sectors occupied by a FAT. This field is used for only FAT12/16 volumes. On the FAT32 volumes, it must be
	// an invalid value 0 and BPB_FATSz32 is used instead. The size of the FAT area becomes BPB_FATSz?? * BPB_NumFATs sectors.
	BPB_FATSz16 uint16
	// Number of sectors per track. This field is relevant only for media that have geometry and used for only disk BIOS of IBM PC.
	BPB_SecPerTrk uint16
	// Number of heads. This field is relevant only for media that have geometry and used for only disk BIOS of IBM PC.
	BPB_NumHeads uint16
	// Number of hidden physical sectors preceding the FAT volume. It is generally related to storage accessed by disk
	// BIOS of IBM PC, and what kind of value is set is platform dependent. This field should always be 0 if the volume
	// starts at the beginning of the storage, e.g. non-partitioned disks, such as floppy disk.
	BPB_HiddSec uint32
	// Total number of sectors of the FAT volume in new 32-bit field. This value is the number of sectors including all
	// four areas of the volume. When the value on the FAT12/16 volume is less than 0x10000, this field must be invalid
	// value 0 and the true value is set to BPB_TotSec16. On the FAT32 volume, this field is always valid and old field
	// is not used.
	BPB_TotSec32 uint32
}

type BootSectorFat struct {
	// Drive number used by disk BIOS of IBM PC. This field is used in MS-DOS bootstrap, 0x00 for floppy disk and 0x80
	// for fixed disk. Actually it depends on the OS.
	BS_DrvNum uint8
	// Reserved (used by Windows NT). It should be set 0 when create the volume.
	BS_Reserved uint8
	// Extended boot signature (0x29). This is a signature byte indicates that the following three fields are present.
	BS_BootSig uint8
	// Volume serial number used with BS_VolLab to track a volume on the removable storage. It enables to detect a wrong
	// media change by FAT driver. This value is typically generated with current time and date on formatting.
	BS_VolID uint32
	// This field is the 11-byte volume label and it matches volume label recorded in the root directory.
	// FAT driver should update this field when the volume label in the root directory is changed. MS-DOS does it, but
	// Windows does not do it. When volume label is not present, "NO NAME " should be set in this field.
	BS_VolLab [11]byte
	// "FAT12   ", "FAT16   " or "FAT     ". Many people think that this string has some effect in determination of the
	// FAT type, but it is clearly a misrecognization. From the name of this field, you will find that this is not a part
	// of BPB. Since this string is often incorrect or not set, Microsoft's FAT driver does not use this field to determine
	// the FAT type. However, some old FAT drivers use this string to determine the FAT type, so that it should be set
	// based on the FAT type of the volume to avoid compatibility problems.
	BS_FilSysType [8]byte
	// Bootstrap program. It is platform dependent and filled with zero when not used.
	BS_BootCode [448]byte
	// 0xAA55. A boot signature indicating that this is a valid boot sector.
	BS_Sign uint16
}

type BootSectorFat32 struct {
	// Size of a FAT in unit of sector. The size of the FAT area is BPB_FATSz32 * BPB_NumFATs sector.
	// This is an only field needs to be referred prior to determine the FAT type while this field exists in only FAT32 volume.
	// But this is not a problem because BPB_FATSz16 is always invalid in FAT32 volume.
	BPB_FATSz32 uint32
	// Bit3-0: Active FAT starting from 0. Valid when bit7 is 1.
	// Bit6-4: Reserved (0).
	// Bit7: 0 means that each FAT are active and mirrored. 1 means that only one FAT indicated by bit3-0 is active.
	// Bit15-8-4: Reserved (0).
	BPB_ExtFlags uint16
	// FAT32 version. Upper byte is major version number and lower byte is minor version number.
	// This document describes FAT32 version 0.0. This field is for futuer extension of FAT32 volume to manage the
	// filesystem verison. However, FAT32 volume will not be updated any longer.
	BPB_FSVer uint16
	// First cluster number of the root directory. It is usually set to 2, the first cluster of the volume, but it does
	// not need to always be 2.
	BPB_RootClus uint32
	// Sector of FSInfo structure in offset from top of the FAT32 volume. It is usually set to 1, next to the boot sector.
	BPB_FSInfo uint16
	// Sector of backup boot sector in offset from top of the FAT32 volume. It is usually set to 6, next to the boot
	// sector, and any other value is not recommended.
	BPB_BkBootSec uint16
	// Reserved (0).
	BPB_Reserved [12]byte
	// Same as the description of bootSectorSmallFats
	BS_DrvNum uint8
	// Same as the description of bootSectorSmallFats
	BS_Reserved uint8
	// Same as the description of bootSectorSmallFats
	BS_BootSig uint8
	// Same as the description of bootSectorSmallFats
	BS_VolID uint32
	// Same as the description of bootSectorSmallFats
	BS_VolLab [11]byte
	// Always "FAT32   " and it does not have any effect in determination of FAT type.
	BS_FilSysType [8]byte
	// Bootstrap program. It is platform dependent and filled with zero when not used.
	BS_BootCode32 [420]byte
	// 0xAA55. A boot signature indicating that this is a valid boot sector.
	BS_Sign uint16
}

type BootSector struct {
	BootSectorCommon
	Fat   *BootSectorFat
	Fat32 *BootSectorFat32
}

func NewBootSector(b []byte) (*BootSector, error) {
	if len(b) < 512 {
		return nil, fmt.Errorf("boot sector too small")
	}

	common := BootSectorCommon{
		BS_JmpBoot:     [3]byte{},
		BS_OEMName:     [8]byte{},
		BPB_BytsPerSec: binary.LittleEndian.Uint16(b[11:13]),
		BPB_SecPerClus: b[13],
		BPB_RsvdSecCnt: binary.LittleEndian.Uint16(b[14:16]),
		BPB_NumFATs:    b[16],
		BPB_RootEntCnt: binary.LittleEndian.Uint16(b[17:19]),
		BPB_TotSec16:   binary.LittleEndian.Uint16(b[19:21]),
		BPB_Media:      b[21],
		BPB_FATSz16:    binary.LittleEndian.Uint16(b[22:24]),
		BPB_SecPerTrk:  binary.LittleEndian.Uint16(b[24:26]),
		BPB_NumHeads:   binary.LittleEndian.Uint16(b[26:28]),
		BPB_HiddSec:    binary.LittleEndian.Uint32(b[28:32]),
		BPB_TotSec32:   binary.LittleEndian.Uint32(b[32:36]),
	}
	copy(common.BS_JmpBoot[:], b[0:3])
	copy(common.BS_OEMName[:], b[3:11])

	fatsZ32 := uint32(0)
	if common.BPB_FATSz16 == 0 {
		fatsZ32 = binary.LittleEndian.Uint32(b[36:40])
	}

	bs := &BootSector{BootSectorCommon: common}

	if common.fatType(fatsZ32) == Fat32 {
		bs.Fat32 = &BootSectorFat32{
			BPB_FATSz32:   fatsZ32,
			BPB_ExtFlags:  binary.LittleEndian.Uint16(b[40:42]),
			BPB_FSVer:     binary.LittleEndian.Uint16(b[42:44]),
			BPB_RootClus:  binary.LittleEndian.Uint32(b[44:48]),
			BPB_FSInfo:    binary.LittleEndian.Uint16(b[48:50]),
			BPB_BkBootSec: binary.LittleEndian.Uint16(b[50:52]),
			BPB_Reserved:  [12]byte{},
			BS_DrvNum:     b[64],
			BS_Reserved:   b[65],
			BS_BootSig:    b[66],
			BS_VolID:      binary.LittleEndian.Uint32(b[67:71]),
			BS_VolLab:     [11]byte{},
			BS_FilSysType: [8]byte{},
			BS_BootCode32: [420]byte{},
			BS_Sign:       binary.LittleEndian.Uint16(b[510:512]),
		}
		copy(bs.Fat32.BPB_Reserved[:], b[52:64])
		copy(bs.Fat32.BS_VolLab[:], b[71:82])
		copy(bs.Fat32.BS_FilSysType[:], b[82:90])
		copy(bs.Fat32.BS_BootCode32[:], b[90:510])

		if bs.Fat32.BS_Sign != 0xAA55 {
			return nil, fmt.Errorf("invalid boot sector signature: 0x%04X", bs.Fat.BS_Sign)
		}
	} else {
		bs.Fat = &BootSectorFat{
			BS_DrvNum:     b[64],
			BS_Reserved:   b[65],
			BS_BootSig:    b[66],
			BS_VolID:      binary.LittleEndian.Uint32(b[67:71]),
			BS_VolLab:     [11]byte{},
			BS_FilSysType: [8]byte{},
			BS_BootCode:   [448]byte{},
			BS_Sign:       binary.LittleEndian.Uint16(b[510:512]),
		}
		copy(bs.Fat.BS_VolLab[:], b[71:82])
		copy(bs.Fat.BS_FilSysType[:], b[82:90])
		copy(bs.Fat.BS_BootCode[:], b[62:510])

		if bs.Fat.BS_Sign != 0xAA55 {
			return nil, fmt.Errorf("invalid boot sector signature: 0x%04X", bs.Fat.BS_Sign)
		}
	}

	return bs, nil
}

func (c *BootSectorCommon) clusterCount(fatsZ32 uint32) int64 {
	fatStartSector := uint32(c.BPB_RsvdSecCnt)
	fatSize := uint32(c.BPB_FATSz16)
	if fatSize == 0 {
		fatSize = fatsZ32
	}
	fatSectorCount := fatSize * uint32(c.BPB_NumFATs)

	rootDirStartSector := fatStartSector + fatSectorCount
	rootDirSectorCount := (32*uint32(c.BPB_RootEntCnt) + uint32(c.BPB_BytsPerSec) - 1) / uint32(c.BPB_BytsPerSec)

	totalSectors := uint32(c.BPB_TotSec16)
	if totalSectors == 0 {
		totalSectors = c.BPB_TotSec32
	}

	dataStartSector := rootDirStartSector + rootDirSectorCount
	dataSectorCount := totalSectors - dataStartSector

	return int64(dataSectorCount / uint32(c.BPB_SecPerClus))
}

// http://elm-chan.org/docs/fat_e.html#fat_determination
func (c *BootSectorCommon) fatType(fatsZ32 uint32) FatType {
	clusterCount := c.clusterCount(fatsZ32)
	if clusterCount <= 4085 {
		return Fat12
	}
	if clusterCount <= 65525 {
		return Fat16
	}

	return Fat32
}

func (c *BootSectorCommon) totalSize(fatsZ32 uint32) int64 {
	return c.clusterCount(fatsZ32) * int64(c.BPB_SecPerClus) * int64(c.BPB_BytsPerSec)
}

func (bs *BootSector) FatType() FatType {
	if bs.Fat32 != nil {
		return Fat32
	}

	return bs.fatType(0)
}

func (bs *BootSector) TotalSize() int64 {
	if bs.Fat32 != nil {
		return bs.totalSize(bs.Fat32.BPB_FATSz32)
	}

	return bs.totalSize(0)
}

func (bs *BootSector) FatSectorSize() uint32 {
	if bs.Fat32 != nil {
		return bs.Fat32.BPB_FATSz32
	}

	return uint32(bs.BPB_FATSz16)
}
