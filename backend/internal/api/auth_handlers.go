package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/config"
	"github.com/prontuario/backend/internal/repo"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Role     string  `json:"role"`
	ClinicID *string `json:"clinic_id,omitempty"`
}

type Handler struct {
	Pool                       *pgxpool.Pool
	Cfg                        *config.Config
	hashPassword               func(string) (string, error)
	sendPasswordResetEmail     func(to, token string) error
	sendContractSignedEmail    func(to, name string, pdf []byte, verificationToken string) error
	sendInviteEmail            func(to, fullName, registerURL string) error
	sendPatientInviteEmail     func(to, fullName, registerURL string) error
	sendContractToSignEmail    func(to, fullName, signURL string) error
	sendContractCancelledEmail func(to, fullName string) error
	sendContractEndedEmail     func(to, fullName, endDate string) error
}

func (h *Handler) SetHashPassword(fn func(string) (string, error)) { h.hashPassword = fn }
func (h *Handler) SetSendPasswordResetEmail(fn func(to, token string) error) {
	h.sendPasswordResetEmail = fn
}
func (h *Handler) SetSendContractSignedEmail(fn func(to, name string, pdf []byte, verificationToken string) error) {
	h.sendContractSignedEmail = fn
}
func (h *Handler) SetSendInviteEmail(fn func(to, fullName, registerURL string) error) {
	h.sendInviteEmail = fn
}
func (h *Handler) SetSendPatientInviteEmail(fn func(to, fullName, registerURL string) error) {
	h.sendPatientInviteEmail = fn
}
func (h *Handler) SetSendContractToSignEmail(fn func(to, fullName, signURL string) error) {
	h.sendContractToSignEmail = fn
}
func (h *Handler) SetSendContractCancelledEmail(fn func(to, fullName string) error) {
	h.sendContractCancelledEmail = fn
}
func (h *Handler) SetSendContractEndedEmail(fn func(to, fullName, endDate string) error) {
	h.sendContractEndedEmail = fn
}

// Login autentica PROFESSIONAL ou SUPER_ADMIN em um único endpoint.
// Prioridade: se o e-mail existir como SUPER_ADMIN, autentica como SUPER_ADMIN (não tenta PROFESSIONAL).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}

	// 1) SUPER_ADMIN
	admin, err := repo.SuperAdminByEmail(r.Context(), h.Pool, req.Email)
	if err == nil {
		if !auth.CheckPassword(admin.PasswordHash, req.Password) {
			genericLoginError(w)
			return
		}
		tok, err := auth.BuildJWT(h.Cfg.JWTSecret, admin.ID.String(), auth.RoleSuperAdmin, nil, false, nil, 24*time.Hour)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{
			Token:     tok,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			User: UserInfo{
				ID:       admin.ID.String(),
				Email:    admin.Email,
				FullName: admin.FullName,
				Role:     auth.RoleSuperAdmin,
			},
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		// Mantém resposta genérica por segurança.
		genericLoginError(w)
		return
	}

	// 2) PROFESSIONAL
	prof, err := repo.ProfessionalByEmail(r.Context(), h.Pool, req.Email)
	if err != nil {
		genericLoginError(w)
		return
	}
	if !auth.CheckPassword(prof.PasswordHash, req.Password) {
		genericLoginError(w)
		return
	}
	clinicID := prof.ClinicID.String()
	tok, err := auth.BuildJWT(h.Cfg.JWTSecret, prof.ID.String(), auth.RoleProfessional, &clinicID, false, nil, 24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     tok,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       prof.ID.String(),
			Email:    prof.Email,
			FullName: prof.FullName,
			Role:     auth.RoleProfessional,
			ClinicID: &clinicID,
		},
	})
}

