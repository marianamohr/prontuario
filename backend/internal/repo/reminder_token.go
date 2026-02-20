package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReminderTokenInfo holds valid appointment and guardian for a token.
// StartTime is string; PostgreSQL TIME is returned as string by the driver.
type ReminderTokenInfo struct {
	AppointmentID   uuid.UUID
	GuardianID      uuid.UUID
	ClinicID        uuid.UUID
	ProfessionalID  uuid.UUID
	PatientID       uuid.UUID
	PatientName     string
	AppointmentDate time.Time
	StartTime       string
	Status          string
}

// GetAppointmentByReminderToken validates the token and returns appointment data. Returns nil if invalid or expired.
func GetAppointmentByReminderToken(ctx context.Context, db *gorm.DB, token string) (*ReminderTokenInfo, error) {
	var r ReminderTokenInfo
	err := db.WithContext(ctx).Raw(`
		SELECT a.id as appointment_id, t.guardian_id, a.clinic_id, a.professional_id, a.patient_id, COALESCE(p.full_name, '') as patient_name,
		       a.appointment_date, a.start_time, a.status
		FROM appointment_reminder_tokens t
		JOIN appointments a ON a.id = t.appointment_id
		JOIN patients p ON p.id = a.patient_id
		WHERE t.token = ? AND t.expires_at > now()
	`, token).Scan(&r).Error
	if err != nil {
		return nil, err
	}
	if r.AppointmentID == uuid.Nil {
		return nil, nil
	}
	return &r, nil
}

// AvailableSlot is a slot available for reschedule.
type AvailableSlot struct {
	Date      string // 2006-01-02
	StartTime string // 15:04
}

// ListAvailableSlotsForProfessional returns available slots for the professional in [from, to].
// excludeAppointmentID: if non-nil, excludes that appointment from occupied slots (for reschedule).
func ListAvailableSlotsForProfessional(ctx context.Context, db *gorm.DB, professionalID, clinicID uuid.UUID, from, to time.Time, excludeAppointmentID *uuid.UUID) ([]AvailableSlot, error) {
	configs, err := ListScheduleConfig(ctx, db, clinicID)
	if err != nil {
		return nil, err
	}
	configMap := make(map[int]*ScheduleConfig)
	for i := range configs {
		configMap[configs[i].DayOfWeek] = &configs[i]
	}
	// Padrão GORM: Table + Where + Find (TIME no Postgres vem como string no driver).
	type occupiedSlot struct {
		AppointmentDate time.Time `gorm:"column:appointment_date"`
		StartTime      string    `gorm:"column:start_time"`
		EndTime        string    `gorm:"column:end_time"`
	}
	var existing []occupiedSlot
	query := db.WithContext(ctx).Table("appointments").
		Select("appointment_date, start_time, end_time").
		Where("professional_id = ? AND clinic_id = ? AND appointment_date >= ? AND appointment_date <= ?",
			professionalID, clinicID, from, to).
		Where("status NOT IN ?", []string{"CANCELLED", "SERIES_ENDED"})
	if excludeAppointmentID != nil {
		query = query.Where("id != ?", *excludeAppointmentID)
	}
	err = query.Find(&existing).Error
	if err != nil {
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
		startT := parseTimeOfDay(cfg.StartTime)
		endT := parseTimeOfDay(cfg.EndTime)
		if startT == nil || endT == nil {
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
		slotStart := time.Date(0, 1, 1, startT.Hour(), startT.Minute(), 0, 0, time.UTC)
		endTVal := time.Date(0, 1, 1, endT.Hour(), endT.Minute(), 0, 0, time.UTC)
		lunchStart := parseTimeOfDay(cfg.LunchStart)
		lunchEnd := parseTimeOfDay(cfg.LunchEnd)
		for slotStart.Before(endTVal) {
			slotEnd := slotStart.Add(time.Duration(dur) * time.Minute)
			if slotEnd.After(endTVal) {
				break
			}
			if lunchStart != nil && lunchEnd != nil {
				ls := time.Date(0, 1, 1, lunchStart.Hour(), lunchStart.Minute(), 0, 0, time.UTC)
				le := time.Date(0, 1, 1, lunchEnd.Hour(), lunchEnd.Minute(), 0, 0, time.UTC)
				if slotStart.Before(le) && slotEnd.After(ls) {
					slotStart = slotStart.Add(time.Duration(interval) * time.Minute)
					continue
				}
			}
			overlaps := false
			for _, e := range existing {
				if e.AppointmentDate.Year() != d.Year() || e.AppointmentDate.YearDay() != d.YearDay() {
					continue
				}
				esT := parseTimeOfDay(&e.StartTime)
				eeT := parseTimeOfDay(&e.EndTime)
				if esT == nil || eeT == nil {
					continue
				}
				es := time.Date(0, 1, 1, esT.Hour(), esT.Minute(), 0, 0, time.UTC)
				ee := time.Date(0, 1, 1, eeT.Hour(), eeT.Minute(), 0, 0, time.UTC)
				// Respeitar intervalo entre consultas: zona proibida = [es-interval, ee+interval]
				intervalDur := time.Duration(interval) * time.Minute
				forbiddenStart := es.Add(-intervalDur)
				forbiddenEnd := ee.Add(intervalDur)
				if slotStart.Before(forbiddenEnd) && slotEnd.After(forbiddenStart) {
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

// parseTimeOfDay interpreta "HH:MM" ou "HH:MM:SS" em *string e retorna *time.Time (só hora utilizada).
func parseTimeOfDay(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("15:04:05", *s)
	if err != nil {
		t, err = time.Parse("15:04", *s)
	}
	if err != nil {
		return nil
	}
	return &t
}
