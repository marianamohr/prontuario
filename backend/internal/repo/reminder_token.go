package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReminderTokenInfo retorna appointment + guardian válidos para um token.
type ReminderTokenInfo struct {
	AppointmentID   uuid.UUID
	GuardianID      uuid.UUID
	ClinicID        uuid.UUID
	ProfessionalID  uuid.UUID
	PatientID       uuid.UUID
	PatientName     string
	AppointmentDate time.Time
	StartTime       time.Time
	Status          string
}

// GetAppointmentByReminderToken valida o token e retorna dados do compromisso. Retorna nil se inválido/expirado.
func GetAppointmentByReminderToken(ctx context.Context, pool *pgxpool.Pool, token string) (*ReminderTokenInfo, error) {
	var r ReminderTokenInfo
	err := pool.QueryRow(ctx, `
		SELECT a.id, t.guardian_id, a.clinic_id, a.professional_id, a.patient_id, COALESCE(p.full_name, ''),
		       a.appointment_date, a.start_time, a.status
		FROM appointment_reminder_tokens t
		JOIN appointments a ON a.id = t.appointment_id
		JOIN patients p ON p.id = a.patient_id
		WHERE t.token = $1 AND t.expires_at > now()
	`, token).Scan(&r.AppointmentID, &r.GuardianID, &r.ClinicID, &r.ProfessionalID, &r.PatientID, &r.PatientName, &r.AppointmentDate, &r.StartTime, &r.Status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	// appointments doesn't have legal_guardian_id directly - we get guardian from patient_guardians
	// Actually the token has guardian_id. We need to join to get the appointment. Let me fix the query.
	// The appointment_reminder_tokens has appointment_id, guardian_id. Appointments has patient_id, not guardian_id.
	// So we join: t -> appointments a, and we need to verify that guardian_id is a guardian of the patient.
	// Simplified: we trust the token - if it exists and not expired, the (appointment_id, guardian_id) pair is valid.
	// Let me fix the query - we don't have legal_guardian_id in appointments. The join is just t -> a.
	return &r, nil
}

// Slot disponível para remarcação.
type AvailableSlot struct {
	Date      string // 2006-01-02
	StartTime string // 15:04
}

// ListAvailableSlotsForProfessional retorna slots disponíveis para o profissional em [from, to].
// excludeAppointmentID: se non-nil, exclui esse compromisso da lista de ocupados (para remarcação).
func ListAvailableSlotsForProfessional(ctx context.Context, pool *pgxpool.Pool, professionalID, clinicID uuid.UUID, from, to time.Time, excludeAppointmentID *uuid.UUID) ([]AvailableSlot, error) {
	configs, err := ListScheduleConfig(ctx, pool, clinicID)
	if err != nil {
		return nil, err
	}
	configMap := make(map[int]*ScheduleConfig)
	for i := range configs {
		configMap[configs[i].DayOfWeek] = &configs[i]
	}
	args := []interface{}{professionalID, clinicID, from, to}
	q := `
		SELECT appointment_date, start_time, end_time
		FROM appointments
		WHERE professional_id = $1 AND clinic_id = $2
		  AND appointment_date >= $3 AND appointment_date <= $4
		  AND status NOT IN ('CANCELLED', 'SERIES_ENDED')
	`
	if excludeAppointmentID != nil {
		q += ` AND id != $5`
		args = append(args, excludeAppointmentID)
	}
	appointments, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer appointments.Close()
	type appt struct {
		date  time.Time
		start time.Time
		end   time.Time
	}
	var existing []appt
	for appointments.Next() {
		var a appt
		var d time.Time
		if err := appointments.Scan(&d, &a.start, &a.end); err != nil {
			return nil, err
		}
		a.date = d
		existing = append(existing, a)
	}
	if err := appointments.Err(); err != nil {
		return nil, err
	}
	var slots []AvailableSlot
	const defaultDuration = 50
	const defaultInterval = 10
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dayOfWeek := int(d.Weekday())
		cfg := configMap[dayOfWeek]
		if cfg == nil || !cfg.Enabled || cfg.StartTime == nil || cfg.EndTime == nil {
			continue
		}
		dur := cfg.ConsultationDurationMinutes
		if dur <= 0 {
			dur = defaultDuration
		}
		interval := cfg.IntervalMinutes
		if interval <= 0 {
			interval = defaultInterval
		}
		// slot start in same-day time
		slotStart := time.Date(0, 1, 1, cfg.StartTime.Hour(), cfg.StartTime.Minute(), 0, 0, time.UTC)
		endT := time.Date(0, 1, 1, cfg.EndTime.Hour(), cfg.EndTime.Minute(), 0, 0, time.UTC)
		lunchStart, lunchEnd := cfg.LunchStart, cfg.LunchEnd
		for slotStart.Before(endT) {
			slotEnd := slotStart.Add(time.Duration(dur) * time.Minute)
			if slotEnd.After(endT) {
				break
			}
			if lunchStart != nil && lunchEnd != nil {
				ls := time.Date(0, 1, 1, lunchStart.Hour(), lunchStart.Minute(), 0, 0, time.UTC)
				le := time.Date(0, 1, 1, lunchEnd.Hour(), lunchEnd.Minute(), 0, 0, time.UTC)
				if (slotStart.Before(le) && slotEnd.After(ls)) {
					slotStart = slotStart.Add(time.Duration(interval) * time.Minute)
					continue
				}
			}
			// check overlap with existing
			overlaps := false
			for _, e := range existing {
				if e.date.Year() != d.Year() || e.date.YearDay() != d.YearDay() {
					continue
				}
				es := time.Date(0, 1, 1, e.start.Hour(), e.start.Minute(), 0, 0, time.UTC)
				ee := time.Date(0, 1, 1, e.end.Hour(), e.end.Minute(), 0, 0, time.UTC)
				if slotStart.Before(ee) && slotEnd.After(es) {
					overlaps = true
					break
				}
			}
			if !overlaps {
				slots = append(slots, AvailableSlot{
					Date:      d.Format("2006-01-02"),
					StartTime: slotStart.Format("15:04"),
				})
			}
			slotStart = slotStart.Add(time.Duration(interval) * time.Minute)
		}
	}
	return slots, nil
}
