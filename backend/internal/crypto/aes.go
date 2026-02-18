package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func Encrypt(plaintext []byte, keyVersion string, keysMap map[string][]byte) (ciphertext, nonce []byte, err error) {
	key, ok := keysMap[keyVersion]
	if !ok {
		return nil, nil, errors.New("key version not found")
	}
	if len(key) != 32 {
		return nil, nil, errors.New("key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

func Decrypt(ciphertext, nonce []byte, keyVersion string, keysMap map[string][]byte) ([]byte, error) {
	key, ok := keysMap[keyVersion]
	if !ok {
		return nil, errors.New("key version not found")
	}
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func ParseKeysEnv(env string) (map[string][]byte, error) {
	out := make(map[string][]byte)
	if env == "" {
		return out, nil
	}
	for _, part := range strings.Split(env, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, ":")
		if idx <= 0 {
			continue
		}
		ver := strings.TrimSpace(part[:idx])
		b64 := strings.TrimSpace(part[idx+1:])
		// 44 chars com "=" no fim decodifica para 33 bytes e quebra; normaliza para 43 chars
		if len(b64) == 44 && strings.HasSuffix(b64, "=") {
			b64 = b64[:43]
		}
		var key []byte
		var err error
		if len(b64)%4 == 3 {
			// 43 chars em base64 = 32 bytes; sem padding usa RawStdEncoding
			key, err = base64.RawStdEncoding.DecodeString(b64)
		} else {
			switch len(b64) % 4 {
			case 2:
				b64 += "=="
			case 3:
				b64 += "="
			}
			key, err = base64.StdEncoding.DecodeString(b64)
		}
		if err != nil {
			return nil, err
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("key must be 32 bytes for AES-256 (got %d)", len(key))
		}
		out[ver] = key
	}
	return out, nil
}
