package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/repo"
)

type CreateSuperAdminInviteRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

func (h *Handler) CreateSuperAdminInvite(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req CreateSuperAdminInviteRequest
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
	// Evita criar convite se já existe super admin para o e-mail.
	if _, err := repo.SuperAdminByEmail(r.Context(), h.DB, req.Email); err == nil {
		http.Error(w, `{"error":"super admin já existe para este email"}`, http.StatusConflict)
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	inv, err := repo.CreateSuperAdminInvite(r.Context(), h.DB, req.Email, req.FullName, expiresAt)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	registerURL := h.Cfg.AppPublicURL + "/register-super-admin?token=" + inv.Token
	if h.sendSuperAdminInviteEmail != nil {
		log.Printf("[super-admin-invite] enviando e-mail de convite para %s", req.Email)
		if err := h.sendSuperAdminInviteEmail(req.Email, req.FullName, registerURL); err != nil {
			log.Printf("[super-admin-invite] falha ao enviar e-mail para %s: %v", req.Email, err)
		}
	} else {
		log.Printf("[super-admin-invite] invite created for %s but email disabled (SMTP/APP_PUBLIC_URL)", req.Email)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Convite enviado por e-mail.",
		"invite_id":  inv.ID.String(),
		"expires_at": inv.ExpiresAt,
	})
}

func (h *Handler) ListSuperAdminInvites(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	limit, offset := ParseLimitOffset(r)
	list, total, err := repo.ListSuperAdminInvitesPaginated(r.Context(), h.DB, limit, offset)
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":  out,
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

func (h *Handler) DeleteSuperAdminInvite(w http.ResponseWriter, r *http.Request) {
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
	if err := repo.DeleteSuperAdminInvite(r.Context(), h.DB, id); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Convite removido."})
}

func (h *Handler) ResendSuperAdminInvite(w http.ResponseWriter, r *http.Request) {
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
	inv, err := repo.GetSuperAdminInviteByID(r.Context(), h.DB, id)
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
	registerURL := h.Cfg.AppPublicURL + "/register-super-admin?token=" + inv.Token
	if h.sendSuperAdminInviteEmail != nil {
		log.Printf("[super-admin-invite] reenviando convite para %s", inv.Email)
		if err := h.sendSuperAdminInviteEmail(inv.Email, inv.FullName, registerURL); err != nil {
			log.Printf("[super-admin-invite] falha ao reenviar e-mail para %s: %v", inv.Email, err)
		}
	} else {
		log.Printf("[super-admin-invite] resend disabled (would send to %s); set APP_PUBLIC_URL and SMTP", inv.Email)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Convite reenviado por e-mail."})
}

func (h *Handler) GetSuperAdminInviteByToken(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	inv, err := repo.GetSuperAdminInviteByToken(r.Context(), h.DB, token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusNotFound)
		return
	}
	if inv.Status != "PENDING" || inv.ExpiresAt.Before(time.Now()) {
		http.Error(w, `{"error":"invite already used or expired"}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"email":      inv.Email,
		"full_name":  inv.FullName,
		"expires_at": inv.ExpiresAt,
	})
}

type AcceptSuperAdminInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func (h *Handler) AcceptSuperAdminInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptSuperAdminInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.FullName = strings.TrimSpace(req.FullName)
	if req.Token == "" || req.Password == "" {
		http.Error(w, `{"error":"token and password required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must have at least 8 characters"}`, http.StatusBadRequest)
		return
	}
	inv, err := repo.GetSuperAdminInviteByToken(r.Context(), h.DB, req.Token)
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
	if err := repo.AcceptSuperAdminInvite(r.Context(), h.DB, inv.ID, passwordHash, req.FullName); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			http.Error(w, `{"error":"email já utilizado"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"could not complete registration"}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Cadastro concluído. Faça login como super admin."})
}
