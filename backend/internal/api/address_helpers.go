package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/repo"
)

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func strFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// AddressInputToRepo converte AddressInput (api) para repo.Address (ponteiros para o DB).
func AddressInputToRepo(in *AddressInput) *repo.Address {
	if in == nil {
		return nil
	}
	return &repo.Address{
		Street:       strPtr(in.Street),
		Number:       strPtr(in.Number),
		Complement:   strPtr(in.Complement),
		Neighborhood: strPtr(in.Neighborhood),
		City:         strPtr(in.City),
		State:        strPtr(in.State),
		Country:      strPtr(in.Country),
		Zip:          strPtr(in.Zip),
	}
}

// parseAddressFromRequest interpreta guardian_address/address do request: string (8 linhas) ou objeto JSON (8 campos).
func parseAddressFromRequest(v interface{}) (*AddressInput, error) {
	if v == nil {
		return nil, ErrInvalidGuardianAddr
	}
	switch val := v.(type) {
	case string:
		return ParseAddressFrom8Lines(val)
	default:
		// Objeto: re-serializar para JSON e decodificar em AddressInput
		b, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		var obj struct {
			Street       string `json:"street"`
			Number       string `json:"number"`
			Complement   string `json:"complement"`
			Neighborhood string `json:"neighborhood"`
			City         string `json:"city"`
			State        string `json:"state"`
			Country      string `json:"country"`
			Zip          string `json:"zip"`
		}
		if err := json.Unmarshal(b, &obj); err != nil {
			return nil, err
		}
		return &AddressInput{
			Street:       obj.Street,
			Number:       obj.Number,
			Complement:   obj.Complement,
			Neighborhood: obj.Neighborhood,
			City:         obj.City,
			State:        obj.State,
			Country:      obj.Country,
			Zip:          obj.Zip,
		}, nil
	}
}

// ParseAddressFromRequest é a versão exportada para uso em outros handlers (retorna erro com mensagem).
func ParseAddressFromRequest(v interface{}) (*AddressInput, error) {
	a, err := parseAddressFromRequest(v)
	if err != nil {
		return nil, fmt.Errorf("address: %w", err)
	}
	return a, nil
}

// AddressToMap converte repo.Address para mapa JSON (8 campos) para resposta da API.
func AddressToMap(a *repo.Address) map[string]interface{} {
	if a == nil {
		return nil
	}
	m := make(map[string]interface{})
	if a.Street != nil {
		m["street"] = *a.Street
	}
	if a.Number != nil {
		m["number"] = *a.Number
	}
	if a.Complement != nil {
		m["complement"] = *a.Complement
	}
	if a.Neighborhood != nil {
		m["neighborhood"] = *a.Neighborhood
	}
	if a.City != nil {
		m["city"] = *a.City
	}
	if a.State != nil {
		m["state"] = *a.State
	}
	if a.Country != nil {
		m["country"] = *a.Country
	}
	if a.Zip != nil {
		m["zip"] = *a.Zip
	}
	return m
}

// addressToMap é alias para AddressToMap (uso interno nos handlers).
func addressToMap(a *repo.Address) map[string]interface{} {
	return AddressToMap(a)
}

// FormatAddressToLines formata repo.Address em linhas (rua, numero, complemento, bairro, cidade, estado, pais, cep) para uso em contratos/PDF.
func FormatAddressToLines(a *repo.Address) string {
	if a == nil {
		return ""
	}
	var parts []string
	for _, p := range []*string{a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip} {
		if p != nil && strings.TrimSpace(*p) != "" {
			parts = append(parts, strings.TrimSpace(*p))
		} else {
			parts = append(parts, "")
		}
	}
	return strings.Join(parts, "\n")
}

// FormatGuardianAddressForContract retorna o endereço do responsável formatado em linhas para preencher [RESPONSAVEL_ENDERECO] no contrato.
func FormatGuardianAddressForContract(ctx context.Context, pool *pgxpool.Pool, guardian *repo.LegalGuardian) string {
	if guardian == nil || guardian.AddressID == nil {
		return ""
	}
	addr, err := repo.GetAddressByID(ctx, pool, *guardian.AddressID)
	if err != nil {
		return ""
	}
	return FormatAddressToLines(addr)
}
