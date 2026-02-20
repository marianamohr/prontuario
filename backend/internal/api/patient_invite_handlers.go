package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
	"gorm.io/gorm"
)

type CreatePatientInviteRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

func (h *Handler) CreatePatientInvite(w http.ResponseWriter, r *http.Request) {
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

	var req CreatePatientInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	if req.Email == "" || req.FullName == "" {
		http.Error(w, `{"error":"email and full_name required"}`, http.StatusBadRequest)
		return
	}
	if !emailRegex.MatchString(req.Email) {
		http.Error(w, `{"error":"invalid email"}`, http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	inv, err := repo.CreatePatientInvite(r.Context(), h.DB, cid, req.Email, req.FullName, expiresAt)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	registerURL := h.Cfg.AppPublicURL + "/register-patient?token=" + inv.Token
	if h.sendPatientInviteEmail != nil {
		log.Printf("[patient-invite] enviando convite de paciente para %s", req.Email)
		if err := h.sendPatientInviteEmail(req.Email, req.FullName, registerURL); err != nil {
			log.Printf("[patient-invite] falha ao enviar e-mail para %s: %v", req.Email, err)
		}
	} else {
		log.Printf("[patient-invite] email disabled (would send to %s)", req.Email)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Convite enviado por e-mail.",
		"invite_id":  inv.ID.String(),
		"expires_at": inv.ExpiresAt,
	})
}

func (h *Handler) GetPatientInviteByToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	inv, err := repo.GetPatientInviteByToken(r.Context(), h.DB, token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusNotFound)
		return
	}
	if inv.Status != "PENDING" || inv.ExpiresAt.Before(time.Now()) {
		http.Error(w, `{"error":"invite already used or expired"}`, http.StatusBadRequest)
		return
	}
	var clinicName string
	_ = h.DB.WithContext(r.Context()).Raw("SELECT name FROM clinics WHERE id = ?", inv.ClinicID).Scan(&clinicName)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"email":       inv.GuardianEmail,
		"full_name":   inv.GuardianFullName,
		"clinic_name": clinicName,
		"expires_at":  inv.ExpiresAt,
	})
}

type AcceptPatientInviteRequest struct {
	Token             string      `json:"token"`
	SamePerson        bool        `json:"same_person"`
	GuardianFullName  string      `json:"guardian_full_name"`
	GuardianCPF       string      `json:"guardian_cpf"`
	GuardianAddress   interface{} `json:"guardian_address"` // objeto (8 campos) ou string (8 linhas)
	GuardianBirthDate string      `json:"guardian_birth_date"`
	PatientFullName   string      `json:"patient_full_name"`
	PatientBirthDate  string      `json:"patient_birth_date"`
}

