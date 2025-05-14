package xtsn

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func noErr(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func TestXtsnCipher(t *testing.T) {
	for i, tc := range testCases {
		xtsn, err := NewXtsnCipher(buf(16), buf(16), tc.sectorSize)
		noErr(t, err, "Failed to create cipher")

		data := buf(32)
		xtsn.Encrypt(data, tc.sectorOffset*tc.sectorSize+tc.skippedBytes)

		actual := hex.EncodeToString(data)
		if actual != tc.expected {
			t.Errorf("Case %d: Expected %s, got %s", i, tc.expected, actual)
		}

		xtsn.Decrypt(data, tc.sectorOffset*tc.sectorSize+tc.skippedBytes)
		noErr(t, err, "Failed to run cipher")

		if !bytes.Equal(data, buf(32)) {
			t.Errorf("Case %d: Expected clear buffer, got %v", i, data)
		}
	}
}

func TestXtsnCases(t *testing.T) {
	xtsn, err := NewXtsnCipher(buf(16), buf(16), 0x4000)
	noErr(t, err, "Failed to create cipher")

	t.Run("read from start to end", func(t *testing.T) {
		zeroes := buf(0x4000 * 4)
		xtsn.Encrypt(zeroes, 0)
		noErr(t, err, "Failed to encrypt")
		xtsn.Decrypt(zeroes, 0)
		noErr(t, err, "Failed to decrypt")
		if !bytes.Equal(zeroes, buf(0x4000*4)) {
			t.Error("Expected all zeros after decryption")
		}
	})

	t.Run("read first half", func(t *testing.T) {
		zeroes := buf(0x4000 * 4)
		xtsn.Encrypt(zeroes, 0)
		noErr(t, err, "Failed to encrypt")
		xtsn.Decrypt(zeroes[:0x4000*2], 0)
		noErr(t, err, "Failed to decrypt")
		if !bytes.Equal(zeroes[:0x4000*2], buf(0x4000*2)) {
			t.Error("Expected all zeros after decryption")
		}
	})

	t.Run("read last half", func(t *testing.T) {
		zeroes := buf(0x4000 * 4)
		xtsn.Encrypt(zeroes, 0)
		noErr(t, err, "Failed to encrypt")
		xtsn.Decrypt(zeroes[0x4000*2:], 0x4000*2)
		noErr(t, err, "Failed to decrypt")
		if !bytes.Equal(zeroes[0x4000*2:], buf(0x4000*2)) {
			t.Error("Expected all zeros after decryption")
		}
	})

	t.Run("read middle half", func(t *testing.T) {
		zeroes := buf(0x4000 * 4)
		xtsn.Encrypt(zeroes, 0)
		noErr(t, err, "Failed to encrypt")
		xtsn.Decrypt(zeroes[0x4000:0x4000*3], 0x4000)
		noErr(t, err, "Failed to decrypt")
		if !bytes.Equal(zeroes[0x4000:0x4000*3], buf(0x4000*2)) {
			t.Error("Expected all zeros after decryption")
		}
	})
}
