package npdm

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	MAGIC_META = 0x4154454D // 'META'
	MAGIC_ACID = 0x44494341 // 'ACID'
	MAGIC_ACI0 = 0x30494341 // 'ACI0'
)

type Npdm struct {
	Magic           uint32
	_               uint32
	_               uint32
	MmuFlags        uint8
	_               uint8
	MainThreadPrio  uint8
	DefaultCpuID    uint8
	_               uint64
	ProcessCategory uint32
	MainStackSize   uint32
	TitleName       [0x50]byte
	Aci0Offset      uint32
	Aci0Size        uint32
	AcidOffset      uint32
	AcidSize        uint32
}

type NpdmAcid struct {
	Signature       [0x100]byte
	Modulus         [0x100]byte
	Magic           uint32
	Size            uint32
	_               uint32
	Flags           uint32
	TitleIDRangeMin uint64
	TitleIDRangeMax uint64
	FacOffset       uint32
	FacSize         uint32
	SacOffset       uint32
	SacSize         uint32
	KacOffset       uint32
	KacSize         uint32
	Padding         uint64
}

type NpdmAci0 struct {
	Magic     uint32
	_         [0xC]byte
	TitleID   uint64
	_         uint64
	FahOffset uint32
	FahSize   uint32
	SacOffset uint32
	SacSize   uint32
	KacOffset uint32
	KacSize   uint32
	Padding   uint64
}

func Process(exefsDir, backupDir string, titleId uint64, noSignNcaSignature bool) (uint64, error) {
	return ProcessWithPublicKeyPath(exefsDir, backupDir, titleId, noSignNcaSignature, filepath.Join("keys", "hacbrewpack.pub.pem"))
}

func ProcessWithPublicKeyPath(exefsDir, backupDir string, titleId uint64, noSignNcaSignature bool, pubKeyPath string) (uint64, error) {
	npdmPath := filepath.Join(exefsDir, "main.npdm")
	f, err := os.OpenFile(npdmPath, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open %s: %w", npdmPath, err)
	}
	defer f.Close()

	var npdm Npdm
	if err := binary.Read(f, binary.LittleEndian, &npdm); err != nil {
		return 0, fmt.Errorf("failed to read NPDM header: %w", err)
	}

	if npdm.Magic != MAGIC_META {
		return 0, fmt.Errorf("invalid NPDM magic")
	}

	// Read ACID
	var acid NpdmAcid
	if _, err := f.Seek(int64(npdm.AcidOffset), 0); err != nil {
		return 0, fmt.Errorf("failed to seek to NPDM ACID: %w", err)
	}
	if err := binary.Read(f, binary.LittleEndian, &acid); err != nil {
		return 0, fmt.Errorf("failed to read NPDM ACID: %w", err)
	}
	if acid.Magic != MAGIC_ACID {
		return 0, fmt.Errorf("invalid ACID magic")
	}

	// Read ACI0
	var aci0 NpdmAci0
	if _, err := f.Seek(int64(npdm.Aci0Offset), 0); err != nil {
		return 0, fmt.Errorf("failed to seek to NPDM ACI0: %w", err)
	}
	if err := binary.Read(f, binary.LittleEndian, &aci0); err != nil {
		return 0, fmt.Errorf("failed to read NPDM ACI0: %w", err)
	}
	if aci0.Magic != MAGIC_ACI0 {
		return 0, fmt.Errorf("invalid ACI0 magic")
	}

	// Determine TitleID
	var tid uint64
	if titleId == 0 {
		tid = aci0.TitleID
	} else {
		tid = titleId
	}

	fmt.Printf("Validating TitleID: 0x%016x\n", tid)
	if tid < 0x0100000000000000 || tid > 0x0fffffffffffffff {
		return 0, fmt.Errorf("bad TitleID found in main.npdm: 0x%016x. Valid range: 0100000000000000 - 0fffffffffffffff", tid)
	}
	if tid > 0x01ffffffffffffff {
		fmt.Printf("Warning: TitleID %016x is greater than 01ffffffffffffff and it's not suggested\n", tid)
	}

	// Patch TitleID if provided
	if tid != 0 {
		if _, err := f.Seek(int64(npdm.Aci0Offset+0x10), 0); err != nil {
			return 0, fmt.Errorf("seek error while patching title ID: %w", err)
		}
		if err := binary.Write(f, binary.LittleEndian, tid); err != nil {
			return 0, fmt.Errorf("write error while patching title ID: %w", err)
		}
	}

	if !noSignNcaSignature {
		fmt.Println("Backing up main.npdm")
		timestamp := time.Now().Unix()
		backupName := fmt.Sprintf("%d_main.npdm", timestamp)
		backupPath := filepath.Join(backupDir, backupName)
		if err := copyFile(npdmPath, backupPath); err != nil {
			return 0, fmt.Errorf("failed to backup main.npdm: %w", err)
		}

		fmt.Println("Patching ACID public key")
		if _, err := f.Seek(int64(npdm.AcidOffset+0x100), 0); err != nil {
			return 0, fmt.Errorf("seek error while patching public key: %w", err)
		}

		pubKey, err := getPublicKeyBytes(pubKeyPath)
		if err != nil {
			return 0, fmt.Errorf("failed to get public key bytes: %w", err)
		}
		if _, err := f.Write(pubKey); err != nil {
			return 0, fmt.Errorf("write error while patching public key: %w", err)
		}
	}

	return tid, nil
}

func getPublicKeyBytes(pubKeyPath string) ([]byte, error) {
	pemData, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	pub, err := parseRSAPublicKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	modulus := pub.N.Bytes()
	if len(modulus) > 0x100 {
		return nil, fmt.Errorf("RSA key modulus too large: %d bytes", len(modulus))
	}

	// Left-pad the modulus with zeros to ensure it's 256 bytes
	padded := make([]byte, 0x100)
	copy(padded[0x100-len(modulus):], modulus)
	return padded, nil
}

func parseRSAPublicKey(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	if block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("invalid PEM block type: %s", block.Type)
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}

	if err := os.WriteFile(dst, input, 0644); err != nil {
		return fmt.Errorf("failed to write backup file %s: %w", dst, err)
	}

	return nil
}
