package api

import (
	"errors"
	"strings"
)

var (
	ErrInvalidEmail        = errors.New("invalid email")
	ErrInvalidCPF          = errors.New("invalid cpf")
	ErrInvalidGuardianAddr = errors.New("invalid guardian address")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidCEP          = errors.New("invalid cep")
)

// AddressInput representa os 8 campos de endereço para validação e parse.
// Ordem para string de 8 linhas: rua, numero, complemento, bairro, cidade, estado, pais, cep.
type AddressInput struct {
	Street       string
	Number       string // opcional
	Complement   string // opcional
	Neighborhood string
	City         string
	State        string
	Country      string
	Zip          string
}

// ValidateAddress valida os campos do endereço: street, neighborhood, city, state, country, zip obrigatórios (não vazios);
// number e complement opcionais; CEP deve ter 8 dígitos.
func ValidateAddress(a *AddressInput) error {
	if a == nil {
		return ErrInvalidAddress
	}
	street := strings.TrimSpace(a.Street)
	neighborhood := strings.TrimSpace(a.Neighborhood)
	city := strings.TrimSpace(a.City)
	state := strings.TrimSpace(a.State)
	country := strings.TrimSpace(a.Country)
	zip := onlyDigits(strings.TrimSpace(a.Zip))
	if street == "" || neighborhood == "" || city == "" || state == "" || country == "" {
		return ErrInvalidAddress
	}
	if len(zip) != 8 {
		return ErrInvalidCEP
	}
	return nil
}

// ParseAddressFrom8Lines interpreta uma string com 8 linhas na ordem: rua, numero, complemento, bairro, cidade, estado, pais, cep.
// Retorna nil se a string estiver vazia ou não tiver 8 linhas.
func ParseAddressFrom8Lines(s string) (*AddressInput, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrInvalidGuardianAddr
	}
	parts := strings.Split(s, "\n")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) < 8 {
		return nil, ErrInvalidGuardianAddr
	}
	return &AddressInput{
		Street:       parts[0],
		Number:       parts[1],
		Complement:   parts[2],
		Neighborhood: parts[3],
		City:         parts[4],
		State:        parts[5],
		Country:      parts[6],
		Zip:          parts[7],
	}, nil
}

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
