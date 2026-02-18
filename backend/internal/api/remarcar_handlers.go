package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/repo"
)

// GetRemarcarByToken retorna dados do compromisso e slots disponíveis para remarcação (público).
func (h *Handler) GetRemarcarByToken(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token obrigatório"}`, http.StatusBadRequest)
		return
	}
	info, err := repo.GetAppointmentByReminderToken(r.Context(), h.Pool, token)
	if err != nil {
		log.Printf("[remarcar] GetAppointmentByReminderToken: %v", err)
		http.Error(w, `{"error":"erro interno"}`, http.StatusInternalServerError)
		return
	}
	if info == nil {
		http.Error(w, `{"error":"link inválido ou expirado"}`, http.StatusNotFound)
		return
	}
	// Slots disponíveis: próximos 14 dias a partir de amanhã
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Now().In(loc)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	endDate := tomorrow.AddDate(0, 0, 14)
	slots, err := repo.ListAvailableSlotsForProfessional(r.Context(), h.Pool, info.ProfessionalID, info.ClinicID, tomorrow, endDate, &info.AppointmentID)
	if err != nil {
		log.Printf("[remarcar] ListAvailableSlots: %v", err)
		slots = nil
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"appointment_id":    info.AppointmentID.String(),
		"patient_name":      info.PatientName,
		"current_date":      info.AppointmentDate.Format("2006-01-02"),
		"current_start_time": info.StartTime.Format("15:04"),
		"slots":             slots,
	})
}

// ConfirmRemarcar registra confirmação de presença via token (público).
func (h *Handler) ConfirmRemarcar(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token obrigatório"}`, http.StatusBadRequest)
		return
	}
	info, err := repo.GetAppointmentByReminderToken(r.Context(), h.Pool, token)
	if err != nil || info == nil {
		http.Error(w, `{"error":"link inválido ou expirado"}`, http.StatusNotFound)
		return
	}
	_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
		Action:       "APPOINTMENT_ATTENDANCE_CONFIRMED",
		ActorType:    "LEGAL_GUARDIAN",
		ActorID:      &info.GuardianID,
		ResourceType: strPtr("APPOINTMENT"),
		ResourceID:   &info.AppointmentID,
		PatientID:    &info.PatientID,
		Source:       strPtr("USER"),
		Metadata:     map[string]string{"guardian_id": info.GuardianID.String(), "via": "reminder_link"},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Presença confirmada."})
}

// RemarcarAppointment altera data/hora do compromisso via token (público).
func (h *Handler) RemarcarAppointment(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token obrigatório"}`, http.StatusBadRequest)
		return
	}
	info, err := repo.GetAppointmentByReminderToken(r.Context(), h.Pool, token)
	if err != nil || info == nil {
		http.Error(w, `{"error":"link inválido ou expirado"}`, http.StatusNotFound)
		return
	}
	var req struct {
		AppointmentDate string `json:"appointment_date"`
		StartTime       string `json:"start_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"corpo inválido"}`, http.StatusBadRequest)
		return
	}
	if req.AppointmentDate == "" || req.StartTime == "" {
		http.Error(w, `{"error":"appointment_date e start_time são obrigatórios"}`, http.StatusBadRequest)
		return
	}
	appointmentDate, err := time.Parse("2006-01-02", req.AppointmentDate)
	if err != nil {
		http.Error(w, `{"error":"data inválida"}`, http.StatusBadRequest)
		return
	}
	startTime, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		http.Error(w, `{"error":"horário inválido"}`, http.StatusBadRequest)
		return
	}
	endTime := startTime.Add(50 * time.Minute)
	if err := repo.UpdateAppointment(r.Context(), h.Pool, info.AppointmentID, info.ClinicID, &appointmentDate, &startTime, &endTime, nil, nil); err != nil {
		log.Printf("[remarcar] UpdateAppointment: %v", err)
		http.Error(w, `{"error":"falha ao atualizar"}`, http.StatusInternalServerError)
		return
	}
	_ = repo.CreateAuditEventFull(r.Context(), h.Pool, repo.AuditEvent{
		Action:       "APPOINTMENT_REMARCARED",
		ActorType:    "LEGAL_GUARDIAN",
		ActorID:      &info.GuardianID,
		ResourceType: strPtr("APPOINTMENT"),
		ResourceID:   &info.AppointmentID,
		PatientID:    &info.PatientID,
		Source:       strPtr("USER"),
		Metadata: map[string]string{
			"guardian_id":      info.GuardianID.String(),
			"via":              "reminder_link",
			"new_date":         req.AppointmentDate,
			"new_start_time":   req.StartTime,
		},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Consulta remarcada com sucesso."})
}
