package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prontuario/backend/internal/repo"
)

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Se o e-mail existir, você receberá instruções."}`))
		return
	}
	const exp = time.Hour
	if prof, err := repo.ProfessionalByEmail(r.Context(), h.Pool, req.Email); err == nil {
		tok, errTok := repo.CreatePasswordResetToken(r.Context(), h.Pool, "PROFESSIONAL", prof.ID, exp)
		_ = errTok
		if tok != "" {
			if h.sendPasswordResetEmail != nil {
				log.Printf("[password-reset] enviando para %s (tipo=PROFESSIONAL)", req.Email)
				if errSend := h.sendPasswordResetEmail(req.Email, tok); errSend != nil {
					log.Printf("[password-reset] falha ao enviar e-mail para %s: %v", req.Email, errSend)
				}
			} else {
				log.Printf("[password-reset] envio desativado (destinatário seria %s, tipo=PROFESSIONAL)", req.Email)
			}
		}
	}
	if admin, err := repo.SuperAdminByEmail(r.Context(), h.Pool, req.Email); err == nil {
		tok, errTok := repo.CreatePasswordResetToken(r.Context(), h.Pool, "SUPER_ADMIN", admin.ID, exp)
		_ = errTok
		if tok != "" {
			if h.sendPasswordResetEmail != nil {
				log.Printf("[password-reset] enviando para %s (tipo=SUPER_ADMIN)", req.Email)
				if errSend := h.sendPasswordResetEmail(req.Email, tok); errSend != nil {
					log.Printf("[password-reset] falha ao enviar e-mail para %s: %v", req.Email, errSend)
				}
			} else {
				log.Printf("[password-reset] envio desativado (destinatário seria %s, tipo=SUPER_ADMIN)", req.Email)
			}
		}
	}
	if g, err := repo.LegalGuardianByEmail(r.Context(), h.Pool, req.Email); err == nil {
		tok, errTok := repo.CreatePasswordResetToken(r.Context(), h.Pool, "LEGAL_GUARDIAN", g.ID, exp)
		_ = errTok
		if tok != "" {
			if h.sendPasswordResetEmail != nil {
				log.Printf("[password-reset] enviando para %s (tipo=LEGAL_GUARDIAN)", req.Email)
				if errSend := h.sendPasswordResetEmail(req.Email, tok); errSend != nil {
					log.Printf("[password-reset] falha ao enviar e-mail para %s: %v", req.Email, errSend)
				}
			} else {
				log.Printf("[password-reset] envio desativado (destinatário seria %s, tipo=LEGAL_GUARDIAN)", req.Email)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Se o e-mail existir, você receberá instruções."}`))
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Token == "" || len(req.NewPassword) < 8 {
		http.Error(w, `{"error":"token and new_password (min 8 chars) required"}`, http.StatusBadRequest)
		return
	}
	userType, userID, err := repo.ConsumePasswordResetToken(r.Context(), h.Pool, req.Token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
		return
	}
	if h.hashPassword == nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	hash, err := h.hashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	switch userType {
	case "PROFESSIONAL":
		_, err = h.Pool.Exec(r.Context(), "UPDATE professionals SET password_hash = $1, updated_at = now() WHERE id = $2", hash, userID)
	case "SUPER_ADMIN":
		_, err = h.Pool.Exec(r.Context(), "UPDATE super_admins SET password_hash = $1, updated_at = now() WHERE id = $2", hash, userID)
	case "LEGAL_GUARDIAN":
		_, err = h.Pool.Exec(r.Context(), "UPDATE legal_guardians SET password_hash = $2, updated_at = now() WHERE id = $1", userID, hash)
	default:
		http.Error(w, `{"error":"invalid token"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"message":"Senha alterada com sucesso."}`))
}
