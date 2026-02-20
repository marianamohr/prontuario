package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/auth"
)

type backofficeRelatedPatient struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	BirthDate *string `json:"birth_date,omitempty"`
}

type backofficeRelatedGuardian struct {
	ID            string `json:"id"`
	FullName      string `json:"full_name"`
	Email         string `json:"email"`
	Status        string `json:"status"`
	PatientsCount int    `json:"patients_count"`
}

// BackofficeProfessionalRelatedData retorna pacientes e responsáveis vinculados ao profissional (via clinic_id).
func (h *Handler) BackofficeProfessionalRelatedData(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	idStr := mux.Vars(r)["id"]
	profID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var clinicID uuid.UUID
	if err := h.DB.WithContext(r.Context()).Raw(`SELECT clinic_id FROM professionals WHERE id = ?`, profID).Scan(&clinicID).Error; err != nil || clinicID == uuid.Nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	var patients []backofficeRelatedPatient
	if err := h.DB.WithContext(r.Context()).Raw(`
		SELECT id::text, full_name, birth_date::text
		FROM patients
		WHERE clinic_id = ? AND deleted_at IS NULL
		ORDER BY full_name
	`, clinicID).Scan(&patients).Error; err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	var guardians []backofficeRelatedGuardian
	if err := h.DB.WithContext(r.Context()).Raw(`
		SELECT g.id::text, g.full_name, g.email, g.status, COUNT(DISTINCT pg.patient_id)::int as patients_count
		FROM legal_guardians g
		JOIN patient_guardians pg ON pg.legal_guardian_id = g.id
		JOIN patients p ON p.id = pg.patient_id
		WHERE p.clinic_id = ? AND p.deleted_at IS NULL AND g.deleted_at IS NULL
		GROUP BY g.id, g.full_name, g.email, g.status
		ORDER BY g.full_name
	`, clinicID).Scan(&guardians).Error; err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"professional_id": profID.String(),
		"clinic_id":       clinicID.String(),
		"patients":        patients,
		"guardians":       guardians,
	})
}
