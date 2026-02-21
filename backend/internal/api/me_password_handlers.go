package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/repo"
	"gorm.io/gorm"
)

type ChangeMyPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFrom(r.Context())
	if role != auth.RoleProfessional && role != auth.RoleSuperAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
		return
	}

	var req ChangeMyPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	req.CurrentPassword = strings.TrimSpace(req.CurrentPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, `{"error":"current_password and new_password required"}`, http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		http.Error(w, `{"error":"new_password deve ter no mínimo 8 caracteres"}`, http.StatusBadRequest)
		return
	}

	var currentHash string
	switch role {
	case auth.RoleProfessional:
		p, err := repo.ProfessionalByID(r.Context(), h.DB, uid)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		currentHash = p.PasswordHash
	case auth.RoleSuperAdmin:
		s, err := repo.SuperAdminByID(r.Context(), h.DB, uid)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		currentHash = s.PasswordHash
	default:
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if !auth.CheckPassword(currentHash, req.CurrentPassword) {
		http.Error(w, `{"error":"senha atual inválida"}`, http.StatusBadRequest)
		return
	}

	hash, err := h.hashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	var q string
	switch role {
	case auth.RoleProfessional:
		q = "UPDATE professionals SET password_hash = ?, updated_at = now() WHERE id = ?"
	case auth.RoleSuperAdmin:
		q = "UPDATE super_admins SET password_hash = ?, updated_at = now() WHERE id = ?"
	default:
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if err := h.DB.WithContext(r.Context()).Exec(q, hash, uid).Error; err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Senha atualizada."})
}
