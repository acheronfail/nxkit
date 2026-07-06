package hacbrewpack

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestOptionsSetDefaults(t *testing.T) {
	root := t.TempDir()
	opt := Options{RepoRoot: root}

	if err := opt.setDefaults(); err != nil {
		t.Fatalf("setDefaults failed: %v", err)
	}

	if opt.TitleID != defaultTitleID {
		t.Fatalf("TitleID = %016x, want %016x", opt.TitleID, defaultTitleID)
	}
	if opt.SDKVersion != defaultSDKVersion {
		t.Fatalf("SDKVersion = %08x, want %08x", opt.SDKVersion, defaultSDKVersion)
	}
	if opt.KeyGeneration != 1 {
		t.Fatalf("KeyGeneration = %d, want 1", opt.KeyGeneration)
	}
	if !bytes.Equal(opt.KeyAreaKey, bytes.Repeat([]byte{0x04}, 16)) {
		t.Fatalf("unexpected default KeyAreaKey: %x", opt.KeyAreaKey)
	}
	if opt.KeysPath != filepath.Join(root, ".data", "prod.keys") {
		t.Fatalf("KeysPath = %s", opt.KeysPath)
	}
}

func TestOptionsSetDefaultsValidation(t *testing.T) {
	for name, opt := range map[string]Options{
		"missing repo root":     {},
		"old sdk version":       {RepoRoot: t.TempDir(), SDKVersion: 0x000a0000},
		"low keygeneration":     {RepoRoot: t.TempDir(), KeyGeneration: -1},
		"high keygeneration":    {RepoRoot: t.TempDir(), KeyGeneration: 33},
		"short key area key":    {RepoRoot: t.TempDir(), KeyAreaKey: []byte{0x04}},
		"oversize key area key": {RepoRoot: t.TempDir(), KeyAreaKey: bytes.Repeat([]byte{0x04}, 17)},
	} {
		t.Run(name, func(t *testing.T) {
			if err := opt.setDefaults(); err == nil {
				t.Fatal("setDefaults unexpectedly succeeded")
			}
		})
	}
}

func TestSetKeyGeneration(t *testing.T) {
	header := make([]byte, ncaHeaderSize)
	setKeyGeneration(header, 1)
	if header[0x206] != 0 || header[0x220] != 0 {
		t.Fatalf("keygeneration 1 should leave crypto types unset")
	}

	header = make([]byte, ncaHeaderSize)
	setKeyGeneration(header, 2)
	if header[0x206] != 0x02 || header[0x220] != 0 {
		t.Fatalf("keygeneration 2 set crypto_type=%02x crypto_type2=%02x", header[0x206], header[0x220])
	}

	header = make([]byte, ncaHeaderSize)
	setKeyGeneration(header, 5)
	if header[0x206] != 0x02 || header[0x220] != 0x05 {
		t.Fatalf("keygeneration 5 set crypto_type=%02x crypto_type2=%02x", header[0x206], header[0x220])
	}
}
