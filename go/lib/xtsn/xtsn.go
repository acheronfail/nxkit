package xtsn

// TODO: just discovered https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.38.0:xts/xts.go
// could potentially look at creating pools of tweaks for performance gains, etc

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"unsafe"
)

type Crypto interface {
	Encrypt(input []byte, byteOffset uint64) ([]byte, error)
	Decrypt(input []byte, byteOffset uint64) ([]byte, error)
}

type XtsnCipher struct {
	tweakCipher  cipher.Block
	cryptoCipher cipher.Block
	sectorSize   uint64
}

func NewXtsnCipher(tweakKey, cryptoKey []byte, sectorSize uint64) (*XtsnCipher, error) {
	if len(tweakKey) != 16 || len(cryptoKey) != 16 {
		return nil, errors.New("keys must be 16 bytes")
	}

	tweakCipher, err := aes.NewCipher(tweakKey)
	if err != nil {
		return nil, err
	}

	cryptoCipher, err := aes.NewCipher(cryptoKey)
	if err != nil {
		return nil, err
	}

	return &XtsnCipher{
		tweakCipher:  tweakCipher,
		cryptoCipher: cryptoCipher,
		sectorSize:   sectorSize,
	}, nil
}

func (x *XtsnCipher) Encrypt(input []byte, byteOffset uint64) ([]byte, error) {
	sectorOffset := byteOffset / x.sectorSize
	skippedBytes := byteOffset % x.sectorSize
	err := x.run(input, sectorOffset, skippedBytes, true)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (x *XtsnCipher) Decrypt(input []byte, byteOffset uint64) ([]byte, error) {
	sectorOffset := byteOffset / x.sectorSize
	skippedBytes := byteOffset % x.sectorSize
	err := x.run(input, sectorOffset, skippedBytes, false)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (x *XtsnCipher) run(input []byte, sectorOffset, skippedBytes uint64, encrypt bool) error {

	var update func([]byte, []byte)
	if encrypt {
		update = x.cryptoCipher.Encrypt
	} else {
		update = x.cryptoCipher.Decrypt
	}

	chunkOffset := uint64(0)
	totalChunks := uint64(len(input) / 8)

	if skippedBytes > 0 {
		fullSectorsToSkip := skippedBytes / x.sectorSize
		sectorOffset += fullSectorsToSkip
		skippedBytes %= x.sectorSize
	}

	if skippedBytes > 0 {
		tweak := make([]byte, 16)
		x.initTweak(tweak, sectorOffset)

		for range int(skippedBytes) / 16 {
			x.updateTweak(tweak)
		}

		err := x.processChunks(input, tweak, &chunkOffset, totalChunks, (x.sectorSize-skippedBytes)/16, update)
		if err != nil {
			return err
		}
		sectorOffset++
	}

	for chunkOffset < totalChunks {
		tweak := make([]byte, 16)
		x.initTweak(tweak, sectorOffset)
		err := x.processChunks(input, tweak, &chunkOffset, totalChunks, x.sectorSize/16, update)
		if err != nil {
			return err
		}
		sectorOffset++
	}

	return nil
}

func (x *XtsnCipher) initTweak(tweak []byte, sectorOffset uint64) {
	binary.BigEndian.PutUint64(tweak[8:], sectorOffset)
	x.tweakCipher.Encrypt(tweak, tweak)
}

func (x *XtsnCipher) updateTweak(tweak []byte) {
	lastHigh := (tweak[15] & 0x80) != 0
	tweak64bit := (*[2]uint64)(unsafe.Pointer(&tweak[0]))
	tweak64bit[1] = (tweak64bit[1] << 1) | (tweak64bit[0] >> 63)
	tweak64bit[0] = (tweak64bit[0] << 1)
	if lastHigh {
		tweak64bit[0] ^= 0x87
	}
}

// FIXME: remove error since it's not used
func (x *XtsnCipher) processChunks(input []byte, tweak []byte, chunkOffset *uint64, totalChunks, runs uint64, update func([]byte, []byte)) error {
	tweak64bit := (*[2]uint64)(unsafe.Pointer(&tweak[0]))
	// Reinterpret byte slice as uint64 slice with [1<<30] for capacity
	// then limit length with [:len(input)/8] to match input bytes
	inputUint64 := (*[1 << 30]uint64)(unsafe.Pointer(&input[0]))[:len(input)/8]

	for i := uint64(0); i < runs; i++ {
		if *chunkOffset >= totalChunks {
			return nil
		}

		inputUint64[*chunkOffset] ^= tweak64bit[0]
		inputUint64[*chunkOffset+1] ^= tweak64bit[1]

		byteOffset := *chunkOffset * 8
		block := input[byteOffset : byteOffset+16]
		update(block, block)

		inputUint64[*chunkOffset] ^= tweak64bit[0]
		inputUint64[*chunkOffset+1] ^= tweak64bit[1]

		x.updateTweak(tweak)

		*chunkOffset += 2
	}

	return nil
}
