package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/repo"
)

// GetRemarcarByToken returns appointment data and available slots for reschedule (public).
func (h *Handler) GetRemarcarByToken(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	info, err := repo.GetAppointmentByReminderToken(r.Context(), h.Pool, token)
	if err != nil {
		log.Printf("[remarcar] GetAppointmentByReminderToken: %v", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if info == nil {
		http.Error(w, `{"error":"link invalid or expired"}`, http.StatusNotFound)
		return
	}
	// Available slots: next 14 days starting tomorrow
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Now().In(loc)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	endDate := tomorrow.AddDate(0, 0, 14)
	slots, err := repo.ListAvailableSlotsForProfessional(r.Context(), h.Pool, info.ProfessionalID, info.ClinicID, tomorrow, endDate, &info.AppointmentID)
	if err != nil {
		log.Printf("[remarcar] ListAvailableSlotsForProfessional: %v", err)
		slots = nil
	}
	slotsOut := make([]map[string]string, len(slots))
	for i, s := range slots {
		slotsOut[i] = map[string]string{"date": s.Date, "start_time": s.StartTime}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"appointment_id":     info.AppointmentID.String(),
		"patient_name":      info.PatientName,
		"current_date":      info.AppointmentDate.Format("2006-01-02"),
		"current_start_time": info.StartTime.Format("15:04"),
		"status":            info.Status,
		"slots":             slotsOut,
	})
}

// ConfirmRemarcar records attendance confirmation via token (public).
// Only updates to CONFIRMADO when current status is AGENDADO; if already CONFIRMADO returns success; otherwise 400.
func (h *Handler) ConfirmRemarcar(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	info, err := repo.GetAppointmentByReminderToken(r.Context(), h.Pool, token)
	if err != nil || info == nil {
		http.Error(w, `{"error":"link invalid or expired"}`, http.StatusNotFound)
		return
	}
	switch info.Status {
	case "CONFIRMADO":
		// Idempotent: already confirmed
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Attendance confirmed."})
		return
	case "AGENDADO":
		// Update to CONFIRMADO
		statusConfirmed := "CONFIRMADO"
		if err := repo.UpdateAppointment(r.Context(), h.Pool, info.AppointmentID, info.ClinicID, nil, nil, nil, &statusConfirmed, nil); err != nil {
			log.Printf("[confirm-remarcar] UpdateAppointment: %v", err)
			http.Error(w, `{"error":"failed to confirm"}`, http.StatusInternalServerError)
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
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Attendance confirmed."})
		return
	default:
		http.Error(w, `{"error":"attendance can only be confirmed when appointment is scheduled"}`, http.StatusBadRequest)
		return
	}
}

// RemarcarAppointment updates appointment date/time via token (public).
func (h *Handler) RemarcarAppointment(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	info, err := repo.GetAppointmentByReminderToken(r.Context(), h.Pool, token)
	if err != nil || info == nil {
		http.Error(w, `{"error":"link invalid or expired"}`, http.StatusNotFound)
		return
	}
	var req struct {
		AppointmentDate string `json:"appointment_date"`
		StartTime       string `json:"start_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.AppointmentDate == "" || req.StartTime == "" {
		http.Error(w, `{"error":"appointment_date and start_time are required"}`, http.StatusBadRequest)
		return
	}
	appointmentDate, err := time.Parse("2006-01-02", req.AppointmentDate)
	if err != nil {
		http.Error(w, `{"error":"invalid date"}`, http.StatusBadRequest)
		return
	}
	startTime, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		http.Error(w, `{"error":"invalid time"}`, http.StatusBadRequest)
		return
	}
	endTime := startTime.Add(50 * time.Minute)
	statusAgendado := "AGENDADO"
	if err := repo.UpdateAppointment(r.Context(), h.Pool, info.AppointmentID, info.ClinicID, &appointmentDate, &startTime, &endTime, &statusAgendado, nil); err != nil {
		log.Printf("[remarcar] UpdateAppointment: %v", err)
		http.Error(w, `{"error":"failed to update"}`, http.StatusInternalServerError)
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
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Appointment rescheduled successfully."})
}
