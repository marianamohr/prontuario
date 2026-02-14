package auth

import (
	"testing"
)

func TestHashPasswordAndCheck(t *testing.T) {
	plain := "SenhaSecreta123!"
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == plain {
		t.Fatal("hash must not equal plaintext")
	}
	if !CheckPassword(hash, plain) {
		t.Fatal("CheckPassword should succeed for correct password")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("CheckPassword should fail for wrong password")
	}
}
