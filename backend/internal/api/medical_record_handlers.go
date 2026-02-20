package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/repo"
)

func (h *Handler) ensureClinicID(r *http.Request) (*uuid.UUID, bool) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		return nil, false
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		return nil, false
	}
	return &cid, true
}

func (h *Handler) canAccessPatientAsProfessional(r *http.Request, patientID uuid.UUID) bool {
	role := auth.RoleFrom(r.Context())
	if role == auth.RoleSuperAdmin {
		return true
	}
	cid, ok := h.ensureClinicID(r)
	if !ok {
		return false
	}
	if role != auth.RoleProfessional {
		return false
	}
	_, err := repo.PatientByIDAndClinic(r.Context(), h.DB, patientID, *cid)
	return err == nil
}

func (h *Handler) canViewMedicalRecordAsGuardian(r *http.Request, patientID uuid.UUID) bool {
	guardianID := auth.UserIDFrom(r.Context())
	if guardianID == "" {
		return false
	}
	gID, errG := uuid.Parse(guardianID)
	if errG != nil {
		return false
	}
	can, err := repo.GuardianCanViewMedicalRecord(r.Context(), h.DB, gID, patientID)
	return err == nil && can
}

func (h *Handler) canAccessMedicalRecord(r *http.Request, patientID uuid.UUID) bool {
	if h.canAccessPatientAsProfessional(r, patientID) {
		return true
	}
	if auth.RoleFrom(r.Context()) == auth.RoleLegalGuardian {
		return h.canViewMedicalRecordAsGuardian(r, patientID)
	}
	return false
}

func (h *Handler) logAccess(r *http.Request, clinicID *uuid.UUID, actorType string, actorID uuid.UUID, action, resourceType string, resourceID, patientID *uuid.UUID) {
	_ = repo.CreateAccessLog(r.Context(), h.DB, clinicID, &actorID, actorType, action, resourceType, resourceID, patientID, r.RemoteAddr, r.UserAgent(), r.Header.Get("X-Request-ID"))
}

func (h *Handler) ListRecordEntries(w http.ResponseWriter, r *http.Request) {
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessMedicalRecord(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	mrID, err := repo.GetOrCreateMedicalRecord(r.Context(), h.DB, patientID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	limit, offset := ParseLimitOffset(r)
	entries, total, err := repo.RecordEntriesByMedicalRecordPaginated(r.Context(), h.DB, mrID, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	keysMap, errKeys := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
	_ = errKeys
	type item struct {
		ID         string `json:"id"`
		Content    string `json:"content"`
		EntryDate  string `json:"entry_date"`
		AuthorID   string `json:"author_id"`
		AuthorType string `json:"author_type"`
		CreatedAt  string `json:"created_at"`
	}
	out := make([]item, 0, len(entries))
	for _, e := range entries {
		plain, errDec := crypto.Decrypt(e.ContentEncrypted, e.ContentNonce, e.ContentKeyVersion, keysMap)
		_ = errDec
		content := ""
		if len(plain) > 0 {
			content = string(plain)
		}
		out = append(out, item{
			ID: e.ID.String(), Content: content, EntryDate: e.EntryDate.Format("2006-01-02"),
			AuthorID: e.AuthorID.String(), AuthorType: e.AuthorType, CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	actorID := auth.UserIDFrom(r.Context())
	if aid, e := uuid.Parse(actorID); e == nil {
		var cid *uuid.UUID
		if auth.ClinicIDFrom(r.Context()) != nil {
			if u, e := uuid.Parse(*auth.ClinicIDFrom(r.Context())); e == nil {
				cid = &u
			}
		}
		h.logAccess(r, cid, auth.RoleFrom(r.Context()), aid, "READ", "MEDICAL_RECORD", &mrID, &patientID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": out,
		"limit":   limit,
		"offset":  offset,
		"total":   total,
	})
}

func (h *Handler) CreateRecordEntry(w http.ResponseWriter, r *http.Request) {
	patientIDStr := mux.Vars(r)["patientId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessMedicalRecord(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	role := auth.RoleFrom(r.Context())
	if role != auth.RoleProfessional && role != auth.RoleSuperAdmin {
		http.Error(w, `{"error":"only professional can create entries"}`, http.StatusForbidden)
		return
	}
	var req struct {
		Content   string `json:"content"`
		EntryDate string `json:"entry_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.EntryDate == "" {
		req.EntryDate = time.Now().Format("2006-01-02")
	}
	entryDate, errParse := time.Parse("2006-01-02", req.EntryDate)
	_ = errParse
	keysMap, err := crypto.ParseKeysEnv(h.Cfg.DataEncryptionKeys)
	if err != nil {
		http.Error(w, `{"error":"config"}`, http.StatusInternalServerError)
		return
	}
	keyVer := h.Cfg.CurrentDataKeyVer
	if keyVer == "" {
		keyVer = "v1"
	}
	enc, nonce, err := crypto.Encrypt([]byte(req.Content), keyVer, keysMap)
	if err != nil {
		http.Error(w, `{"error":"encryption"}`, http.StatusInternalServerError)
		return
	}
	mrID, err := repo.GetOrCreateMedicalRecord(r.Context(), h.DB, patientID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	authorID, errAuth := uuid.Parse(auth.UserIDFrom(r.Context()))
	_ = errAuth
	id, err := repo.CreateRecordEntry(r.Context(), h.DB, mrID, enc, nonce, keyVer, entryDate, authorID, role)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	var cid *uuid.UUID
	if auth.ClinicIDFrom(r.Context()) != nil {
		if u, e := uuid.Parse(*auth.ClinicIDFrom(r.Context())); e == nil {
			cid = &u
		}
	}
	aid, errAid := uuid.Parse(auth.UserIDFrom(r.Context()))
	_ = errAid
	h.logAccess(r, cid, role, aid, "READ", "RECORD_ENTRY", &id, &patientID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
}
