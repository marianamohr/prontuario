package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
)

type CreateInviteRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req CreateInviteRequest
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
	// Evita criar múltiplos convites para o mesmo e-mail se o profissional já existe.
	if _, err := repo.ProfessionalByEmail(r.Context(), h.Pool, req.Email); err == nil {
		http.Error(w, `{"error":"professional já existe para este email"}`, http.StatusConflict)
		return
	}
	// Cria uma clinic interna para este profissional (modelo 1:1).
	var clinicID uuid.UUID
	err := h.Pool.QueryRow(r.Context(), "INSERT INTO clinics (name) VALUES ($1) RETURNING id", req.FullName).Scan(&clinicID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	inv, err := repo.CreateProfessionalInvite(r.Context(), h.Pool, req.Email, req.FullName, clinicID, expiresAt)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	registerURL := h.Cfg.AppPublicURL + "/register?token=" + inv.Token
	if h.sendInviteEmail != nil {
		_ = h.sendInviteEmail(req.Email, req.FullName, registerURL)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Convite enviado por e-mail.",
		"invite_id":  inv.ID.String(),
		"expires_at": inv.ExpiresAt,
	})
}

func (h *Handler) GetInviteByToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	inv, err := repo.GetProfessionalInviteByToken(r.Context(), h.Pool, token)
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email":       inv.Email,
		"full_name":   inv.FullName,
		"clinic_name": clinicName,
		"expires_at":  inv.ExpiresAt,
	})
}

type AcceptInviteRequest struct {
	Token         string  `json:"token"`
	Password      string  `json:"password"`
	FullName      string  `json:"full_name"`
	TradeName     string  `json:"trade_name"`
	BirthDate     *string `json:"birth_date"`
	CPF           string  `json:"cpf"`
	Address       string  `json:"address"`
	MaritalStatus string  `json:"marital_status"`
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Token == "" || req.Password == "" {
		http.Error(w, `{"error":"token and password required"}`, http.StatusBadRequest)
		return
	}
	if req.CPF == "" {
		http.Error(w, `{"error":"cpf obrigatorio"}`, http.StatusBadRequest)
		return
	}
	inv, err := repo.GetProfessionalInviteByToken(r.Context(), h.Pool, req.Token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
		return
	}
	if inv.Status != "PENDING" || inv.ExpiresAt.Before(time.Now()) {
		http.Error(w, `{"error":"invite already used or expired"}`, http.StatusBadRequest)
		return
	}
	passwordHash, err := h.hashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	tradeName := strings.TrimSpace(req.TradeName)
	n := crypto.NormalizeCPF(req.CPF)
	if len(n) != 11 {
		http.Error(w, `{"error":"cpf invalido"}`, http.StatusBadRequest)
		return
	}
	ch := crypto.CPFHash(n)
	cpfHash := &ch
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
	var address, maritalStatus *string
	if req.Address != "" {
		address = &req.Address
	}
	if req.MaritalStatus != "" {
		maritalStatus = &req.MaritalStatus
	}
	if err := repo.AcceptProfessionalInvite(r.Context(), h.Pool, inv.ID, passwordHash, req.FullName, tradeName, req.BirthDate, cpfEnc, nonce, &keyVer, cpfHash, address, maritalStatus); err != nil {
		http.Error(w, `{"error":"could not complete registration"}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Cadastro concluído. Faça login na área do profissional."})
}
