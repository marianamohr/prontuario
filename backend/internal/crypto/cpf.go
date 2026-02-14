package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

var onlyDigits = regexp.MustCompile(`[^0-9]`)

// NormalizeCPF remove tudo que não for dígito (11 dígitos).
func NormalizeCPF(cpf string) string {
	return onlyDigits.ReplaceAllString(cpf, "")
}

// CPFHash retorna SHA-256 do CPF normalizado em hex.
func CPFHash(cpfNormalized string) string {
	h := sha256.Sum256([]byte(cpfNormalized))
	return hex.EncodeToString(h[:])
}
