package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
)

type ImpersonateStartRequest struct {
	TargetUserType string `json:"target_user_type"`
	TargetUserID   string `json:"target_user_id"`
	Reason         string `json:"reason"`
}

type ImpersonateStartResponse struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	ExpiresIn int    `json:"expires_in_seconds"`
}

func (h *Handler) ListClinics(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	rows, err := h.Pool.Query(r.Context(), "SELECT id, name FROM clinics ORDER BY name")
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type clinicRow struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var list []clinicRow
	for rows.Next() {
		var c clinicRow
		var id uuid.UUID
		if err := rows.Scan(&id, &c.Name); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		c.ID = id.String()
		list = append(list, c)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"clinics": list})
}

func (h *Handler) ListUsersBackoffice(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicID := r.URL.Query().Get("clinic_id")
	limit, offset := ParseLimitOffset(r)
	type userRow struct {
		Type     string `json:"type"`
		ID       string `json:"id"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		ClinicID string `json:"clinic_id,omitempty"`
		Status   string `json:"status"`
	}
	var list []userRow
	var total int
	if clinicID == "" {
		var c1, c2 int
		if err := h.Pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM professionals").Scan(&c1); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		if err := h.Pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM legal_guardians WHERE deleted_at IS NULL").Scan(&c2); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		total = c1 + c2
		q := `
			SELECT type, id, email, full_name, clinic_id, status FROM (
				SELECT 'PROFESSIONAL' as type, id, email, full_name, clinic_id::text, status FROM professionals
				UNION ALL
				SELECT 'LEGAL_GUARDIAN', id, email, full_name, NULL::text, status FROM legal_guardians WHERE deleted_at IS NULL
			) u ORDER BY type, email LIMIT $1 OFFSET $2
		`
		rows, err := h.Pool.Query(r.Context(), q, limit, offset)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var u userRow
			var cid *string
			if err := rows.Scan(&u.Type, &u.ID, &u.Email, &u.FullName, &cid, &u.Status); err != nil {
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
			if cid != nil {
				u.ClinicID = *cid
			}
			list = append(list, u)
		}
	} else {
		cid, err := uuid.Parse(clinicID)
		if err != nil {
			http.Error(w, `{"error":"invalid clinic_id"}`, http.StatusBadRequest)
			return
		}
		if err := h.Pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM professionals WHERE clinic_id = $1", cid).Scan(&total); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		rows, err := h.Pool.Query(r.Context(), "SELECT id, email, full_name, status FROM professionals WHERE clinic_id = $1 ORDER BY email LIMIT $2 OFFSET $3", cid, limit, offset)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var u userRow
			u.Type = "PROFESSIONAL"
			u.ClinicID = clinicID
			if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Status); err != nil {
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
			list = append(list, u)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"users":  list,
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

type BackofficeAddressResponse struct {
	Street       *string `json:"street,omitempty"`
	Number       *string `json:"number,omitempty"`
	Complement   *string `json:"complement,omitempty"`
	Neighborhood *string `json:"neighborhood,omitempty"`
	City         *string `json:"city,omitempty"`
	State        *string `json:"state,omitempty"`
	Country      *string `json:"country,omitempty"`
	Zip          *string `json:"zip,omitempty"`
}

func repoAddressToBackoffice(a *repo.Address) *BackofficeAddressResponse {
	if a == nil {
		return nil
	}
	return &BackofficeAddressResponse{
		Street: a.Street, Number: a.Number, Complement: a.Complement,
		Neighborhood: a.Neighborhood, City: a.City, State: a.State,
		Country: a.Country, Zip: a.Zip,
	}
}

type BackofficeUserDetailResponse struct {
	Type          string                    `json:"type"`
	ID            string                    `json:"id"`
	Email         string                    `json:"email"`
	FullName      string                    `json:"full_name"`
	TradeName     *string                   `json:"trade_name,omitempty"`
	Status        string                    `json:"status"`
	ClinicID      *string                   `json:"clinic_id,omitempty"`
	BirthDate     *string                   `json:"birth_date,omitempty"`
	Address       *BackofficeAddressResponse `json:"address,omitempty"`
	Phone         *string                   `json:"phone,omitempty"`
	MaritalStatus *string                   `json:"marital_status,omitempty"`
	CPF           *string                   `json:"cpf,omitempty"`
	AuthProvider  *string                   `json:"auth_provider,omitempty"`
	HasGoogleSub  *bool                     `json:"has_google_sub,omitempty"`
}

func (h *Handler) GetBackofficeUser(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	userType := strings.ToUpper(strings.TrimSpace(vars["type"]))
	idStr := vars["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	var resp BackofficeUserDetailResponse
	switch userType {
	case "PROFESSIONAL":
		p, err := repo.ProfessionalAdminByID(r.Context(), h.Pool, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		// Descriptografa CPF (apenas para backoffice/super admin).
		var cpfStr *string
		if p.CPFKeyVersion != nil && *p.CPFKeyVersion != "" && len(p.CPFEncrypted) > 0 && len(p.CPFNonce) > 0 {
			keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
			if err == nil {
				dec, err := crypto.Decrypt(p.CPFEncrypted, p.CPFNonce, *p.CPFKeyVersion, keysMap)
				if err == nil && len(dec) > 0 {
					s := string(dec)
					cpfStr = &s
				}
			}
		}
		cid := p.ClinicID.String()
		var addrResp *BackofficeAddressResponse
		if p.AddressID != nil {
			if addr, err := repo.GetAddressByID(r.Context(), h.Pool, *p.AddressID); err == nil {
				addrResp = repoAddressToBackoffice(addr)
			}
		}
		resp = BackofficeUserDetailResponse{
			Type:          "PROFESSIONAL",
			ID:            p.ID.String(),
			Email:         p.Email,
			FullName:      p.FullName,
			TradeName:     p.TradeName,
			Status:        p.Status,
			ClinicID:      &cid,
			BirthDate:     p.BirthDate,
			CPF:           cpfStr,
			Address:       addrResp,
			MaritalStatus: p.MaritalStatus,
		}
	case "LEGAL_GUARDIAN":
		g, err := repo.LegalGuardianByID(r.Context(), h.Pool, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		// Descriptografa CPF (apenas para backoffice/super admin).
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
		hasGoogle := g.GoogleSub != nil && strings.TrimSpace(*g.GoogleSub) != ""
		var addrResp *BackofficeAddressResponse
		if g.AddressID != nil {
			if addr, err := repo.GetAddressByID(r.Context(), h.Pool, *g.AddressID); err == nil {
				addrResp = repoAddressToBackoffice(addr)
			}
		}
		resp = BackofficeUserDetailResponse{
			Type:         "LEGAL_GUARDIAN",
			ID:           g.ID.String(),
			Email:        g.Email,
			FullName:     g.FullName,
			Status:       g.Status,
			BirthDate:    g.BirthDate,
			Address:      addrResp,
			Phone:        g.Phone,
			CPF:          cpfStr,
			AuthProvider: &g.AuthProvider,
			HasGoogleSub: &hasGoogle,
		}
	default:
		http.Error(w, `{"error":"invalid type"}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"user": resp})
}

