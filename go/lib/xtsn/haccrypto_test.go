package xtsn

import (
	"encoding/hex"
	"testing"
)

type testCase struct {
	sectorOffset uint64
	sectorSize   uint64
	skippedBytes uint64
	expected     string
}

var testCases = []testCase{
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "917cf69ebd68b2ec9b9fe9a3eadda692cd43d2f59598ed858c02c2652fbf922e",
	},
	{
		sectorOffset: 1,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "1a8bbf023c9ea34d53b178e2f78a311f7885eda2505a642d84915d8d0eaa4943",
	},
	{
		sectorOffset: 2,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "8c7777fa01ad48beb8703abe8562a167faad9ae6a5b9d829774ca00ba81b1c6b",
	},
	{
		sectorOffset: 3,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "7e3227727e7525a944a3c40dd64cee473a686d319ccf94da000f1dc7c3e8774a",
	},
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 16,
		expected:     "cd43d2f59598ed858c02c2652fbf922e734867fd279b516a094b9713c18e7729",
	},
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 64,
		expected:     "4d8ebc0275bacb9e8680f1dfbce7f457e38eef2d13e6ac22bf8d24c7ea94e9ff",
	},
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 128,
		expected:     "8fb544486005c349bd303154f975016f51e78fd464536f4eb511a406173bae90",
	},
	{
		sectorOffset: 0,
		sectorSize:   512,
		skippedBytes: 0,
		expected:     "917cf69ebd68b2ec9b9fe9a3eadda692cd43d2f59598ed858c02c2652fbf922e",
	},
	{
		sectorOffset: 0,
		sectorSize:   512,
		skippedBytes: 512,
		expected:     "1a8bbf023c9ea34d53b178e2f78a311f7885eda2505a642d84915d8d0eaa4943",
	},
	{
		sectorOffset: 0,
		sectorSize:   512,
		skippedBytes: 256,
		expected:     "688f867c72098ed3f0b437fa240b6e77285421b75f810f9fd5d6183345644b3d",
	},
	{
		sectorOffset: 16,
		sectorSize:   512,
		skippedBytes: 256,
		expected:     "27a36c24e6933e7ac266d6004a818d56eb7a29e9dd2810fcc3f02aa3cca3cd61",
	},
}

func buf(size int) []byte {
	return make([]byte, size)
}

func TestHcCompatibleCipher(t *testing.T) {
	cipher, err := NewHcCompatibleCipher(buf(16), buf(16))
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	for i, tc := range testCases {
		data := buf(32)
		result, err := cipher.EncryptHC(data, tc.sectorOffset, tc.sectorSize, tc.skippedBytes)
		if err != nil {
			t.Fatalf("Case %d: Failed to run cipher: %v", i, err)
		}

		actual := hex.EncodeToString(result)
		if actual != tc.expected {
			t.Errorf("Case %d: Expected %s, got %s", i, tc.expected, actual)
		}
	}
}
