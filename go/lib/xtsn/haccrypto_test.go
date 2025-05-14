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
		expected:     "06ffb48a716e78b3c5414d1476aa600aabe471682b86c522f66ad898fef1fcfd",
	},
	{
		sectorOffset: 1,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "61a5b6b142d725c396dce2f467f157ddbe487056842c286945953a62f03eff91",
	},
	{
		sectorOffset: 2,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "e8a5243e513f312d5e8ae4e563afe3d5a8d049436c837c0628e8a30488396744",
	},
	{
		sectorOffset: 3,
		sectorSize:   16384,
		skippedBytes: 0,
		expected:     "c6a82e79bf15f7601ec9b032db42e791e5e7c3c41f5c770c214d7c296b5bcf2f",
	},
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 16,
		expected:     "abe471682b86c522f66ad898fef1fcfdfe7952e2abfbea8c5830bfabed22aa6b",
	},
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 64,
		expected:     "3084c04bf5e7b4de2e9fce845225c3ddfb56bda7c775003efeb8ddbfe5babd23",
	},
	{
		sectorOffset: 0,
		sectorSize:   16384,
		skippedBytes: 128,
		expected:     "a75e804d817917e00987928bad99587886a753bd35149e3068b9d7404d7d4ab6",
	},
	{
		sectorOffset: 0,
		sectorSize:   512,
		skippedBytes: 0,
		expected:     "06ffb48a716e78b3c5414d1476aa600aabe471682b86c522f66ad898fef1fcfd",
	},
	{
		sectorOffset: 0,
		sectorSize:   512,
		skippedBytes: 512,
		expected:     "61a5b6b142d725c396dce2f467f157ddbe487056842c286945953a62f03eff91",
	},
	{
		sectorOffset: 0,
		sectorSize:   512,
		skippedBytes: 256,
		expected:     "6eefef4835a0d4789ec5336475564e3a0d20547d9b053f59201f7101d05e39e5",
	},
	{
		sectorOffset: 16,
		sectorSize:   512,
		skippedBytes: 256,
		expected:     "d8c7ecaf0de20be3e7f616096be5b57262a6a164a3a4ab472e590d44eeb87ca4",
	},
}

func buf(size int, fill byte) []byte {
	b := make([]byte, size)
	if fill != 0 {
		for i := range size {
			b[i] = fill
		}
	}

	return b
}

func TestHcCompatibleCipher(t *testing.T) {
	cipher, err := NewHcCompatibleCipher(buf(16, 0), buf(16, 1))
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	for i, tc := range testCases {
		data := buf(32, 0)
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