type PatchBackofficeUserRequest struct {
	Email         *string                   `json:"email"`
	FullName      *string                   `json:"full_name"`
	TradeName     *string                   `json:"trade_name"`
	Status        *string                   `json:"status"`
	ClinicID      *string                   `json:"clinic_id"`
	BirthDate     *string                   `json:"birth_date"`
	Address       *BackofficeAddressResponse `json:"address"` // 8 campos
	Phone         *string                   `json:"phone"`
	MaritalStatus *string                   `json:"marital_status"`
	CPF           *string                   `json:"cpf"`
	NewPassword   *string                   `json:"new_password"`
}

func (h *Handler) PatchBackofficeUser(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	userType := strings.ToUpper(strings.TrimSpace(vars["type"]))
	idStr := vars["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	var req PatchBackofficeUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Email != nil {
		e := strings.ToLower(strings.TrimSpace(*req.Email))
		*req.Email = e
		if e == "" {
			http.Error(w, `{"error":"email inválido"}`, http.StatusBadRequest)
			return
		}
		if !emailRegex.MatchString(e) {
			http.Error(w, `{"error":"email inválido"}`, http.StatusBadRequest)
			return
		}
	}
	if req.FullName != nil && strings.TrimSpace(*req.FullName) == "" {
		http.Error(w, `{"error":"full_name inválido"}`, http.StatusBadRequest)
		return
	}
	if req.Status != nil {
		s := strings.TrimSpace(*req.Status)
		if s != "ACTIVE" && s != "SUSPENDED" && s != "CANCELLED" {
			http.Error(w, `{"error":"status inválido"}`, http.StatusBadRequest)
			return
		}
		*req.Status = s
	}
	var passwordHash *string
	if req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != "" {
		if len(strings.TrimSpace(*req.NewPassword)) < 8 {
			http.Error(w, `{"error":"new_password deve ter no mínimo 8 caracteres"}`, http.StatusBadRequest)
			return
		}
		if h.hashPassword == nil {
			http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
			return
		}
		hash, err := h.hashPassword(strings.TrimSpace(*req.NewPassword))
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		passwordHash = &hash
	}

	switch userType {
	case "PROFESSIONAL":
		var clinicUUID *uuid.UUID
		if req.ClinicID != nil {
			if strings.TrimSpace(*req.ClinicID) == "" {
				http.Error(w, `{"error":"clinic_id inválido"}`, http.StatusBadRequest)
				return
			}
			cid, err := uuid.Parse(strings.TrimSpace(*req.ClinicID))
			if err != nil {
				http.Error(w, `{"error":"clinic_id inválido"}`, http.StatusBadRequest)
				return
			}
			clinicUUID = &cid
		}
		var cpfEnc []byte
		var cpfNonce []byte
		var cpfKeyVer *string
		var cpfHash *string
		if req.CPF != nil && strings.TrimSpace(*req.CPF) != "" {
			cpfNorm := crypto.NormalizeCPF(*req.CPF)
			if len(cpfNorm) != 11 {
				http.Error(w, `{"error":"cpf inválido"}`, http.StatusBadRequest)
				return
			}
			hh := crypto.CPFHash(cpfNorm)
			cpfHash = &hh
			keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
			if err != nil {
				http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
				return
			}
			keyVer := h.Cfg.CurrentDataKeyVer
			if keyVer == "" {
				keyVer = "v1"
			}
			enc, nonce, err := crypto.Encrypt([]byte(cpfNorm), keyVer, keysMap)
			if err != nil {
				http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
				return
			}
			cpfEnc = enc
			cpfNonce = nonce
			cpfKeyVer = &keyVer
		}
		var addressID *uuid.UUID
		if req.Address != nil {
			addrInput := &AddressInput{
				Street:       strFromPtr(req.Address.Street),
				Number:       strFromPtr(req.Address.Number),
				Complement:   strFromPtr(req.Address.Complement),
				Neighborhood: strFromPtr(req.Address.Neighborhood),
				City:         strFromPtr(req.Address.City),
				State:        strFromPtr(req.Address.State),
				Country:      strFromPtr(req.Address.Country),
				Zip:          strFromPtr(req.Address.Zip),
			}
			if err := ValidateAddress(addrInput); err != nil {
				http.Error(w, `{"error":"endereço inválido (CEP 8 dígitos; rua, bairro, cidade, estado, país obrigatórios)"}`, http.StatusBadRequest)
				return
			}
			addr := AddressInputToRepo(addrInput)
			id, err := repo.CreateAddress(r.Context(), h.Pool, addr)
			if err != nil {
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
			addressID = &id
		}
		if err := repo.UpdateProfessionalAdmin(
			r.Context(),
			h.Pool,
			id,
			req.Email,
			req.FullName,
			req.TradeName,
			clinicUUID,
			req.Status,
			req.BirthDate,
			addressID,
			req.MaritalStatus,
			cpfHash,
			cpfEnc,
			cpfNonce,
			cpfKeyVer,
			passwordHash,
			nil,
		); err != nil {
			if isUniqueViolation(err) {
				http.Error(w, `{"error":"e-mail já está em uso"}`, http.StatusConflict)
				return
			}
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				log.Printf("[backoffice] patch user PROFESSIONAL falhou: id=%s code=%s msg=%s detail=%s", id.String(), pgErr.Code, pgErr.Message, pgErr.Detail)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal", "detail": pgErr.Message})
				return
			}
			log.Printf("[backoffice] patch user PROFESSIONAL falhou: id=%s err=%v", id.String(), err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
	case "LEGAL_GUARDIAN":
		_, err := repo.LegalGuardianByID(r.Context(), h.Pool, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		var authProvider *string
		if passwordHash != nil {
			ap := "LOCAL"
			authProvider = &ap
		}
		var cpfEnc []byte
		var cpfNonce []byte
		var cpfKeyVer *string
		var cpfHash *string
		if req.CPF != nil && strings.TrimSpace(*req.CPF) != "" {
			cpfNorm := crypto.NormalizeCPF(*req.CPF)
			if len(cpfNorm) != 11 {
				http.Error(w, `{"error":"cpf inválido"}`, http.StatusBadRequest)
				return
			}
			hh := crypto.CPFHash(cpfNorm)
			cpfHash = &hh
			keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
			if err != nil {
				http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
				return
			}
			keyVer := h.Cfg.CurrentDataKeyVer
			if keyVer == "" {
				keyVer = "v1"
			}
			enc, nonce, err := crypto.Encrypt([]byte(cpfNorm), keyVer, keysMap)
			if err != nil {
				http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
				return
			}
			cpfEnc = enc
			cpfNonce = nonce
			cpfKeyVer = &keyVer
		}
		var lgAddressID *uuid.UUID
		if req.Address != nil {
			addrInput := &AddressInput{
				Street:       strFromPtr(req.Address.Street),
				Number:       strFromPtr(req.Address.Number),
				Complement:   strFromPtr(req.Address.Complement),
				Neighborhood: strFromPtr(req.Address.Neighborhood),
				City:         strFromPtr(req.Address.City),
				State:        strFromPtr(req.Address.State),
				Country:      strFromPtr(req.Address.Country),
				Zip:          strFromPtr(req.Address.Zip),
			}
			if err := ValidateAddress(addrInput); err != nil {
				http.Error(w, `{"error":"endereço inválido (CEP 8 dígitos; rua, bairro, cidade, estado, país obrigatórios)"}`, http.StatusBadRequest)
				return
			}
			addr := AddressInputToRepo(addrInput)
			aid, err := repo.CreateAddress(r.Context(), h.Pool, addr)
			if err != nil {
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
			lgAddressID = &aid
		}
		if err := repo.UpdateLegalGuardianAdmin(
			r.Context(),
			h.Pool,
			id,
			req.FullName,
			req.Email,
			lgAddressID,
			req.BirthDate,
			req.Phone,
			req.Status,
			passwordHash,
			authProvider,
			cpfEnc,
			cpfNonce,
			cpfKeyVer,
			cpfHash,
		); err != nil {
			if isUniqueViolation(err) {
				http.Error(w, `{"error":"e-mail ou cpf já está em uso"}`, http.StatusConflict)
				return
			}
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				log.Printf("[backoffice] patch user LEGAL_GUARDIAN falhou: id=%s code=%s msg=%s detail=%s", id.String(), pgErr.Code, pgErr.Message, pgErr.Detail)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal", "detail": pgErr.Message})
				return
			}
			log.Printf("[backoffice] patch user LEGAL_GUARDIAN falhou: id=%s err=%v", id.String(), err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, `{"error":"invalid type"}`, http.StatusBadRequest)
		return
	}
	// Retorna o usuário atualizado
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Usuário atualizado."})
}

// CleanupOrphanAddresses remove endereços não referenciados (apenas SUPER_ADMIN).
func (h *Handler) CleanupOrphanAddresses(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	deleted, err := repo.CleanupOrphanAddresses(r.Context(), h.Pool)
	if err != nil {
		log.Printf("[backoffice] cleanup orphan addresses: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"deleted": deleted, "message": "Endereços órfãos removidos."})
}

// TriggerReminder dispara os lembretes de consulta via serviço reminder (proxy). Aceita professional_id opcional.
func (h *Handler) TriggerReminder(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if h.Cfg.ReminderServiceURL == "" {
		http.Error(w, `{"error":"REMINDER_SERVICE_URL não configurado"}`, http.StatusServiceUnavailable)
		return
	}
	url := strings.TrimSuffix(h.Cfg.ReminderServiceURL, "/") + "/trigger"
	if idStr := r.URL.Query().Get("professional_id"); idStr != "" {
		url += "?professional_id=" + idStr
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if h.Cfg.ReminderAPIKey != "" {
		req.Header.Set("X-API-Key", h.Cfg.ReminderAPIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[backoffice] reminder trigger: %v", err)
		http.Error(w, `{"error":"falha ao chamar serviço reminder"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (h *Handler) ImpersonateStart(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req ImpersonateStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.TargetUserType == "" || req.TargetUserID == "" || req.Reason == "" {
		http.Error(w, `{"error":"target_user_type, target_user_id and reason required"}`, http.StatusBadRequest)
		return
	}
	if req.TargetUserType != "PROFESSIONAL" && req.TargetUserType != "LEGAL_GUARDIAN" {
		http.Error(w, `{"error":"invalid target_user_type"}`, http.StatusBadRequest)
		return
	}
	targetID, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		http.Error(w, `{"error":"invalid target_user_id"}`, http.StatusBadRequest)
		return
	}
	adminID := auth.UserIDFrom(r.Context())
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		http.Error(w, `{"error":"invalid admin"}`, http.StatusInternalServerError)
		return
	}
	var clinicID *uuid.UUID
	if req.TargetUserType == "PROFESSIONAL" {
		var cid uuid.UUID
		if err := h.Pool.QueryRow(r.Context(), "SELECT clinic_id FROM professionals WHERE id = $1", targetID).Scan(&cid); err == nil {
			clinicID = &cid
		}
	}
	sessionID, err := repo.StartImpersonation(r.Context(), h.Pool, adminUUID, req.TargetUserType, targetID, clinicID, req.Reason)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	sessionIDStr := sessionID.String()
	exp := repo.ImpersonationTTL
	tok, err := auth.BuildJWT(h.Cfg.JWTSecret, req.TargetUserID, req.TargetUserType, ptrString(clinicID.String()), true, &sessionIDStr, exp)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	_ = repo.CreateAuditEvent(r.Context(), h.Pool, "IMPERSONATION_START", "SUPER_ADMIN", &adminUUID, map[string]string{
		"target_user_type": req.TargetUserType, "target_user_id": req.TargetUserID, "reason": req.Reason, "session_id": sessionID.String()})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ImpersonateStartResponse{Token: tok, SessionID: sessionID.String(), ExpiresIn: int(exp.Seconds())})
}

func ptrString(s string) *string { return &s }

func (h *Handler) ImpersonateEnd(w http.ResponseWriter, r *http.Request) {
	c := auth.ClaimsFrom(r.Context())
	if c == nil || !c.IsImpersonated || c.ImpersonationSessionID == nil {
		http.Error(w, `{"error":"not in impersonation"}`, http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(*c.ImpersonationSessionID)
	if err != nil {
		http.Error(w, `{"error":"invalid session"}`, http.StatusBadRequest)
		return
	}
	if err := repo.EndImpersonation(r.Context(), h.Pool, sessionID); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Auditoria: fim de impersonate (ação do super admin, embora o token atual esteja como impersonated).
	// Sem PII; registra apenas IDs e request_id.
	_ = repo.CreateAuditEvent(r.Context(), h.Pool, "IMPERSONATION_END", "SUPER_ADMIN", nil, map[string]string{
		"session_id": sessionID.String(),
		"request_id": r.Header.Get("X-Request-ID"),
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"message":"Impersonation ended."}`))
}