func (h *Handler) LoginProfessional(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}
	prof, err := repo.ProfessionalByEmail(r.Context(), h.Pool, req.Email)
	if err != nil {
		genericLoginError(w)
		return
	}
	if !auth.CheckPassword(prof.PasswordHash, req.Password) {
		genericLoginError(w)
		return
	}
	clinicID := prof.ClinicID.String()
	tok, err := auth.BuildJWT(h.Cfg.JWTSecret, prof.ID.String(), auth.RoleProfessional, &clinicID, false, nil, 24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     tok,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       prof.ID.String(),
			Email:    prof.Email,
			FullName: prof.FullName,
			Role:     auth.RoleProfessional,
			ClinicID: &clinicID,
		},
	})
}

func (h *Handler) LoginSuperAdmin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}
	admin, err := repo.SuperAdminByEmail(r.Context(), h.Pool, req.Email)
	if err != nil {
		genericLoginError(w)
		return
	}
	if !auth.CheckPassword(admin.PasswordHash, req.Password) {
		genericLoginError(w)
		return
	}
	tok, err := auth.BuildJWT(h.Cfg.JWTSecret, admin.ID.String(), auth.RoleSuperAdmin, nil, false, nil, 24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     tok,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       admin.ID.String(),
			Email:    admin.Email,
			FullName: admin.FullName,
			Role:     auth.RoleSuperAdmin,
		},
	})
}

func genericLoginError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"invalid credentials"}`))
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	c := auth.ClaimsFrom(r.Context())
	if c == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UserInfo{
		ID:       c.UserID,
		Role:     c.Role,
		ClinicID: c.ClinicID,
	})
}

// GetMySignature retorna a imagem de assinatura do profissional (apenas role PROFESSIONAL).
func (h *Handler) GetMySignature(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	profID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
		return
	}
	prof, err := repo.ProfessionalByID(r.Context(), h.Pool, profID)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	sig := ""
	if prof.SignatureImageData != nil {
		sig = *prof.SignatureImageData
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"signature_image_data": sig})
}

// PutMySignature atualiza a imagem de assinatura do profissional (apenas role PROFESSIONAL).
func (h *Handler) PutMySignature(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	profID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		SignatureImageData string `json:"signature_image_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	var sig *string
	if req.SignatureImageData != "" {
		// Limite razoável para data URL (ex.: 500KB)
		if len(req.SignatureImageData) > 500*1024 {
			http.Error(w, `{"error":"imagem muito grande"}`, http.StatusBadRequest)
			return
		}
		sig = &req.SignatureImageData
	}
	if err := repo.UpdateProfessionalSignature(r.Context(), h.Pool, profID, sig); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Assinatura atualizada."})
}