func (h *Handler) AcceptPatientInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptPatientInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.GuardianFullName = strings.TrimSpace(req.GuardianFullName)
	req.GuardianCPF = strings.TrimSpace(req.GuardianCPF)
	req.GuardianBirthDate = strings.TrimSpace(req.GuardianBirthDate)
	req.PatientFullName = strings.TrimSpace(req.PatientFullName)
	req.PatientBirthDate = strings.TrimSpace(req.PatientBirthDate)

	if req.Token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	if req.GuardianFullName == "" || req.GuardianCPF == "" || req.GuardianBirthDate == "" || req.PatientBirthDate == "" {
		http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
		return
	}

	// guardian_address: objeto (8 campos) ou string (8 linhas)
	addrInput, err := parseAddressFromRequest(req.GuardianAddress)
	if err != nil {
		http.Error(w, `{"error":"guardian_address invalid: use object with 8 fields or 8-line string (street, number, complement, neighborhood, city, state, country, zip)"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateAddress(addrInput); err != nil {
		http.Error(w, `{"error":"address invalid (8-digit ZIP; street, neighborhood, city, state, country required)"}`, http.StatusBadRequest)
		return
	}

	inv, err := repo.GetPatientInviteByToken(r.Context(), h.DB, req.Token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
		return
	}
	if inv.Status != "PENDING" || inv.ExpiresAt.Before(time.Now()) {
		http.Error(w, `{"error":"invite already used or expired"}`, http.StatusBadRequest)
		return
	}

	// Normaliza/valida CPF e prepara criptografia.
	cpfNorm := crypto.NormalizeCPF(req.GuardianCPF)
	if len(cpfNorm) != 11 {
		http.Error(w, `{"error":"invalid CPF"}`, http.StatusBadRequest)
		return
	}
	cpfHash := crypto.CPFHash(cpfNorm)
	keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
	if err != nil {
		http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
		return
	}
	keyVer := h.Cfg.CurrentDataKeyVer
	if keyVer == "" {
		keyVer = "v1"
	}
	cpfEnc, nonce, err := crypto.Encrypt([]byte(cpfNorm), keyVer, keysMap)
	if err != nil {
		http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
		return
	}

	// Transação: upsert guardião + cria paciente + vínculo + aceita invite.
	addr := AddressInputToRepo(addrInput)
	err = h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var row struct {
			InviteID      uuid.UUID
			GuardianEmail string
			ClinicID      uuid.UUID
		}
		if err := tx.Raw(`
			SELECT id as invite_id, guardian_email, clinic_id
			FROM patient_invites
			WHERE id = ? AND status = 'PENDING' AND expires_at > now()
		`, inv.ID).Scan(&row).Error; err != nil || row.InviteID == uuid.Nil {
			return errors.New("invalid or expired token")
		}
		addressID, err := repo.CreateAddressTx(r.Context(), tx, addr)
		if err != nil {
			return err
		}
		var guardianID uuid.UUID
		if err := tx.Raw(`SELECT id FROM legal_guardians WHERE email = ? AND deleted_at IS NULL`, row.GuardianEmail).Scan(&guardianID).Error; err != nil || guardianID == uuid.Nil {
			guardianID = uuid.New()
			if err := tx.Exec(`
				INSERT INTO legal_guardians (id, email, full_name, cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date, auth_provider, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'LOCAL'::auth_provider_enum, 'ACTIVE')
			`, guardianID, row.GuardianEmail, req.GuardianFullName, cpfEnc, nonce, keyVer, cpfHash, addressID, req.GuardianBirthDate).Error; err != nil {
				return errors.New("email ou cpf já utilizado")
			}
		} else {
			if err := tx.Exec(`
				UPDATE legal_guardians
				SET full_name = COALESCE(NULLIF(?::text, ''), full_name),
				    cpf_encrypted = ?, cpf_nonce = ?, cpf_key_version = ?::text, cpf_hash = ?::text,
				    address_id = ?, birth_date = NULLIF(?::text, '')::date, updated_at = now()
				WHERE id = ? AND deleted_at IS NULL
			`, req.GuardianFullName, cpfEnc, nonce, keyVer, cpfHash, addressID, req.GuardianBirthDate, guardianID).Error; err != nil {
				return err
			}
		}
		patientName := req.PatientFullName
		if req.SamePerson || patientName == "" {
			patientName = req.GuardianFullName
		}
		patientID := uuid.New()
		if err := tx.Exec(`INSERT INTO patients (id, clinic_id, full_name, birth_date) VALUES (?, ?, ?, NULLIF(?::text, '')::date)`, patientID, row.ClinicID, patientName, req.PatientBirthDate).Error; err != nil {
			return err
		}
		relation := "Titular"
		if !req.SamePerson && req.PatientFullName != "" {
			relation = "Responsável"
		}
		if err := tx.Exec(`INSERT INTO patient_guardians (patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts) VALUES (?, ?, ?, true, true)`, patientID, guardianID, relation).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE patient_invites SET status = 'ACCEPTED', updated_at = now() WHERE id = ?`, row.InviteID).Error
	})
	if err != nil {
		if err.Error() == "invalid or expired token" {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
			return
		}
		if err.Error() == "email ou cpf já utilizado" {
			http.Error(w, `{"error":"email ou cpf já utilizado"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Cadastro do paciente concluído."})
}
