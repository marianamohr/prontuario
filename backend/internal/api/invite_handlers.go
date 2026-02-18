package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
		log.Printf("[invite] enviando e-mail de convite para %s", req.Email)
		if err := h.sendInviteEmail(req.Email, req.FullName, registerURL); err != nil {
			log.Printf("[invite] falha ao enviar e-mail para %s: %v", req.Email, err)
		}
	} else {
		log.Printf("[invite] convite criado para %s mas envio de e-mail desativado (SMTP/APP_PUBLIC_URL)", req.Email)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Convite enviado por e-mail.",
		"invite_id":  inv.ID.String(),
		"expires_at": inv.ExpiresAt,
	})
}

// ListInvites returns all professional invites for backoffice (super admin only).
func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	list, err := repo.ListProfessionalInvites(r.Context(), h.Pool)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, inv := range list {
		out = append(out, map[string]interface{}{
			"id":         inv.ID.String(),
			"email":      inv.Email,
			"full_name":  inv.FullName,
			"status":     inv.Status,
			"expires_at": inv.ExpiresAt,
			"created_at": inv.CreatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// DeleteInvite removes a professional invite by id (super admin only).
func (h *Handler) DeleteInvite(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	if err := repo.DeleteProfessionalInvite(r.Context(), h.Pool, id); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Convite removido."})
}

// ResendInvite sends the invite email again for a PENDING, non-expired invite (super admin only).
func (h *Handler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	inv, err := repo.GetProfessionalInviteByID(r.Context(), h.Pool, id)
	if err != nil {
		http.Error(w, `{"error":"invite not found"}`, http.StatusNotFound)
		return
	}
	if inv.Status != "PENDING" {
		http.Error(w, `{"error":"só é possível reenviar convite pendente"}`, http.StatusBadRequest)
		return
	}
	if inv.ExpiresAt.Before(time.Now()) {
		http.Error(w, `{"error":"convite expirado"}`, http.StatusBadRequest)
		return
	}
	registerURL := h.Cfg.AppPublicURL + "/register?token=" + inv.Token
	if h.sendInviteEmail != nil {
		log.Printf("[invite] reenviando convite para %s", inv.Email)
		if err := h.sendInviteEmail(inv.Email, inv.FullName, registerURL); err != nil {
			log.Printf("[invite] falha ao reenviar e-mail para %s: %v", inv.Email, err)
		}
	} else {
		log.Printf("[invite] reenvio desativado (destinatário seria %s); configure APP_PUBLIC_URL e SMTP", inv.Email)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Convite reenviado por e-mail."})
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
	errSc := h.Pool.QueryRow(r.Context(), "SELECT name FROM clinics WHERE id = $1", inv.ClinicID).Scan(&clinicName)
	_ = errSc
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"email":       inv.Email,
		"full_name":   inv.FullName,
		"clinic_name": clinicName,
		"expires_at":  inv.ExpiresAt,
	})
}

type AcceptInviteRequest struct {
	Token         string      `json:"token"`
	Password      string      `json:"password"`
	FullName      string      `json:"full_name"`
	TradeName     string      `json:"trade_name"`
	BirthDate     *string     `json:"birth_date"`
	CPF           string      `json:"cpf"`
	Address       interface{} `json:"address"` // objeto (8 campos) ou string (8 linhas) — obrigatório
	MaritalStatus string      `json:"marital_status"`
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
	addrInput, err := parseAddressFromRequest(req.Address)
	if err != nil {
		http.Error(w, `{"error":"endereço obrigatório: use objeto com 8 campos ou string de 8 linhas (rua, numero, complemento, bairro, cidade, estado, pais, cep)"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateAddress(addrInput); err != nil {
		http.Error(w, `{"error":"endereço inválido (CEP 8 dígitos; rua, bairro, cidade, estado, país obrigatórios)"}`, http.StatusBadRequest)
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
		log.Printf("[invite] accept: DATA_ENCRYPTION_KEYS inválida: %v", err)
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
	addr := AddressInputToRepo(addrInput)
	addressID, err := repo.CreateAddress(r.Context(), h.Pool, addr)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	var maritalStatus *string
	if req.MaritalStatus != "" {
		maritalStatus = &req.MaritalStatus
	}
	if err := repo.AcceptProfessionalInvite(r.Context(), h.Pool, inv.ID, passwordHash, req.FullName, tradeName, req.BirthDate, cpfEnc, nonce, &keyVer, cpfHash, &addressID, maritalStatus); err != nil {
		http.Error(w, `{"error":"could not complete registration"}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Cadastro concluído. Faça login na área do profissional."})
}