// GetMyBranding retorna a aparência (white-label) da clínica do profissional.
func (h *Handler) GetMyBranding(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicIDStr := auth.ClinicIDFrom(r.Context())
	if clinicIDStr == nil || *clinicIDStr == "" {
		http.Error(w, `{"error":"no clinic"}`, http.StatusForbidden)
		return
	}
	clinicID, err := uuid.Parse(*clinicIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	b, err := repo.GetClinicBranding(r.Context(), h.Pool, clinicID)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(map[string]interface{}{
		"primary_color":         b.PrimaryColor,
		"background_color":      b.BackgroundColor,
		"home_label":            b.HomeLabel,
		"home_image_url":        b.HomeImageURL,
		"action_button_color":   b.ActionButtonColor,
		"negation_button_color": b.NegationButtonColor,
	})
}

// PutMyBranding atualiza a aparência (white-label) da clínica do profissional.
func (h *Handler) PutMyBranding(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicIDStr := auth.ClinicIDFrom(r.Context())
	if clinicIDStr == nil || *clinicIDStr == "" {
		http.Error(w, `{"error":"no clinic"}`, http.StatusForbidden)
		return
	}
	clinicID, err := uuid.Parse(*clinicIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		PrimaryColor        *string `json:"primary_color"`
		BackgroundColor     *string `json:"background_color"`
		HomeLabel           *string `json:"home_label"`
		HomeImageURL        *string `json:"home_image_url"`
		ActionButtonColor   *string `json:"action_button_color"`
		NegationButtonColor *string `json:"negation_button_color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	// Limites: cores hex até 20 chars; label até 100; URL até 2000
	if req.PrimaryColor != nil && len(*req.PrimaryColor) > 20 {
		http.Error(w, `{"error":"primary_color too long"}`, http.StatusBadRequest)
		return
	}
	if req.BackgroundColor != nil && len(*req.BackgroundColor) > 20 {
		http.Error(w, `{"error":"background_color too long"}`, http.StatusBadRequest)
		return
	}
	if req.HomeLabel != nil && len(*req.HomeLabel) > 100 {
		http.Error(w, `{"error":"home_label too long"}`, http.StatusBadRequest)
		return
	}
	if req.HomeImageURL != nil && len(*req.HomeImageURL) > 2000 {
		http.Error(w, `{"error":"home_image_url too long"}`, http.StatusBadRequest)
		return
	}
	if req.ActionButtonColor != nil && len(*req.ActionButtonColor) > 20 {
		http.Error(w, `{"error":"action_button_color too long"}`, http.StatusBadRequest)
		return
	}
	if req.NegationButtonColor != nil && len(*req.NegationButtonColor) > 20 {
		http.Error(w, `{"error":"negation_button_color too long"}`, http.StatusBadRequest)
		return
	}
	b := &repo.ClinicBranding{
		PrimaryColor:        req.PrimaryColor,
		BackgroundColor:     req.BackgroundColor,
		HomeLabel:           req.HomeLabel,
		HomeImageURL:        req.HomeImageURL,
		ActionButtonColor:   req.ActionButtonColor,
		NegationButtonColor: req.NegationButtonColor,
	}
	if err := repo.UpdateClinicBranding(r.Context(), h.Pool, clinicID, b); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Aparência atualizada."})
}

// GetMyProfile retorna dados editáveis do perfil do profissional (não inclui CPF).
func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	profID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
		return
	}
	p, err := repo.ProfessionalProfileByID(r.Context(), h.Pool, profID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":             p.ID.String(),
		"email":          p.Email,
		"full_name":      p.FullName,
		"trade_name":     p.TradeName,
		"birth_date":     p.BirthDate,
		"address":        p.Address,
		"marital_status": p.MaritalStatus,
	})
}

// PatchMyProfile atualiza perfil do profissional (trade_name + dados pessoais), sem permitir alterar email/CPF.
// Também sincroniza clinics.name (clinic interna) com trade_name (se informado) ou full_name.
func (h *Handler) PatchMyProfile(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	profID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		FullName      string  `json:"full_name"`
		TradeName     *string `json:"trade_name"`
		BirthDate     *string `json:"birth_date"`
		Address       *string `json:"address"`
		MaritalStatus *string `json:"marital_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		http.Error(w, `{"error":"full_name obrigatório"}`, http.StatusBadRequest)
		return
	}
	var tradeName *string
	if req.TradeName != nil {
		t := strings.TrimSpace(*req.TradeName)
		tradeName = &t
	}
	if err := repo.UpdateProfessionalProfile(r.Context(), h.Pool, profID, fullName, tradeName, req.BirthDate, req.Address, req.MaritalStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	// Sincroniza o nome da clinic interna.
	p, err := repo.ProfessionalProfileByID(r.Context(), h.Pool, profID)
	if err == nil {
		effectiveName := fullName
		if p.TradeName != nil && strings.TrimSpace(*p.TradeName) != "" {
			effectiveName = strings.TrimSpace(*p.TradeName)
		}
		_, _ = h.Pool.Exec(r.Context(), "UPDATE clinics SET name = $1, updated_at = now() WHERE id = $2", effectiveName, p.ClinicID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Perfil atualizado."})
}
