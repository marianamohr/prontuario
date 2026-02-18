package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
)

const authProviderLocal = "LOCAL"

// emailRegex está definido em patients_handlers.go (pacote api)

type GuardianRegisterRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
	CPF      string `json:"cpf"`
}

type GuardianLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) GuardianRegister(w http.ResponseWriter, r *http.Request) {
	var req GuardianRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.FullName == "" || len(req.Password) < 8 {
		http.Error(w, `{"error":"email, full_name and password (min 8) required"}`, http.StatusBadRequest)
		return
	}
	if !emailRegex.MatchString(strings.TrimSpace(req.Email)) {
		http.Error(w, `{"error":"email inválido"}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CPF) == "" {
		http.Error(w, `{"error":"cpf obrigatório"}`, http.StatusBadRequest)
		return
	}
	cpfNorm := crypto.NormalizeCPF(req.CPF)
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
	passHash, err := h.hashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	g := &repo.LegalGuardian{
		Email:         req.Email,
		FullName:      req.FullName,
		PasswordHash:  &passHash,
		CPFEncrypted:  cpfEnc,
		CPFNonce:      nonce,
		CPFKeyVersion: &keyVer,
		CPFHash:       &cpfHash,
		AuthProvider:  authProviderLocal,
		Status:        "ACTIVE",
	}
	err = repo.CreateLegalGuardian(r.Context(), h.Pool, g)
	if err != nil {
		if isUniqueViolation(err) {
			genericLoginError(w)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	tok, err := h.issueGuardianJWT(g.ID.String(), nil, false, nil)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     tok,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       g.ID.String(),
			Email:    g.Email,
			FullName: g.FullName,
			Role:     auth.RoleLegalGuardian,
		},
	})
}

func (h *Handler) GuardianLogin(w http.ResponseWriter, r *http.Request) {
	var req GuardianLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}
	g, err := repo.LegalGuardianByEmail(r.Context(), h.Pool, req.Email)
	if err != nil {
		genericLoginError(w)
		return
	}
	if g.PasswordHash == nil {
		genericLoginError(w)
		return
	}
	if !auth.CheckPassword(*g.PasswordHash, req.Password) {
		genericLoginError(w)
		return
	}
	tok, err := h.issueGuardianJWT(g.ID.String(), nil, false, nil)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     tok,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       g.ID.String(),
			Email:    g.Email,
			FullName: g.FullName,
			Role:     auth.RoleLegalGuardian,
		},
	})
}

func (h *Handler) issueGuardianJWT(userID string, clinicID *string, isImpersonated bool, impersonationSessionID *string) (string, error) {
	return auth.BuildJWT(h.Cfg.JWTSecret, userID, auth.RoleLegalGuardian, clinicID, isImpersonated, impersonationSessionID, 24*time.Hour)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
