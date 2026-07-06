package hacbrewpack

import (
	"bytes"
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"
)

func TestSignPSSDeterministicToggle(t *testing.T) {
	priv, err := rsa.GenerateKey(cryptorand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	data := []byte("header bytes to sign")

	deterministicA, err := signPSS(priv, data, false)
	if err != nil {
		t.Fatalf("deterministic sign failed: %v", err)
	}
	deterministicB, err := signPSS(priv, data, false)
	if err != nil {
		t.Fatalf("deterministic sign failed: %v", err)
	}
	if !bytes.Equal(deterministicA, deterministicB) {
		t.Fatal("deterministic PSS signatures differed")
	}

	randomA, err := signPSS(priv, data, true)
	if err != nil {
		t.Fatalf("random sign failed: %v", err)
	}
	randomB, err := signPSS(priv, data, true)
	if err != nil {
		t.Fatalf("random sign failed: %v", err)
	}
	if bytes.Equal(randomA, randomB) {
		t.Fatal("random PSS signatures unexpectedly matched")
	}

	hash := sha256.Sum256(data)
	for name, sig := range map[string][]byte{
		"deterministic": deterministicA,
		"randomA":       randomA,
		"randomB":       randomB,
	} {
		if err := rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, hash[:], sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256}); err != nil {
			t.Fatalf("%s signature did not verify: %v", name, err)
		}
	}
}
