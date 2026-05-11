package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptGCM(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	plain := []byte("wireguard-private-key-material")

	s, err := EncryptGCM(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecryptGCM(key, s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatal("plaintext mismatch")
	}
}
