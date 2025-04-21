package xtsn

import "errors"

type HcCompatibleCipher struct {
	tweakKey  []byte
	cryptoKey []byte
}

func NewHcCompatibleCipher(tweakKey, cryptoKey []byte) (*HcCompatibleCipher, error) {
	if len(tweakKey) != 16 || len(cryptoKey) != 16 {
		return nil, errors.New("keys must be 16 bytes")
	}

	return &HcCompatibleCipher{
		tweakKey:  tweakKey,
		cryptoKey: cryptoKey,
	}, nil
}

func (h *HcCompatibleCipher) EncryptHC(input []byte, sectorOffset, sectorSize, skippedBytes uint64) ([]byte, error) {
	cipher, err := NewXtsnCipher(h.cryptoKey, h.tweakKey, sectorSize)
	if err != nil {
		return nil, err
	}

	err = cipher.run(input, sectorOffset, skippedBytes, true)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (h *HcCompatibleCipher) DecryptHC(input []byte, sectorOffset, sectorSize, skippedBytes uint64) ([]byte, error) {
	cipher, err := NewXtsnCipher(h.cryptoKey, h.tweakKey, sectorSize)
	if err != nil {
		return nil, err
	}

	err = cipher.run(input, sectorOffset, skippedBytes, false)
	if err != nil {
		return nil, err
	}

	return input, nil
}
