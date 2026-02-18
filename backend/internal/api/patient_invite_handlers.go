package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
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
		http.Error(w, `{"error":"email inválido"}`, http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	inv, err := repo.CreatePatientInvite(r.Context(), h.Pool, cid, req.Email, req.FullName, expiresAt)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	registerURL := h.Cfg.AppPublicURL + "/register-patient?token=" + inv.Token
	if h.sendPatientInviteEmail != nil {
		_ = h.sendPatientInviteEmail(req.Email, req.FullName, registerURL)
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
	inv, err := repo.GetPatientInviteByToken(r.Context(), h.Pool, token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusNotFound)
		return
	}
	if inv.Status != "PENDING" || inv.ExpiresAt.Before(time.Now()) {
		http.Error(w, `{"error":"invite already used or expired"}`, http.StatusBadRequest)
		return
	}
	var clinicName string
	_ = h.Pool.QueryRow(r.Context(), "SELECT name FROM clinics WHERE id = $1", inv.ClinicID).Scan(&clinicName)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"email":       inv.GuardianEmail,
		"full_name":   inv.GuardianFullName,
		"clinic_name": clinicName,
		"expires_at":  inv.ExpiresAt,
	})
}

type AcceptPatientInviteRequest struct {
	Token             string `json:"token"`
	SamePerson        bool   `json:"same_person"`
	GuardianFullName  string `json:"guardian_full_name"`
	GuardianCPF       string `json:"guardian_cpf"`
	GuardianAddress   string `json:"guardian_address"`
	GuardianBirthDate string `json:"guardian_birth_date"`
	PatientFullName   string `json:"patient_full_name"`
	PatientBirthDate  string `json:"patient_birth_date"`
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
	req.GuardianAddress = strings.TrimSpace(req.GuardianAddress)
	req.GuardianBirthDate = strings.TrimSpace(req.GuardianBirthDate)
	req.PatientFullName = strings.TrimSpace(req.PatientFullName)
	req.PatientBirthDate = strings.TrimSpace(req.PatientBirthDate)

	if req.Token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	if req.GuardianFullName == "" || req.GuardianCPF == "" || req.GuardianAddress == "" || req.GuardianBirthDate == "" || req.PatientBirthDate == "" {
		http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
		return
	}

	inv, err := repo.GetPatientInviteByToken(r.Context(), h.Pool, req.Token)
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
		http.Error(w, `{"error":"cpf inválido"}`, http.StatusBadRequest)
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
	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Recarrega o invite na TX (evita race).
	var inviteID uuid.UUID
	var guardianEmail string
	var clinicID uuid.UUID
	err = tx.QueryRow(r.Context(), `
		SELECT id, guardian_email, clinic_id
		FROM patient_invites
		WHERE id = $1 AND status = 'PENDING' AND expires_at > now()
	`, inv.ID).Scan(&inviteID, &guardianEmail, &clinicID)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
		return
	}

	// Upsert do responsável por email.
	var guardianID uuid.UUID
	err = tx.QueryRow(r.Context(), `SELECT id FROM legal_guardians WHERE email = $1 AND deleted_at IS NULL`, guardianEmail).Scan(&guardianID)
	if err != nil {
		if err != pgx.ErrNoRows {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		// Cria novo.
		guardianID = uuid.New()
		_, err = tx.Exec(r.Context(), `
			INSERT INTO legal_guardians (id, email, full_name, cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address, birth_date, auth_provider, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'LOCAL'::auth_provider_enum, 'ACTIVE')
		`, guardianID, guardianEmail, req.GuardianFullName, cpfEnc, nonce, keyVer, cpfHash, req.GuardianAddress, req.GuardianBirthDate)
		if err != nil {
			http.Error(w, `{"error":"email ou cpf já utilizado"}`, http.StatusBadRequest)
			return
		}
	} else {
		// Atualiza dados complementares do existente (mantém GoogleSub/senha).
		_, err = tx.Exec(r.Context(), `
			UPDATE legal_guardians
			SET full_name = COALESCE(NULLIF($1::text, ''), full_name),
			    cpf_encrypted = $2,
			    cpf_nonce = $3,
			    cpf_key_version = $4::text,
			    cpf_hash = $5::text,
			    address = $6::text,
			    birth_date = NULLIF($7::text, '')::date,
			    updated_at = now()
			WHERE id = $8 AND deleted_at IS NULL
		`, req.GuardianFullName, cpfEnc, nonce, keyVer, cpfHash, req.GuardianAddress, req.GuardianBirthDate, guardianID)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
	}

	patientName := req.PatientFullName
	if req.SamePerson || patientName == "" {
		patientName = req.GuardianFullName
	}

	patientID := uuid.New()
	_, err = tx.Exec(r.Context(), `
		INSERT INTO patients (id, clinic_id, full_name, birth_date)
		VALUES ($1, $2, $3, NULLIF($4::text, '')::date)
	`, patientID, clinicID, patientName, req.PatientBirthDate)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	relation := "Titular"
	if !req.SamePerson && req.PatientFullName != "" {
		relation = "Responsável"
	}
	_, err = tx.Exec(r.Context(), `
		INSERT INTO patient_guardians (patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts)
		VALUES ($1, $2, $3, true, true)
	`, patientID, guardianID, relation)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(r.Context(), `UPDATE patient_invites SET status = 'ACCEPTED', updated_at = now() WHERE id = $1`, inviteID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Cadastro do paciente concluído."})
}
