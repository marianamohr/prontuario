package crypto

import (
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	keysMap := map[string][]byte{
		"v1": make([]byte, 32),
	}
	plain := []byte("dado sensível")
	cipher, nonce, err := Encrypt(plain, "v1", keysMap)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(cipher) == 0 || len(nonce) == 0 {
		t.Fatal("cipher and nonce must be non-empty")
	}
	dec, err := Decrypt(cipher, nonce, "v1", keysMap)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(dec) != string(plain) {
		t.Fatalf("decrypted %q != plain %q", dec, plain)
	}
}

func TestParseKeysEnv(t *testing.T) {
	// 32 bytes in base64 = 43 chars (no padding)
	key := strings.Repeat("A", 43)
	env := "v1:" + key
	m, err := ParseKeysEnv(env)
	if err != nil {
		t.Fatalf("ParseKeysEnv: %v", err)
	}
	if len(m["v1"]) != 32 {
		t.Fatalf("key length: %d", len(m["v1"]))
	}
	// formato antigo 44 chars com "=" (ex-default no Railway) também deve funcionar
	envOld := "v1:" + key + "="
	mOld, err := ParseKeysEnv(envOld)
	if err != nil {
		t.Fatalf("ParseKeysEnv (44 chars): %v", err)
	}
	if len(mOld["v1"]) != 32 {
		t.Fatalf("key length (44 chars): %d", len(mOld["v1"]))
	}
	// múltiplas chaves (43 chars = 32 bytes decoded)
	env2 := "v1:" + key + ", v2:" + strings.Repeat("B", 43)
	m2, err := ParseKeysEnv(env2)
	if err != nil {
		t.Fatalf("ParseKeysEnv multi: %v", err)
	}
	if len(m2["v1"]) != 32 || len(m2["v2"]) != 32 {
		t.Fatalf("multi key lengths: v1=%d v2=%d", len(m2["v1"]), len(m2["v2"]))
	}
}
