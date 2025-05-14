package xtsn

import "testing"

func BenchmarkXtsnCipher(b *testing.B) {
	xtsn, err := NewXtsnCipher(buf(16, 0), buf(16, 0), 0x2000)
	if err != nil {
		b.Fatalf("Failed to create XtsnCipher: %v", err)
	}

	input := buf(0x2000, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xtsn.Encrypt(input, 0)
	}
	b.StopTimer()
}
