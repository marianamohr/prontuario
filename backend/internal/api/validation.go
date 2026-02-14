package api

import (
	"errors"
	"strings"
)

var (
	ErrInvalidEmail         = errors.New("invalid email")
	ErrInvalidCPF           = errors.New("invalid cpf")
	ErrInvalidGuardianAddr  = errors.New("invalid guardian address")
	ErrInvalidCEP           = errors.New("invalid cep")
)

// ValidateEmailRegex valida formato de e-mail com o regex padrão do backend.
func ValidateEmailRegex(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ErrInvalidEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateGuardianAddress valida que o endereço tem 6 linhas e CEP com 8 dígitos (última linha).
// Formato: Rua, Bairro, Cidade, Estado, País, CEP
func ValidateGuardianAddress(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ErrInvalidGuardianAddr
	}
	parts := strings.Split(addr, "\n")
	if len(parts) < 6 {
		return ErrInvalidGuardianAddr
	}
	cep := onlyDigits(strings.TrimSpace(parts[5]))
	if len(cep) != 8 {
		return ErrInvalidCEP
	}
	return nil
}

func onlyDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

