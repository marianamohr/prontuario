package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/repo"
)

const debugLogPath = "/Users/mariana.mohr/Documents/workspace/prontuario/.cursor/debug.log"

func debugLog(location, message string, data map[string]interface{}) {
	line, _ := json.Marshal(map[string]interface{}{
		"location": location, "message": message, "data": data,
		"timestamp": time.Now().UnixMilli(), "hypothesisId": "SC1",
	})
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

// GetScheduleConfig returns the schedule config for the clinic (all 7 weekdays).
// clinic_id is taken from the JWT claims (set at professional login or when super-admin impersonates).
func (h *Handler) GetScheduleConfig(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	clinicIDStr := auth.ClinicIDFrom(r.Context())
	fmt.Println("clinicIDStr", clinicIDStr)
	if clinicIDStr == nil || *clinicIDStr == "" {
		http.Error(w, `{"error":"no clinic"}`, http.StatusForbidden)
		return
	}
	clinicID, err := uuid.Parse(*clinicIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
		return
	}
	if h.Cache != nil {
		if cached := h.Cache.Get("schedule:" + clinicID.String()); cached != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(cached)
			return
		}
	}
	list, err := repo.ListScheduleConfig(r.Context(), h.DB, clinicID)
	fmt.Printf("list: %+v\n", list)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if len(list) == 0 {
		log.Printf("[schedule] GET schedule-config: 0 rows for clinic %s (DB empty for this clinic)", clinicID.String())
	}
	// Build 7 days (0-6); missing days get default (enabled=false)
	out := make([]map[string]interface{}, 7)
	formatTimeStr := func(p *string) interface{} {
		if p == nil {
			return nil
		}
		return *p
	}
	for i := 0; i < 7; i++ {
		out[i] = map[string]interface{}{
			"day_of_week":                   i,
			"enabled":                       false,
			"start_time":                    nil,
			"end_time":                      nil,
			"consultation_duration_minutes": 50,
			"interval_minutes":              10,
			"lunch_start":                   nil,
			"lunch_end":                     nil,
		}
	}
	for _, s := range list {
		if s.DayOfWeek >= 0 && s.DayOfWeek < 7 {
			out[s.DayOfWeek] = map[string]interface{}{
				"day_of_week":                   s.DayOfWeek,
				"enabled":                       s.Enabled,
				"start_time":                    formatTimeStr(s.StartTime),
				"end_time":                      formatTimeStr(s.EndTime),
				"consultation_duration_minutes": s.ConsultationDurationMinutes,
				"interval_minutes":              s.IntervalMinutes,
				"lunch_start":                   formatTimeStr(s.LunchStart),
				"lunch_end":                     formatTimeStr(s.LunchEnd),
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	payload := map[string]interface{}{"days": out}
	buf, _ := json.Marshal(payload)
	// Only cache when we have persisted data; avoid caching "7 empty days" so a later GET after save always hits DB
	if h.Cache != nil && len(list) > 0 {
		h.Cache.Set("schedule:"+clinicID.String(), buf)
	}
	_, _ = w.Write(buf)
}

// GetAvailableSlots retorna slots disponíveis para o profissional logado no intervalo [from, to].
// Respeita a configuração da agenda e exclui horários já ocupados.
func (h *Handler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
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
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"no user"}`, http.StatusForbidden)
		return
	}
	professionalID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusBadRequest)
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		http.Error(w, `{"error":"from and to query params required (YYYY-MM-DD)"}`, http.StatusBadRequest)
		return
	}
	from, err1 := time.Parse("2006-01-02", fromStr)
	to, err2 := time.Parse("2006-01-02", toStr)
	if err1 != nil || err2 != nil {
		http.Error(w, `{"error":"from and to must be YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}
	if to.Before(from) {
		http.Error(w, `{"error":"to must be >= from"}`, http.StatusBadRequest)
		return
	}
	slots, err := repo.ListAvailableSlotsForProfessional(r.Context(), h.DB, professionalID, clinicID, from, to, nil)
	if err != nil {
		log.Printf("[available-slots] ListAvailableSlotsForProfessional: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	out := make([]map[string]string, len(slots))
	for i, s := range slots {
		out[i] = map[string]string{"date": s.Date, "start_time": s.StartTime}
	}
	configs, _ := repo.ListScheduleConfig(r.Context(), h.DB, clinicID)
	var configuredDays []int
	for _, c := range configs {
		if c.Enabled && c.StartTime != nil && c.EndTime != nil && *c.StartTime != "" && *c.EndTime != "" {
			configuredDays = append(configuredDays, c.DayOfWeek)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"slots": out, "configured_days": configuredDays})
}

// PutScheduleConfig atualiza a configuração de um ou mais dias.
func (h *Handler) PutScheduleConfig(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
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
	if h.Cache != nil {
		h.Cache.Delete("schedule:" + clinicID.String())
	}
	var req struct {
		Days []struct {
			DayOfWeek                   int     `json:"day_of_week"`
			Enabled                     *bool   `json:"enabled"`
			StartTime                   *string `json:"start_time"`
			EndTime                     *string `json:"end_time"`
			ConsultationDurationMinutes *int    `json:"consultation_duration_minutes"`
			IntervalMinutes             *int    `json:"interval_minutes"`
			LunchStart                  *string `json:"lunch_start"`
			LunchEnd                    *string `json:"lunch_end"`
		} `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if len(req.Days) == 0 {
		http.Error(w, `{"error":"days required (at least one day)"}`, http.StatusBadRequest)
		return
	}
	// Substitui a configuração da clínica: remove todos os dias e grava só os que vieram no body (apenas dias habilitados).
	if err := repo.DeleteAllScheduleConfig(r.Context(), h.DB, clinicID); err != nil {
		log.Printf("[schedule] PUT schedule-config DeleteAllScheduleConfig: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	for _, d := range req.Days {
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			continue
		}
		s := &repo.ScheduleConfig{
			ClinicID:                    clinicID,
			DayOfWeek:                   d.DayOfWeek,
			Enabled:                     true,
			StartTime:                   d.StartTime,
			EndTime:                     d.EndTime,
			ConsultationDurationMinutes: 50,
			IntervalMinutes:             10,
			LunchStart:                  d.LunchStart,
			LunchEnd:                    d.LunchEnd,
		}
		if d.ConsultationDurationMinutes != nil && *d.ConsultationDurationMinutes > 0 {
			s.ConsultationDurationMinutes = *d.ConsultationDurationMinutes
		}
		if d.IntervalMinutes != nil && *d.IntervalMinutes >= 0 {
			s.IntervalMinutes = *d.IntervalMinutes
		}
		if err := repo.UpsertScheduleConfig(r.Context(), h.DB, s); err != nil {
			log.Printf("[schedule] PUT schedule-config UpsertScheduleConfig: %v", err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
	}
	if h.Cache != nil {
		h.Cache.Delete("schedule:" + clinicID.String())
	}
	// Read back from DB so response and next GET are consistent; fallback to req.Days if read returns nothing.
	list, err := repo.ListScheduleConfig(r.Context(), h.DB, clinicID)
	if err != nil {
		log.Printf("[schedule] PUT schedule-config ListScheduleConfig: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	formatTimeStr := func(p *string) interface{} {
		if p == nil {
			return nil
		}
		return *p
	}
	out := make([]map[string]interface{}, 7)
	for i := 0; i < 7; i++ {
		out[i] = map[string]interface{}{
			"day_of_week":                   i,
			"enabled":                       false,
			"start_time":                    nil,
			"end_time":                      nil,
			"consultation_duration_minutes": 50,
			"interval_minutes":              10,
			"lunch_start":                   nil,
			"lunch_end":                     nil,
		}
	}
	if len(list) > 0 {
		for _, s := range list {
			if s.DayOfWeek >= 0 && s.DayOfWeek < 7 {
				out[s.DayOfWeek] = map[string]interface{}{
					"day_of_week":                   s.DayOfWeek,
					"enabled":                       s.Enabled,
					"start_time":                    formatTimeStr(s.StartTime),
					"end_time":                      formatTimeStr(s.EndTime),
					"consultation_duration_minutes": s.ConsultationDurationMinutes,
					"interval_minutes":              s.IntervalMinutes,
					"lunch_start":                   formatTimeStr(s.LunchStart),
					"lunch_end":                     formatTimeStr(s.LunchEnd),
				}
			}
		}
	} else {
		log.Printf("[schedule] PUT schedule-config: no rows after upsert for clinic %s – response built from request body", clinicID.String())
		for _, d := range req.Days {
			if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
				continue
			}
			enabled := false
			if d.Enabled != nil {
				enabled = *d.Enabled
			}
			consultationDur := 50
			if d.ConsultationDurationMinutes != nil && *d.ConsultationDurationMinutes > 0 {
				consultationDur = *d.ConsultationDurationMinutes
			}
			interval := 10
			if d.IntervalMinutes != nil && *d.IntervalMinutes >= 0 {
				interval = *d.IntervalMinutes
			}
			out[d.DayOfWeek] = map[string]interface{}{
				"day_of_week":                   d.DayOfWeek,
				"enabled":                       enabled,
				"start_time":                    d.StartTime,
				"end_time":                      d.EndTime,
				"consultation_duration_minutes": consultationDur,
				"interval_minutes":              interval,
				"lunch_start":                   d.LunchStart,
				"lunch_end":                     d.LunchEnd,
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Schedule configuration saved.", "days": out})
}

// CopyScheduleConfigDay copies one day's schedule config to another.
func (h *Handler) CopyScheduleConfigDay(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
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
		FromDay int `json:"from_day"`
		ToDay   int `json:"to_day"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.FromDay < 0 || req.FromDay > 6 || req.ToDay < 0 || req.ToDay > 6 {
		http.Error(w, `{"error":"invalid day"}`, http.StatusBadRequest)
		return
	}
	if err := repo.CopyScheduleConfigDay(r.Context(), h.DB, clinicID, req.FromDay, req.ToDay); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Dia copiado."})
}

// ListAppointments lista compromissos da clínica em um período.
func (h *Handler) ListAppointments(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
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
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		http.Error(w, `{"error":"from and to required (YYYY-MM-DD)"}`, http.StatusBadRequest)
		return
	}
	from, err1 := time.Parse("2006-01-02", fromStr)
	to, err2 := time.Parse("2006-01-02", toStr)
	if err1 != nil || err2 != nil {
		http.Error(w, `{"error":"invalid date format"}`, http.StatusBadRequest)
		return
	}
	list, err := repo.ListAppointmentsByClinicAndDateRangeWithPatientName(r.Context(), h.DB, clinicID, from, to)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	out := make([]map[string]interface{}, len(list))
	for i, a := range list {
		contractID := ""
		if a.ContractID != nil {
			contractID = a.ContractID.String()
		}
		notes := ""
		if a.Notes != nil {
			notes = *a.Notes
		}
		out[i] = map[string]interface{}{
			"id":               a.ID.String(),
			"patient_id":       a.PatientID.String(),
			"patient_name":     a.PatientName,
			"contract_id":      contractID,
			"appointment_date": a.AppointmentDate.Format("2006-01-02"),
			"start_time":       repo.TimeStringToHHMM(a.StartTime),
			"end_time":         repo.TimeStringToHHMM(a.EndTime),
			"status":           a.Status,
			"notes":            notes,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"appointments": out})
}

// PatchAppointment altera um compromisso (data, horário, status, notas).
func (h *Handler) PatchAppointment(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
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
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		AppointmentDate *string `json:"appointment_date"`
		StartTime       *string `json:"start_time"`
		EndTime         *string `json:"end_time"`
		Status          *string `json:"status"`
		Notes           *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	var appointmentDate *time.Time
	if req.AppointmentDate != nil && *req.AppointmentDate != "" {
		t, err := time.Parse("2006-01-02", *req.AppointmentDate)
		if err == nil {
			appointmentDate = &t
		}
	}
	var startTime, endTime *time.Time
	if req.StartTime != nil && *req.StartTime != "" {
		t, err := time.Parse("15:04", *req.StartTime)
		if err == nil {
			startTime = &t
		}
	}
	if req.EndTime != nil && *req.EndTime != "" {
		t, err := time.Parse("15:04", *req.EndTime)
		if err == nil {
			endTime = &t
		}
	}
	if req.Status != nil {
		allowed := map[string]bool{
			"PRE_AGENDADO": true, "AGENDADO": true, "CONFIRMADO": true,
			"CANCELLED": true, "COMPLETED": true, "SERIES_ENDED": true,
		}
		if !allowed[*req.Status] {
			http.Error(w, `{"error":"invalid status"}`, http.StatusBadRequest)
			return
		}
	}
	if err := repo.UpdateAppointment(r.Context(), h.DB, id, clinicID, appointmentDate, startTime, endTime, req.Status, req.Notes); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Auditoria: registra apenas campos alterados (sem PII).
	changed := make([]string, 0, 5)
	if req.AppointmentDate != nil {
		changed = append(changed, "appointment_date")
	}
	if req.StartTime != nil {
		changed = append(changed, "start_time")
	}
	if req.EndTime != nil {
		changed = append(changed, "end_time")
	}
	if req.Status != nil {
		changed = append(changed, "status")
	}
	if req.Notes != nil {
		changed = append(changed, "notes")
	}
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if c := auth.ClaimsFrom(r.Context()); c != nil && c.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*c.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	resType := "APPOINTMENT"
	_ = repo.CreateAuditEventFull(r.Context(), h.DB, repo.AuditEvent{
		Action:                 "APPOINTMENT_UPDATED",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               &clinicID,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           &resType,
		ResourceID:             &id,
		PatientID:              nil,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"changed_fields": changed},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Compromisso atualizado."})
}

// EndContract define a data de término do contrato e encerra os agendamentos a partir dessa data.
func (h *Handler) EndContract(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientIDStr := mux.Vars(r)["patientId"]
	contractIDStr := mux.Vars(r)["contractId"]
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid contract_id"}`, http.StatusBadRequest)
		return
	}
	if !h.canAccessPatientAsProfessional(r, patientID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req struct {
		EndDate string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EndDate == "" {
		http.Error(w, `{"error":"end_date required (YYYY-MM-DD)"}`, http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		http.Error(w, `{"error":"invalid end_date"}`, http.StatusBadRequest)
		return
	}
	var c *repo.Contract
	clinicIDStr := auth.ClinicIDFrom(r.Context())
	if clinicIDStr != nil && *clinicIDStr != "" {
		clinicID, parseErr := uuid.Parse(*clinicIDStr)
		if parseErr != nil {
			http.Error(w, `{"error":"invalid clinic"}`, http.StatusBadRequest)
			return
		}
		c, err = repo.ContractByIDAndClinic(r.Context(), h.DB, contractID, clinicID)
		if err != nil {
			http.Error(w, `{"error":"contract not found"}`, http.StatusBadRequest)
			return
		}
	} else if auth.IsSuperAdmin(r.Context()) {
		c, err = repo.ContractByID(r.Context(), h.DB, contractID)
		if err != nil {
			http.Error(w, `{"error":"contract not found"}`, http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, `{"error":"no clinic"}`, http.StatusForbidden)
		return
	}
	clinicID := c.ClinicID
	if c.PatientID != patientID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if c.Status != "SIGNED" {
		http.Error(w, `{"error":"only signed contracts can be ended"}`, http.StatusBadRequest)
		return
	}
	if err := repo.SetContractEndDate(r.Context(), h.DB, contractID, clinicID, endDate); err != nil {
		log.Printf("[end-contract] SetContractEndDate failed: contract=%s clinic=%s err=%v", contractID, clinicID, err)
		http.Error(w, `{"error":"could not end contract. Check that status was not changed and try again."}`, http.StatusBadRequest)
		return
	}
	cancelledIDs, errCancel := repo.CancelAppointmentsByContractFromDateIDs(r.Context(), h.DB, contractID, endDate)
	_ = errCancel // logged below if len > 0; does not fail the response
	if len(cancelledIDs) > 0 {
		log.Printf("[end-contract] %d appointment(s) cancelled from %s", len(cancelledIDs), endDate.Format("02/01/2006"))
	}
	// Audit: contract ended + batch of appointments updated
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if cc := auth.ClaimsFrom(r.Context()); cc != nil && cc.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*cc.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	resType := "CONTRACT"
	_ = repo.CreateAuditEventFull(r.Context(), h.DB, repo.AuditEvent{
		Action:                 "CONTRACT_ENDED",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               &clinicID,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           &resType,
		ResourceID:             &contractID,
		PatientID:              &patientID,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"changed_fields": []string{"end_date", "status"}, "end_date": req.EndDate},
	})
	if len(cancelledIDs) > 0 {
		idStrs := make([]string, 0, len(cancelledIDs))
		for _, id := range cancelledIDs {
			idStrs = append(idStrs, id.String())
		}
		sys := "SYSTEM"
		_ = repo.CreateAuditEventFull(r.Context(), h.DB, repo.AuditEvent{
			Action:                 "APPOINTMENTS_SERIES_ENDED_BATCH",
			ActorType:              "SYSTEM",
			ActorID:                nil,
			ClinicID:               &clinicID,
			RequestID:              r.Header.Get("X-Request-ID"),
			IP:                     r.RemoteAddr,
			UserAgent:              r.UserAgent(),
			ResourceType:           nil,
			ResourceID:             nil,
			PatientID:              &patientID,
			IsImpersonated:         false,
			ImpersonationSessionID: nil,
			Source:                 &sys,
			Severity:               &sev,
			Metadata:               map[string]interface{}{"contract_id": contractID.String(), "affected_ids": idStrs, "count": len(idStrs)},
		})
	}
	// Email guardian: contract ended (service up to end date)
	if h.sendContractEndedEmail != nil {
		guardian, errG := repo.LegalGuardianByID(r.Context(), h.DB, c.LegalGuardianID)
		if errG == nil {
			endDateStr := endDate.Format("02/01/2006")
			log.Printf("[end-contract] sending end notification to %s", guardian.Email)
			if err := h.sendContractEndedEmail(guardian.Email, guardian.FullName, endDateStr); err != nil {
				log.Printf("[end-contract] failed to send email to %s: %v", guardian.Email, err)
			}
		} else {
			log.Printf("[end-contract] could not get guardian (legal_guardian_id=%s) to send email: %v", c.LegalGuardianID, errG)
		}
	} else {
		log.Printf("[end-contract] email disabled (contract ended)")
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Contract ended. Appointments from that date were finalized. Guardian was notified by email."})
}

// CreateAppointments creates one or more appointments linked to a signed contract.
func (h *Handler) CreateAppointments(w http.ResponseWriter, r *http.Request) {
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
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
		ContractID string `json:"contract_id"`
		Slots      []struct {
			AppointmentDate string `json:"appointment_date"`
			StartTime       string `json:"start_time"`
		} `json:"slots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.ContractID == "" || len(req.Slots) == 0 {
		http.Error(w, `{"error":"contract_id and at least one slot (appointment_date, start_time) required"}`, http.StatusBadRequest)
		return
	}
	contractID, err := uuid.Parse(req.ContractID)
	if err != nil {
		http.Error(w, `{"error":"invalid contract_id"}`, http.StatusBadRequest)
		return
	}
	contract, err := repo.ContractByIDAndClinic(r.Context(), h.DB, contractID, clinicID)
	if err != nil || contract == nil {
		http.Error(w, `{"error":"contract not found"}`, http.StatusBadRequest)
		return
	}
	if contract.Status != "SIGNED" {
		http.Error(w, `{"error":"contract must be signed to add appointments"}`, http.StatusBadRequest)
		return
	}
	professionalID := contract.ProfessionalID
	if professionalID == nil && auth.RoleFrom(r.Context()) == auth.RoleProfessional {
		userID := auth.UserIDFrom(r.Context())
		if userID != "" {
			if p, e := uuid.Parse(userID); e == nil {
				professionalID = &p
			}
		}
	}
	if professionalID == nil {
		http.Error(w, `{"error":"no professional for contract"}`, http.StatusBadRequest)
		return
	}
	const durationMin = 50
	created := 0
	createdIDs := make([]string, 0, len(req.Slots))
	for _, slot := range req.Slots {
		if slot.AppointmentDate == "" || slot.StartTime == "" {
			continue
		}
		appointmentDate, err1 := time.Parse("2006-01-02", slot.AppointmentDate)
		startTime, err2 := time.Parse("15:04", slot.StartTime)
		if err1 != nil || err2 != nil {
			continue
		}
		endTime := startTime.Add(time.Duration(durationMin) * time.Minute)
		apptID, err := repo.CreateAppointment(r.Context(), h.DB, clinicID, *professionalID, contract.PatientID, &contractID, appointmentDate, startTime, endTime, "AGENDADO", "")
		if err != nil {
			http.Error(w, `{"error":"failed to create appointment"}`, http.StatusInternalServerError)
			return
		}
		created++
		createdIDs = append(createdIDs, apptID.String())
	}
	// Auditoria: criação em lote
	var actorID *uuid.UUID
	if uid, e := uuid.Parse(auth.UserIDFrom(r.Context())); e == nil {
		actorID = &uid
	}
	actorType := auth.RoleFrom(r.Context())
	var sessionID *uuid.UUID
	if cc := auth.ClaimsFrom(r.Context()); cc != nil && cc.ImpersonationSessionID != nil {
		if sid, e := uuid.Parse(*cc.ImpersonationSessionID); e == nil {
			sessionID = &sid
		}
	}
	src := "USER"
	sev := "INFO"
	_ = repo.CreateAuditEventFull(r.Context(), h.DB, repo.AuditEvent{
		Action:                 "APPOINTMENTS_CREATED_BATCH",
		ActorType:              actorType,
		ActorID:                actorID,
		ClinicID:               &clinicID,
		RequestID:              r.Header.Get("X-Request-ID"),
		IP:                     r.RemoteAddr,
		UserAgent:              r.UserAgent(),
		ResourceType:           nil,
		ResourceID:             nil,
		PatientID:              &contract.PatientID,
		IsImpersonated:         auth.IsImpersonated(r.Context()),
		ImpersonationSessionID: sessionID,
		Source:                 &src,
		Severity:               &sev,
		Metadata:               map[string]interface{}{"contract_id": contractID.String(), "affected_ids": createdIDs, "count": created},
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Agendamentos criados.", "created": created})
}
