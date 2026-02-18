package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
)

// emailRegex valida formato de e-mail (uma @ e domínio com ponto).
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// formatDateBR converte YYYY-MM-DD em DD/MM/YYYY; retorna "" se inválido.
func formatDateBR(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return t.Format("02/01/2006")
}

func (h *Handler) ListPatients(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	list, err := repo.PatientsByClinic(r.Context(), h.Pool, cid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type patientResp struct {
		ID        string  `json:"id"`
		FullName  string  `json:"full_name"`
		BirthDate *string `json:"birth_date,omitempty"`
	}
	out := make([]patientResp, len(list))
	for i := range list {
		out[i] = patientResp{ID: list[i].ID.String(), FullName: list[i].FullName, BirthDate: list[i].BirthDate}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"patients": out})
}

func (h *Handler) GetPatient(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	p, err := repo.PatientByIDAndClinic(r.Context(), h.Pool, patientID, cid)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	out := map[string]interface{}{
		"id":         p.ID.String(),
		"full_name":  p.FullName,
		"birth_date": p.BirthDate,
		"email":      p.Email,
	}
	// CPF do paciente (opcional)
	var patientCPFStr *string
	if p.CPFKeyVersion != nil && *p.CPFKeyVersion != "" && len(p.CPFEncrypted) > 0 && len(p.CPFNonce) > 0 {
		keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
		if err == nil {
			dec, err := crypto.Decrypt(p.CPFEncrypted, p.CPFNonce, *p.CPFKeyVersion, keysMap)
			if err == nil && len(dec) > 0 {
				s := string(dec)
				patientCPFStr = &s
			}
		}
	}
	out["cpf"] = patientCPFStr
	guardians, errGuardians := repo.GuardiansByPatient(r.Context(), h.Pool, patientID)
	_ = errGuardians
	if len(guardians) > 0 {
		g, errG := repo.LegalGuardianByID(r.Context(), h.Pool, guardians[0].ID)
		_ = errG
		if g != nil {
			// Descriptografa CPF do responsável (para PROFESSIONAL do tenant / SUPER_ADMIN).
			var cpfStr *string
			if g.CPFKeyVersion != nil && *g.CPFKeyVersion != "" && len(g.CPFEncrypted) > 0 && len(g.CPFNonce) > 0 {
				keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
				if err == nil {
					dec, err := crypto.Decrypt(g.CPFEncrypted, g.CPFNonce, *g.CPFKeyVersion, keysMap)
					if err == nil && len(dec) > 0 {
						s := string(dec)
						cpfStr = &s
					}
				}
			}
			var guardianAddr map[string]interface{}
			if g.AddressID != nil {
				if addr, err := repo.GetAddressByID(r.Context(), h.Pool, *g.AddressID); err == nil {
					guardianAddr = AddressToMap(addr)
				}
			}
			out["guardian"] = map[string]interface{}{
				"id":         g.ID.String(),
				"full_name":  g.FullName,
				"email":      g.Email,
				"cpf":        cpfStr,
				"address":    guardianAddr,
				"birth_date": g.BirthDate,
				"phone":      g.Phone,
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

type UpdatePatientRequest struct {
	FullName          *string     `json:"full_name"`
	BirthDate         *string     `json:"birth_date"`
	Email             *string     `json:"email"`
	PatientCPF        *string     `json:"patient_cpf"`
	PatientAddress   interface{} `json:"patient_address,omitempty"` // opcional: objeto ou 8 linhas
	GuardianFullName  *string     `json:"guardian_full_name"`
	GuardianEmail     *string     `json:"guardian_email"`
	GuardianAddress   interface{} `json:"guardian_address"` // objeto ou 8 linhas
	GuardianBirthDate *string     `json:"guardian_birth_date"`
	GuardianPhone     *string     `json:"guardian_phone"`
	GuardianCPF       *string     `json:"guardian_cpf"`
}

func (h *Handler) UpdatePatient(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	p, err := repo.PatientByIDAndClinic(r.Context(), h.Pool, patientID, cid)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	var req UpdatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	fullName := p.FullName
	if req.FullName != nil {
		fullName = strings.TrimSpace(*req.FullName)
		if fullName == "" {
			http.Error(w, `{"error":"full_name cannot be empty"}`, http.StatusBadRequest)
			return
		}
	}
	var birthDate *string
	if req.BirthDate != nil {
		s := strings.TrimSpace(*req.BirthDate)
		birthDate = &s
	} else {
		birthDate = p.BirthDate
	}
	var email *string
	if req.Email != nil {
		s := strings.TrimSpace(*req.Email)
		email = &s
	} else {
		email = p.Email
	}
	var patientAddrID *uuid.UUID
	if req.PatientAddress != nil {
		addrInput, err := parseAddressFromRequest(req.PatientAddress)
		if err != nil {
			http.Error(w, `{"error":"patient_address inválido: use objeto com 8 campos ou string de 8 linhas"}`, http.StatusBadRequest)
			return
		}
		if err := ValidateAddress(addrInput); err != nil {
			http.Error(w, `{"error":"patient_address inválido (CEP 8 dígitos; rua, bairro, cidade, estado, país obrigatórios)"}`, http.StatusBadRequest)
			return
		}
		addr := AddressInputToRepo(addrInput)
		id, err := repo.CreateAddress(r.Context(), h.Pool, addr)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		patientAddrID = &id
	} else {
		patientAddrID = p.AddressID
	}
	if err := repo.UpdatePatient(r.Context(), h.Pool, p.ID, cid, fullName, birthDate, email, patientAddrID); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// CPF do paciente (opcional): atualiza apenas quando enviado no payload.
	if req.PatientCPF != nil {
		s := strings.TrimSpace(*req.PatientCPF)
		if s == "" {
			if err := repo.ClearPatientCPF(r.Context(), h.Pool, p.ID, cid); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
		} else {
			n := crypto.NormalizeCPF(s)
			if len(n) != 11 {
				http.Error(w, `{"error":"cpf do paciente inválido"}`, http.StatusBadRequest)
				return
			}
			cpfHash := crypto.CPFHash(n)
			keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
			if err != nil {
				http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
				return
			}
			keyVer := h.Cfg.CurrentDataKeyVer
			if keyVer == "" {
				keyVer = "v1"
			}
			enc, nonce, err := crypto.Encrypt([]byte(n), keyVer, keysMap)
			if err != nil {
				http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
				return
			}
			if err := repo.SetPatientCPF(r.Context(), h.Pool, p.ID, cid, enc, nonce, keyVer, cpfHash); err != nil {
				if isUniqueViolation(err) {
					http.Error(w, `{"error":"cpf já cadastrado para outro paciente"}`, http.StatusBadRequest)
					return
				}
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
		}
	}
	guardians, errGuardians := repo.GuardiansByPatient(r.Context(), h.Pool, patientID)
	_ = errGuardians
	if len(guardians) > 0 && (req.GuardianFullName != nil || req.GuardianEmail != nil || req.GuardianAddress != nil || req.GuardianBirthDate != nil || req.GuardianPhone != nil || req.GuardianCPF != nil) {
		g, err := repo.LegalGuardianByID(r.Context(), h.Pool, guardians[0].ID)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		gFullName := g.FullName
		if req.GuardianFullName != nil {
			gFullName = strings.TrimSpace(*req.GuardianFullName)
			if gFullName == "" {
				http.Error(w, `{"error":"guardian_full_name cannot be empty"}`, http.StatusBadRequest)
				return
			}
		}
		gEmail := g.Email
		if req.GuardianEmail != nil {
			gEmail = strings.TrimSpace(*req.GuardianEmail)
			if gEmail != "" && !emailRegex.MatchString(gEmail) {
				http.Error(w, `{"error":"guardian_email inválido"}`, http.StatusBadRequest)
				return
			}
		}
		var gAddrID *uuid.UUID
		if req.GuardianAddress != nil {
			addrInput, err := parseAddressFromRequest(req.GuardianAddress)
			if err != nil {
				http.Error(w, `{"error":"guardian_address inválido: use objeto com 8 campos ou string de 8 linhas"}`, http.StatusBadRequest)
				return
			}
			if err := ValidateAddress(addrInput); err != nil {
				http.Error(w, `{"error":"guardian_address inválido (CEP 8 dígitos; rua, bairro, cidade, estado, país obrigatórios)"}`, http.StatusBadRequest)
				return
			}
			addr := AddressInputToRepo(addrInput)
			id, err := repo.CreateAddress(r.Context(), h.Pool, addr)
			if err != nil {
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
			gAddrID = &id
		} else {
			gAddrID = g.AddressID
		}
		var gBirth *string
		if req.GuardianBirthDate != nil {
			s := strings.TrimSpace(*req.GuardianBirthDate)
			gBirth = &s
		} else {
			gBirth = g.BirthDate
		}
		var gPhone *string
		if req.GuardianPhone != nil {
			s := strings.TrimSpace(*req.GuardianPhone)
			if s != "" {
				gPhone = &s
			} else {
				gPhone = nil
			}
		} else {
			gPhone = g.Phone
		}
		// Atualiza dados não sensíveis
		if err := repo.UpdateLegalGuardian(r.Context(), h.Pool, g.ID, gFullName, gEmail, gAddrID, gBirth, gPhone, nil); err != nil {
			if isUniqueViolation(err) {
				http.Error(w, `{"error":"email já utilizado"}`, http.StatusBadRequest)
				return
			}
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		// CPF do responsável (se enviado): criptografa + atualiza hash
		if req.GuardianCPF != nil && strings.TrimSpace(*req.GuardianCPF) != "" {
			n := crypto.NormalizeCPF(*req.GuardianCPF)
			if len(n) != 11 {
				http.Error(w, `{"error":"cpf inválido"}`, http.StatusBadRequest)
				return
			}
			cpfHash := crypto.CPFHash(n)
			keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
			if err != nil {
				http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
				return
			}
			keyVer := h.Cfg.CurrentDataKeyVer
			if keyVer == "" {
				keyVer = "v1"
			}
			enc, nonce, err := crypto.Encrypt([]byte(n), keyVer, keysMap)
			if err != nil {
				http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
				return
			}
			if err := repo.UpdateLegalGuardianCPF(r.Context(), h.Pool, g.ID, enc, nonce, keyVer, cpfHash); err != nil {
				if isUniqueViolation(err) {
					http.Error(w, `{"error":"cpf já utilizado"}`, http.StatusBadRequest)
					return
				}
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Paciente atualizado."})
}

type CreatePatientRequest struct {
	FullName   string  `json:"full_name"`
	BirthDate  *string `json:"birth_date,omitempty"`
	PatientCPF *string `json:"patient_cpf,omitempty"`
	PatientAddress interface{} `json:"patient_address,omitempty"` // opcional: objeto ou 8 linhas
	// Com guardião legal (quando preenchido cria responsável + vínculo)
	SamePerson        bool        `json:"same_person"`
	GuardianFullName  string      `json:"guardian_full_name"`
	GuardianEmail     string      `json:"guardian_email"`
	GuardianCPF       string      `json:"guardian_cpf"`
	GuardianAddress   interface{} `json:"guardian_address"` // objeto (8 campos) ou string (8 linhas)
	GuardianBirthDate string      `json:"guardian_birth_date"`
	GuardianPhone     string      `json:"guardian_phone"`
	PatientFullName   string      `json:"patient_full_name"`
}

func (h *Handler) CreatePatient(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	var req CreatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.GuardianEmail != "" {
		if !emailRegex.MatchString(strings.TrimSpace(req.GuardianEmail)) {
			http.Error(w, `{"error":"guardian_email inválido"}`, http.StatusBadRequest)
			return
		}
		if req.GuardianFullName == "" {
			http.Error(w, `{"error":"guardian_full_name required when guardian_email is set"}`, http.StatusBadRequest)
			return
		}
		if req.GuardianCPF == "" {
			http.Error(w, `{"error":"guardian_cpf required"}`, http.StatusBadRequest)
			return
		}
		guardianAddrInput, err := parseAddressFromRequest(req.GuardianAddress)
		if err != nil {
			http.Error(w, `{"error":"guardian_address required: use objeto com 8 campos ou string de 8 linhas (rua, numero, complemento, bairro, cidade, estado, pais, cep)"}`, http.StatusBadRequest)
			return
		}
		if err := ValidateAddress(guardianAddrInput); err != nil {
			http.Error(w, `{"error":"guardian_address inválido (CEP 8 dígitos; rua, bairro, cidade, estado, país obrigatórios)"}`, http.StatusBadRequest)
			return
		}
		if req.GuardianBirthDate == "" {
			http.Error(w, `{"error":"guardian_birth_date required"}`, http.StatusBadRequest)
			return
		}
		if req.BirthDate == nil || *req.BirthDate == "" {
			http.Error(w, `{"error":"patient birth_date required"}`, http.StatusBadRequest)
			return
		}
		patientName := req.PatientFullName
		if req.SamePerson || patientName == "" {
			patientName = req.GuardianFullName
		}
		n := crypto.NormalizeCPF(req.GuardianCPF)
		if len(n) != 11 {
			http.Error(w, `{"error":"cpf inválido"}`, http.StatusBadRequest)
			return
		}
		cpfHash := crypto.CPFHash(n)
		keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
		if err != nil {
			http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
			return
		}
		keyVer := h.Cfg.CurrentDataKeyVer
		if keyVer == "" {
			keyVer = "v1"
		}
		cpfEnc, nonce, err := crypto.Encrypt([]byte(n), keyVer, keysMap)
		if err != nil {
			http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
			return
		}
		guardianAddr := AddressInputToRepo(guardianAddrInput)
		guardianAddressID, err := repo.CreateAddress(r.Context(), h.Pool, guardianAddr)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		guardianBirth := req.GuardianBirthDate
		var gPhone *string
		if s := strings.TrimSpace(req.GuardianPhone); s != "" {
			gPhone = &s
		}
		g := &repo.LegalGuardian{
			Email:         req.GuardianEmail,
			FullName:      req.GuardianFullName,
			PasswordHash:  nil,
			CPFEncrypted:  cpfEnc,
			CPFNonce:      nonce,
			CPFKeyVersion: &keyVer,
			CPFHash:       &cpfHash,
			AddressID:     &guardianAddressID,
			BirthDate:     &guardianBirth,
			Phone:         gPhone,
			AuthProvider:  "LOCAL",
			Status:        "ACTIVE",
		}
		if err := repo.CreateLegalGuardian(r.Context(), h.Pool, g); err != nil {
			http.Error(w, `{"error":"email já utilizado ou internal"}`, http.StatusBadRequest)
			return
		}
		var patientAddrID *uuid.UUID
		if req.PatientAddress != nil {
			addrInput, err := parseAddressFromRequest(req.PatientAddress)
			if err == nil && ValidateAddress(addrInput) == nil {
				addr := AddressInputToRepo(addrInput)
				if id, err := repo.CreateAddress(r.Context(), h.Pool, addr); err == nil {
					patientAddrID = &id
				}
			}
		}
		patientID, err := repo.CreatePatient(r.Context(), h.Pool, cid, patientName, req.BirthDate, nil, patientAddrID)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		// CPF do paciente (opcional)
		if req.PatientCPF != nil {
			s := strings.TrimSpace(*req.PatientCPF)
			if s != "" {
				n := crypto.NormalizeCPF(s)
				if len(n) != 11 {
					http.Error(w, `{"error":"cpf do paciente inválido"}`, http.StatusBadRequest)
					return
				}
				cpfHash := crypto.CPFHash(n)
				keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
				if err != nil {
					http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
					return
				}
				keyVer := h.Cfg.CurrentDataKeyVer
				if keyVer == "" {
					keyVer = "v1"
				}
				enc, nonce, err := crypto.Encrypt([]byte(n), keyVer, keysMap)
				if err != nil {
					http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
					return
				}
				if err := repo.SetPatientCPF(r.Context(), h.Pool, patientID, cid, enc, nonce, keyVer, cpfHash); err != nil {
					if isUniqueViolation(err) {
						http.Error(w, `{"error":"cpf já cadastrado para outro paciente"}`, http.StatusBadRequest)
						return
					}
					http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
					return
				}
			}
		}
		relation := "Titular"
		if !req.SamePerson && req.PatientFullName != "" {
			relation = "Responsável"
		}
		if err := repo.CreatePatientGuardian(r.Context(), h.Pool, patientID, g.ID, relation, true, true); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": patientID.String()})
		return
	}
	if req.FullName == "" {
		http.Error(w, `{"error":"full_name required"}`, http.StatusBadRequest)
		return
	}
	id, err := repo.CreatePatient(r.Context(), h.Pool, cid, req.FullName, req.BirthDate, nil, nil)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// CPF do paciente (opcional)
	if req.PatientCPF != nil {
		s := strings.TrimSpace(*req.PatientCPF)
		if s != "" {
			n := crypto.NormalizeCPF(s)
			if len(n) != 11 {
				http.Error(w, `{"error":"cpf do paciente inválido"}`, http.StatusBadRequest)
				return
			}
			cpfHash := crypto.CPFHash(n)
			keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
			if err != nil {
				http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
				return
			}
			keyVer := h.Cfg.CurrentDataKeyVer
			if keyVer == "" {
				keyVer = "v1"
			}
			enc, nonce, err := crypto.Encrypt([]byte(n), keyVer, keysMap)
			if err != nil {
				http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
				return
			}
			if err := repo.SetPatientCPF(r.Context(), h.Pool, id, cid, enc, nonce, keyVer, cpfHash); err != nil {
				if isUniqueViolation(err) {
					http.Error(w, `{"error":"cpf já cadastrado para outro paciente"}`, http.StatusBadRequest)
					return
				}
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": id.String()})
}

func (h *Handler) ListPatientGuardians(w http.ResponseWriter, r *http.Request) {
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) && !h.canViewMedicalRecordAsGuardian(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	list, err := repo.GuardiansByPatient(r.Context(), h.Pool, patientID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type row struct {
		ID       string `json:"id"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Relation string `json:"relation"`
	}
	out := make([]row, len(list))
	for i := range list {
		out[i] = row{ID: list[i].ID.String(), FullName: list[i].FullName, Email: list[i].Email, Relation: list[i].Relation}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"guardians": out})
}

type SendContractRequest struct {
	GuardianID      string `json:"guardian_id"`
	TemplateID      string `json:"template_id"`
	DataInicio      string `json:"data_inicio"`      // opcional, formato YYYY-MM-DD
	DataFim         string `json:"data_fim"`         // opcional, formato YYYY-MM-DD
	Valor           string `json:"valor"`            // obrigatório, valor do serviço (placeholder [VALOR])
	Periodicidade   string `json:"periodicidade"`    // opcional, ex.: semanal (placeholder [PERIODICIDADE])
	SignPlace       string `json:"sign_place"`       // opcional, local de assinatura (placeholder [LOCAL])
	SignDate        string `json:"sign_date"`        // opcional, data prevista para assinatura YYYY-MM-DD (placeholder [DATA] até assinar)
	NumAppointments *int   `json:"num_appointments"` // opcional, quantidade de agendamentos a criar ao assinar (ex.: 4); null = sem limite
	ScheduleRules   []struct {
		DayOfWeek int    `json:"day_of_week"` // 0=domingo .. 6=sábado
		SlotTime  string `json:"slot_time"`   // "15:00"
	} `json:"schedule_rules"` // opcional, pré-agendamento (ex.: toda terça 15h)
}

func (h *Handler) SendContractForPatient(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && auth.RoleFrom(r.Context()) != auth.RoleSuperAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req SendContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.GuardianID == "" || req.TemplateID == "" {
		http.Error(w, `{"error":"guardian_id and template_id required"}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Valor) == "" {
		http.Error(w, `{"error":"valor é obrigatório"}`, http.StatusBadRequest)
		return
	}
	guardianID, err := uuid.Parse(req.GuardianID)
	if err != nil {
		http.Error(w, `{"error":"invalid guardian_id"}`, http.StatusBadRequest)
		return
	}
	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		http.Error(w, `{"error":"invalid template_id"}`, http.StatusBadRequest)
		return
	}
	tpl, err := repo.ContractTemplateByIDAndClinic(r.Context(), h.Pool, templateID, cid)
	if err != nil {
		http.Error(w, `{"error":"template not found"}`, http.StatusBadRequest)
		return
	}
	guardian, err := repo.LegalGuardianByID(r.Context(), h.Pool, guardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not found"}`, http.StatusBadRequest)
		return
	}
	_, err = repo.PatientGuardianByPatientAndGuardian(r.Context(), h.Pool, patientID, guardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not linked to patient"}`, http.StatusBadRequest)
		return
	}
	var profID *uuid.UUID
	if auth.RoleFrom(r.Context()) == auth.RoleProfessional {
		userID := auth.UserIDFrom(r.Context())
		if p, e := uuid.Parse(userID); e == nil {
			profID = &p
		}
	}
	var startDate, endDate *time.Time
	if req.DataInicio != "" {
		if t, err := time.Parse("2006-01-02", req.DataInicio); err == nil {
			startDate = &t
		}
	}
	if req.DataFim != "" {
		if t, err := time.Parse("2006-01-02", req.DataFim); err == nil {
			endDate = &t
		}
	}
	var valorPtr *string
	if req.Valor != "" {
		valorPtr = &req.Valor
	}
	var periodicidadePtr *string
	if strings.TrimSpace(req.Periodicidade) != "" {
		periodicidadePtr = &req.Periodicidade
	}
	var signPlacePtr *string
	if strings.TrimSpace(req.SignPlace) != "" {
		signPlacePtr = &req.SignPlace
	}
	var signDatePtr *time.Time
	if req.SignDate != "" {
		if t, err := time.Parse("2006-01-02", req.SignDate); err == nil {
			signDatePtr = &t
		}
	}
	var numAppointmentsPtr *int
	if req.NumAppointments != nil && *req.NumAppointments > 0 {
		numAppointmentsPtr = req.NumAppointments
	}
	contractID, err := repo.CreateContract(r.Context(), h.Pool, cid, patientID, guardianID, profID, templateID, "Responsável", false, tpl.Version, startDate, endDate, valorPtr, periodicidadePtr, signPlacePtr, signDatePtr, numAppointmentsPtr)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if len(req.ScheduleRules) > 0 {
		var rules []repo.ContractScheduleRule
		for _, r := range req.ScheduleRules {
			if r.DayOfWeek < 0 || r.DayOfWeek > 6 || r.SlotTime == "" {
				continue
			}
			t, err := time.Parse("15:04", r.SlotTime)
			if err != nil {
				continue
			}
			rules = append(rules, repo.ContractScheduleRule{ContractID: contractID, DayOfWeek: r.DayOfWeek, SlotTime: t})
		}
		if len(rules) > 0 {
			_ = repo.CreateContractScheduleRules(r.Context(), h.Pool, contractID, rules)
		}
	}
	accessToken, err := repo.CreateContractAccessToken(r.Context(), h.Pool, contractID, 7*24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	signURL := h.Cfg.AppPublicURL + "/sign-contract?token=" + accessToken
	if h.sendContractToSignEmail != nil {
		if err := h.sendContractToSignEmail(guardian.Email, guardian.FullName, signURL); err != nil {
			log.Printf("[send-contract] falha ao enviar e-mail para %s: %v (configure SMTP para receber o link)", guardian.Email, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Contrato enviado por e-mail.", "contract_id": contractID.String()})
}

// GetContractPreviewByID retorna o body_html de um contrato existente (para preview na lista, inclusive cancelados/encerrados).
func (h *Handler) GetContractPreviewByID(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, ok := h.ensureClinicID(r)
	if !ok {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	contractIDStr := mux.Vars(r)["contractId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid contract_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	c, err := repo.ContractByIDAndClinic(r.Context(), h.Pool, contractID, *cid)
	if err != nil {
		http.Error(w, `{"error":"contract not found"}`, http.StatusNotFound)
		return
	}
	if c.PatientID != patientID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	tpl, err := repo.ContractTemplateByID(r.Context(), h.Pool, c.TemplateID)
	if err != nil {
		http.Error(w, `{"error":"template not found"}`, http.StatusNotFound)
		return
	}
	patient, err := repo.PatientByID(r.Context(), h.Pool, c.PatientID)
	if err != nil {
		http.Error(w, `{"error":"patient not found"}`, http.StatusNotFound)
		return
	}
	guardian, err := repo.LegalGuardianByID(r.Context(), h.Pool, c.LegalGuardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not found"}`, http.StatusNotFound)
		return
	}
	clinic, errClinic := repo.ClinicByID(r.Context(), h.Pool, c.ClinicID)
	_ = errClinic
	contratado := ""
	if clinic != nil {
		contratado = clinic.Name
	}
	var signatureData *string
	var professionalName *string
	if c.ProfessionalID != nil {
		if prof, err := repo.ProfessionalByID(r.Context(), h.Pool, *c.ProfessionalID); err == nil {
			signatureData = prof.SignatureImageData
			professionalName = &prof.FullName
			if contratado != "" {
				contratado = prof.FullName + " – " + contratado
			} else {
				contratado = prof.FullName
			}
		}
	}
	dataInicioDisplay := ""
	if c.StartDate != nil {
		dataInicioDisplay = c.StartDate.Format("02/01/2006")
	}
	dataFimDisplay := ""
	if c.EndDate != nil {
		dataFimDisplay = c.EndDate.Format("02/01/2006")
	}
	valorStr := strPtrVal(c.Valor)
	periodicidadeDisplay := strPtrVal(c.Periodicidade)
	if periodicidadeDisplay == "" {
		periodicidadeDisplay = strPtrVal(tpl.Periodicidade)
	}
	objeto := strPtrVal(tpl.TipoServico)
	if objeto == "" {
		objeto = tpl.Name
	}
	rules, errRules := repo.ListContractScheduleRules(r.Context(), h.Pool, c.ID)
	_ = errRules
	consultasPrevistas := FormatScheduleRulesText(rules)
	guardianAddrStr := FormatGuardianAddressForContract(r.Context(), h.Pool, guardian)
	body := FillContractBody(tpl.BodyHTML, patient, guardian, contratado, objeto, strPtrVal(tpl.TipoServico), periodicidadeDisplay, valorStr, signatureData, professionalName, dataInicioDisplay, dataFimDisplay, "", consultasPrevistas, "", "", guardianAddrStr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"body_html": body})
}

// GetContractPreview retorna o body_html do modelo com placeholders preenchidos (paciente, responsável, contratado).
// Query: guardian_id, template_id. Apenas profissional ou super admin.
func (h *Handler) GetContractPreview(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, ok := h.ensureClinicID(r)
	if !ok {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	guardianIDStr := r.URL.Query().Get("guardian_id")
	templateIDStr := r.URL.Query().Get("template_id")
	dataInicioStr := r.URL.Query().Get("data_inicio")
	dataFimStr := r.URL.Query().Get("data_fim")
	valorStr := r.URL.Query().Get("valor")
	periodicidadeStr := r.URL.Query().Get("periodicidade")
	if guardianIDStr == "" || templateIDStr == "" {
		http.Error(w, `{"error":"guardian_id and template_id required"}`, http.StatusBadRequest)
		return
	}
	dataInicioDisplay := formatDateBR(dataInicioStr)
	dataFimDisplay := formatDateBR(dataFimStr)
	guardianID, err := uuid.Parse(guardianIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid guardian_id"}`, http.StatusBadRequest)
		return
	}
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid template_id"}`, http.StatusBadRequest)
		return
	}
	tpl, err := repo.ContractTemplateByIDAndClinic(r.Context(), h.Pool, templateID, *cid)
	if err != nil {
		http.Error(w, `{"error":"template not found"}`, http.StatusBadRequest)
		return
	}
	guardian, err := repo.LegalGuardianByID(r.Context(), h.Pool, guardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not found"}`, http.StatusBadRequest)
		return
	}
	_, err = repo.PatientGuardianByPatientAndGuardian(r.Context(), h.Pool, patientID, guardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not linked to patient"}`, http.StatusBadRequest)
		return
	}
	patient, err := repo.PatientByID(r.Context(), h.Pool, patientID)
	if err != nil {
		http.Error(w, `{"error":"patient not found"}`, http.StatusBadRequest)
		return
	}
	clinic, errClinic := repo.ClinicByID(r.Context(), h.Pool, *cid)
	_ = errClinic // best-effort for display
	contratado := ""
	if clinic != nil {
		contratado = clinic.Name
	}
	var signatureData *string
	var professionalName *string
	if auth.RoleFrom(r.Context()) == auth.RoleProfessional {
		userID := auth.UserIDFrom(r.Context())
		if profID, e := uuid.Parse(userID); e == nil {
			if prof, err := repo.ProfessionalByIDAndClinic(r.Context(), h.Pool, profID, *cid); err == nil {
				signatureData = prof.SignatureImageData
				professionalName = &prof.FullName
				if contratado != "" {
					contratado = prof.FullName + " – " + contratado
				} else {
					contratado = prof.FullName
				}
			}
		}
	}
	objeto := strPtrVal(tpl.TipoServico)
	if objeto == "" {
		objeto = tpl.Name
	}
	periodicidadeDisplay := periodicidadeStr
	if periodicidadeDisplay == "" {
		periodicidadeDisplay = strPtrVal(tpl.Periodicidade)
	}
	consultasPrevistas := ""
	guardianAddrStr := FormatGuardianAddressForContract(r.Context(), h.Pool, guardian)
	body := FillContractBody(tpl.BodyHTML, patient, guardian, contratado, objeto, strPtrVal(tpl.TipoServico), periodicidadeDisplay, valorStr, signatureData, professionalName, dataInicioDisplay, dataFimDisplay, "", consultasPrevistas, "", "", guardianAddrStr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"body_html": body})
}

// ListPatientContracts retorna os contratos enviados para o paciente (para a profissional ver status e reenviar/ver assinado).
func (h *Handler) ListPatientContracts(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, ok := h.ensureClinicID(r)
	if !ok {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	list, err := repo.ContractsByPatientAndClinic(r.Context(), h.Pool, patientID, *cid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		ID                string  `json:"id"`
		LegalGuardianID   string  `json:"legal_guardian_id"`
		GuardianName      string  `json:"guardian_name"`
		GuardianEmail     string  `json:"guardian_email"`
		TemplateName      string  `json:"template_name"`
		Status            string  `json:"status"`
		SignedAt          *string `json:"signed_at,omitempty"`
		VerificationToken *string `json:"verification_token,omitempty"`
		VerifyURL         string  `json:"verify_url,omitempty"`
	}
	out := make([]item, len(list))
	baseURL := h.Cfg.AppPublicURL
	if baseURL == "" {
		baseURL = r.URL.Scheme + "://" + r.Host
	}
	for i := range list {
		out[i] = item{
			ID:              list[i].ID.String(),
			LegalGuardianID: list[i].LegalGuardianID.String(),
			GuardianName:    list[i].GuardianName,
			GuardianEmail:   list[i].GuardianEmail,
			TemplateName:    list[i].TemplateName,
			Status:          list[i].Status,
		}
		if list[i].SignedAt != nil {
			s := list[i].SignedAt.Format(time.RFC3339)
			out[i].SignedAt = &s
		}
		if (list[i].Status == "SIGNED" || list[i].Status == "ENDED") && list[i].VerificationToken != nil && *list[i].VerificationToken != "" {
			out[i].VerificationToken = list[i].VerificationToken
			out[i].VerifyURL = baseURL + "/verify/" + *list[i].VerificationToken
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"contracts": out})
}

// ResendContract reenvia o e-mail com link para assinatura de um contrato ainda pendente.
func (h *Handler) ResendContract(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	contractIDStr := mux.Vars(r)["contractId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid contract_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	c, err := repo.ContractByIDAndClinic(r.Context(), h.Pool, contractID, cid)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if c.PatientID != patientID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if c.Status != "PENDING" {
		http.Error(w, `{"error":"contract already signed"}`, http.StatusBadRequest)
		return
	}
	guardian, err := repo.LegalGuardianByID(r.Context(), h.Pool, c.LegalGuardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not found"}`, http.StatusInternalServerError)
		return
	}
	accessToken, err := repo.CreateContractAccessToken(r.Context(), h.Pool, c.ID, 7*24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	signURL := ""
	if h.Cfg.AppPublicURL != "" {
		signURL = h.Cfg.AppPublicURL + "/sign-contract?token=" + accessToken
	}
	if h.sendContractToSignEmail != nil {
		if err := h.sendContractToSignEmail(guardian.Email, guardian.FullName, signURL); err != nil {
			log.Printf("[resend-contract] falha ao enviar e-mail para %s: %v", guardian.Email, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Contrato reenviado por e-mail."})
}

// CancelContract cancela um contrato (PENDING ou SIGNED), marca como inativo e envia e-mail ao responsável.
func (h *Handler) CancelContract(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	contractIDStr := mux.Vars(r)["contractId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid contract_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	c, err := repo.ContractByIDAndClinic(r.Context(), h.Pool, contractID, cid)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if c.PatientID != patientID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if c.Status == "CANCELLED" {
		http.Error(w, `{"error":"contrato já está cancelado"}`, http.StatusBadRequest)
		return
	}
	if c.Status != "PENDING" && c.Status != "SIGNED" {
		http.Error(w, `{"error":"contrato já encerrado ou cancelado"}`, http.StatusBadRequest)
		return
	}
	guardian, err := repo.LegalGuardianByID(r.Context(), h.Pool, c.LegalGuardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not found"}`, http.StatusInternalServerError)
		return
	}
	if err := repo.CancelContract(r.Context(), h.Pool, contractID, cid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Cancela os agendamentos vinculados a este contrato
	cancelledIDs, errAppt := repo.CancelAppointmentsByContractIDs(r.Context(), h.Pool, contractID)
	if errAppt != nil {
		log.Printf("[cancel-contract] falha ao cancelar agendamentos do contrato %s: %v", contractID, errAppt)
	}
	// Auditoria: contrato cancelado + batch de agendamentos cancelados
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if cc := auth.ClaimsFrom(r.Context()); cc != nil && cc.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*cc.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	resType := "CONTRACT"
	_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
		Action:                 "CONTRACT_CANCELLED",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               &cid,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           &resType,
		ResourceID:             &contractID,
		PatientID:              &patientID,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"changed_fields": []string{"status"}, "status": "CANCELLED"},
	})
	if len(cancelledIDs) > 0 {
		idStrs := make([]string, 0, len(cancelledIDs))
		for _, id := range cancelledIDs {
			idStrs = append(idStrs, id.String())
		}
		sys := "SYSTEM"
		_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
			Action:                 "APPOINTMENTS_CANCELLED_BATCH",
			ActorType:              "SYSTEM",
			ActorID:                nil,
			ClinicID:               &cid,
			RequestID:              r.Header.Get("X-Request-ID"),
			IP:                     r.RemoteAddr,
			UserAgent:              r.UserAgent(),
			ResourceType:           nil,
			ResourceID:             nil,
			PatientID:              &patientID,
			IsImpersonated:         false,
			ImpersonationSessionID: nil,
			Source:                 &sys,
			Severity:               &sev,
			Metadata:               map[string]interface{}{"contract_id": contractID.String(), "affected_ids": idStrs, "count": len(idStrs)},
		})
	}
	if h.sendContractCancelledEmail != nil {
		if err := h.sendContractCancelledEmail(guardian.Email, guardian.FullName); err != nil {
			log.Printf("[cancel-contract] falha ao enviar e-mail para %s: %v", guardian.Email, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	msg := "Contrato cancelado. O responsável foi notificado por e-mail."
	if len(cancelledIDs) > 0 {
		if len(cancelledIDs) == 1 {
			msg = "Contrato cancelado. 1 agendamento vinculado foi cancelado. O responsável foi notificado por e-mail."
		} else {
			msg = "Contrato cancelado. " + strconv.Itoa(len(cancelledIDs)) + " agendamentos vinculados foram cancelados. O responsável foi notificado por e-mail."
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": msg})
}

// SoftDeletePatient exclui um paciente (soft delete). Apenas SUPER_ADMIN.
func (h *Handler) SoftDeletePatient(w http.ResponseWriter, r *http.Request) {
	// Permite SUPER_ADMIN ou SUPER_ADMIN em modo impersonate (token terá Role=PROFESSIONAL, mas IsImpersonated=true).
	if !(auth.IsSuperAdmin(r.Context()) || auth.IsImpersonated(r.Context())) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	p, err := repo.PatientByID(r.Context(), h.Pool, patientID)
	if err != nil {
		http.Error(w, `{"error":"patient not found"}`, http.StatusNotFound)
		return
	}
	if err := repo.SoftDeletePatient(r.Context(), h.Pool, patientID, p.ClinicID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"patient not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Auditoria
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if cc := auth.ClaimsFrom(r.Context()); cc != nil && cc.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*cc.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	resType := "PATIENT"
	_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
		Action:                 "PATIENT_SOFT_DELETED",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               &p.ClinicID,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           &resType,
		ResourceID:             &patientID,
		PatientID:              &patientID,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"changed_fields": []string{"deleted_at"}},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Paciente excluído."})
}

// SoftDeleteGuardian exclui um responsável legal (soft delete). Apenas SUPER_ADMIN.
func (h *Handler) SoftDeleteGuardian(w http.ResponseWriter, r *http.Request) {
	// Permite SUPER_ADMIN ou SUPER_ADMIN em modo impersonate (token terá Role=PROFESSIONAL, mas IsImpersonated=true).
	if !(auth.IsSuperAdmin(r.Context()) || auth.IsImpersonated(r.Context())) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	guardianIDStr := mux.Vars(r)["guardianId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	guardianID, err := uuid.Parse(guardianIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid guardian_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	// Verifica se o responsável está vinculado ao paciente
	_, err = repo.PatientGuardianByPatientAndGuardian(r.Context(), h.Pool, patientID, guardianID)
	if err != nil {
		http.Error(w, `{"error":"guardian not found for patient"}`, http.StatusNotFound)
		return
	}
	if err := repo.SoftDeleteLegalGuardian(r.Context(), h.Pool, guardianID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"guardian not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Auditoria
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if cc := auth.ClaimsFrom(r.Context()); cc != nil && cc.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*cc.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	resType := "LEGAL_GUARDIAN"
	_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
		Action:                 "LEGAL_GUARDIAN_SOFT_DELETED",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               nil,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           &resType,
		ResourceID:             &guardianID,
		PatientID:              &patientID,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"changed_fields": []string{"deleted_at"}, "patient_id": patientID.String()},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Responsável excluído."})
}

// SoftDeleteContract exclui um contrato (soft delete). Apenas SUPER_ADMIN.
func (h *Handler) SoftDeleteContract(w http.ResponseWriter, r *http.Request) {
	// Permite SUPER_ADMIN ou SUPER_ADMIN em modo impersonate (token terá Role=PROFESSIONAL, mas IsImpersonated=true).
	if !(auth.IsSuperAdmin(r.Context()) || auth.IsImpersonated(r.Context())) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	contractIDStr := mux.Vars(r)["contractId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid contract_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var c *repo.Contract
	clinicIDStr := auth.ClinicIDFrom(r.Context())
	if clinicIDStr != nil && *clinicIDStr != "" {
		cid, err := uuid.Parse(*clinicIDStr)
		if err != nil {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		c, err = repo.ContractByIDAndClinic(r.Context(), h.Pool, contractID, cid)
		if err != nil {
			http.Error(w, `{"error":"contract not found"}`, http.StatusNotFound)
			return
		}
	} else {
		c, err = repo.ContractByID(r.Context(), h.Pool, contractID)
		if err != nil {
			http.Error(w, `{"error":"contract not found"}`, http.StatusNotFound)
			return
		}
	}
	cid := c.ClinicID
	if c.PatientID != patientID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if err := repo.SoftDeleteContract(r.Context(), h.Pool, contractID, cid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"contract not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Auditoria
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if cc := auth.ClaimsFrom(r.Context()); cc != nil && cc.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*cc.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	resType := "CONTRACT"
	_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
		Action:                 "CONTRACT_SOFT_DELETED",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               &cid,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           &resType,
		ResourceID:             &contractID,
		PatientID:              &patientID,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"changed_fields": []string{"deleted_at"}},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Contrato excluído."})
}
