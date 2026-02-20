package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScheduleConfig is the schedule configuration for one weekday (day_of_week 0=Sunday .. 6=Saturday).
// Enabled: only days with enabled=true are shown for configuration and on the agenda.
// Time fields are *string (e.g. "07:00" or "07:00:00"); PostgreSQL TIME is returned as string by the driver.
type ScheduleConfig struct {
	ClinicID                    uuid.UUID `gorm:"primaryKey;type:uuid"`
	DayOfWeek                   int       `gorm:"primaryKey"`
	Enabled                     bool      `gorm:"default:true"`
	StartTime                   *string   `gorm:"type:time"`
	EndTime                     *string   `gorm:"type:time"`
	ConsultationDurationMinutes int       `gorm:"default:50"`
	IntervalMinutes             int       `gorm:"default:10"`
	LunchStart                  *string   `gorm:"type:time"`
	LunchEnd                    *string   `gorm:"type:time"`
}

// TableName overrides GORM table name.
func (ScheduleConfig) TableName() string { return "clinic_schedule_config" }

func GetScheduleConfig(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, dayOfWeek int) (*ScheduleConfig, error) {
	var s ScheduleConfig
	err := db.WithContext(ctx).Where("clinic_id = ? AND day_of_week = ?", clinicID, dayOfWeek).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ListScheduleConfig(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) ([]ScheduleConfig, error) {
	var list []ScheduleConfig
	err := db.WithContext(ctx).Where("clinic_id = ?", clinicID).Order("day_of_week").Find(&list).Error
	return list, err
}

// UpsertScheduleConfig cria ou atualiza a configuração do dia (FirstOrCreate + Assign = upsert no GORM).
func UpsertScheduleConfig(ctx context.Context, db *gorm.DB, s *ScheduleConfig) error {
	return db.WithContext(ctx).
		Where("clinic_id = ? AND day_of_week = ?", s.ClinicID, s.DayOfWeek).
		Assign(s).
		FirstOrCreate(s).Error
}

// DeleteScheduleConfigDay remove a configuração de um dia (quando o profissional desmarca o dia).
func DeleteScheduleConfigDay(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, dayOfWeek int) error {
	return db.WithContext(ctx).
		Where("clinic_id = ? AND day_of_week = ?", clinicID, dayOfWeek).
		Delete(&ScheduleConfig{}).Error
}

// DeleteAllScheduleConfig remove todas as configurações de dias da clínica (para substituir pela nova lista).
func DeleteAllScheduleConfig(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) error {
	return db.WithContext(ctx).Where("clinic_id = ?", clinicID).Delete(&ScheduleConfig{}).Error
}

func CopyScheduleConfigDay(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, fromDay, toDay int) error {
	fromC, err := GetScheduleConfig(ctx, db, clinicID, fromDay)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Dia de origem sem configuração: usar padrão (50 min, 10 interval, sem horários); destino fica habilitado
			fromC = &ScheduleConfig{
				ClinicID:                    clinicID,
				DayOfWeek:                   fromDay,
				Enabled:                     true,
				ConsultationDurationMinutes: 50,
				IntervalMinutes:             10,
			}
		} else {
			return err
		}
	}
	toC := &ScheduleConfig{
		ClinicID:                    clinicID,
		DayOfWeek:                   toDay,
		Enabled:                     fromC.Enabled,
		StartTime:                   fromC.StartTime,
		EndTime:                     fromC.EndTime,
		ConsultationDurationMinutes: fromC.ConsultationDurationMinutes,
		IntervalMinutes:             fromC.IntervalMinutes,
		LunchStart:                  fromC.LunchStart,
		LunchEnd:                    fromC.LunchEnd,
	}
	return UpsertScheduleConfig(ctx, db, toC)
}

// ContractScheduleRule is a pre-schedule rule on the contract (e.g. day 2 = Tuesday, 15:00).
// SlotTime is string (e.g. "15:00:00"); PostgreSQL TIME is returned as string by the driver.
type ContractScheduleRule struct {
	ID         uuid.UUID
	ContractID uuid.UUID
	DayOfWeek  int
	SlotTime   string `gorm:"column:slot_time;type:time"`
}

func CreateContractScheduleRules(ctx context.Context, db *gorm.DB, contractID uuid.UUID, rules []ContractScheduleRule) error {
	if len(rules) == 0 {
		return nil
	}
	// Batch insert: one query with multiple VALUES to avoid N round-trips.
	const cols = 3 // contract_id, day_of_week, slot_time
	args := make([]interface{}, 0, len(rules)*cols)
	placeholders := make([]string, 0, len(rules))
	for i, r := range rules {
		args = append(args, contractID, r.DayOfWeek, r.SlotTime)
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", i*cols+1, i*cols+2, i*cols+3))
	}
	query := `INSERT INTO contract_schedule_rules (contract_id, day_of_week, slot_time) VALUES ` + strings.Join(placeholders, ", ")
	return db.WithContext(ctx).Exec(query, args...).Error
}

// ParseSlotTimeOnDate parses a time string ("15:04" or "15:04:05") and returns a time.Time on the given date.
func ParseSlotTimeOnDate(slotTimeStr string, date time.Time) (time.Time, error) {
	slotTimeStr = strings.TrimSpace(slotTimeStr)
	var t time.Time
	var err error
	if len(slotTimeStr) > 5 && slotTimeStr[5] == ':' {
		t, err = time.Parse("15:04:05", slotTimeStr)
	} else {
		t, err = time.Parse("15:04", slotTimeStr)
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), t.Second(), 0, date.Location()), nil
}

func ListContractScheduleRules(ctx context.Context, db *gorm.DB, contractID uuid.UUID) ([]ContractScheduleRule, error) {
	var list []ContractScheduleRule
	err := db.WithContext(ctx).Raw("SELECT id, contract_id, day_of_week, slot_time FROM contract_schedule_rules WHERE contract_id = ? ORDER BY day_of_week, slot_time", contractID).Scan(&list).Error
	return list, err
}

// Appointment is an agenda appointment.
// StartTime and EndTime are string (e.g. "09:00:00"); PostgreSQL TIME is returned as string by the driver.
type Appointment struct {
	ID              uuid.UUID
	ClinicID        uuid.UUID
	ProfessionalID  uuid.UUID
	PatientID       uuid.UUID
	ContractID      *uuid.UUID
	AppointmentDate time.Time
	StartTime       string `gorm:"column:start_time;type:time"`
	EndTime         string `gorm:"column:end_time;type:time"`
	Status          string
	Notes           *string
}

// TimeStringToHHMM returns "HH:MM" from a DB time string ("HH:MM:SS" or "HH:MM").
func TimeStringToHHMM(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

func CreateAppointment(ctx context.Context, db *gorm.DB, clinicID, professionalID, patientID uuid.UUID, contractID *uuid.UUID, appointmentDate time.Time, startTime, endTime time.Time, status, notes string) (uuid.UUID, error) {
	var n *string
	if notes != "" {
		n = &notes
	}
	startStr := startTime.Format("15:04:05")
	endStr := endTime.Format("15:04:05")
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		INSERT INTO appointments (clinic_id, professional_id, patient_id, contract_id, appointment_date, start_time, end_time, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
	`, clinicID, professionalID, patientID, contractID, appointmentDate, startStr, endStr, status, n).Scan(&res).Error
	return res.ID, err
}

func ListAppointmentsByClinicAndDateRange(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, from, to time.Time) ([]Appointment, error) {
	var list []Appointment
	err := db.WithContext(ctx).Raw(`
		SELECT a.id, a.clinic_id, a.professional_id, a.patient_id, a.contract_id, a.appointment_date, a.start_time, a.end_time, a.status, a.notes
		FROM appointments a
		WHERE a.clinic_id = ? AND a.appointment_date >= ? AND a.appointment_date <= ? AND a.status NOT IN ('CANCELLED', 'SERIES_ENDED')
		ORDER BY a.appointment_date, a.start_time
	`, clinicID, from, to).Scan(&list).Error
	return list, err
}

// AppointmentWithPatientName is an appointment with patient name (for agenda display).
type AppointmentWithPatientName struct {
	Appointment
	PatientName string
}

func ListAppointmentsByClinicAndDateRangeWithPatientName(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, from, to time.Time) ([]AppointmentWithPatientName, error) {
	var list []AppointmentWithPatientName
	err := db.WithContext(ctx).Raw(`
		SELECT a.id, a.clinic_id, a.professional_id, a.patient_id, a.contract_id, a.appointment_date, a.start_time, a.end_time, a.status, a.notes, COALESCE(p.full_name, '') as patient_name
		FROM appointments a
		LEFT JOIN patients p ON p.id = a.patient_id AND p.deleted_at IS NULL
		WHERE a.clinic_id = ? AND a.appointment_date >= ? AND a.appointment_date <= ? AND a.status NOT IN ('CANCELLED', 'SERIES_ENDED')
		ORDER BY a.appointment_date, a.start_time
	`, clinicID, from, to).Scan(&list).Error
	return list, err
}

func AppointmentByIDAndClinic(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) (*Appointment, error) {
	var a Appointment
	err := db.WithContext(ctx).Table("appointments").Where("id = ? AND clinic_id = ?", id, clinicID).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func UpdateAppointment(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID, appointmentDate *time.Time, startTime, endTime *time.Time, status *string, notes *string) error {
	updates := map[string]interface{}{"updated_at": gorm.Expr("now()")}
	if appointmentDate != nil {
		updates["appointment_date"] = *appointmentDate
	}
	if startTime != nil {
		updates["start_time"] = startTime.Format("15:04:05")
	}
	if endTime != nil {
		updates["end_time"] = endTime.Format("15:04:05")
	}
	if status != nil {
		updates["status"] = *status
	}
	if notes != nil {
		updates["notes"] = *notes
	}
	return db.WithContext(ctx).Table("appointments").Where("id = ? AND clinic_id = ?", id, clinicID).Updates(updates).Error
}

// CancelAppointmentsByContractFromDate cancels appointments with date after the end date.
// E.g. end on 15/02 keeps 14/02 and 15/02, cancels 16/02 onward.
func CancelAppointmentsByContractFromDate(ctx context.Context, db *gorm.DB, contractID uuid.UUID, endDate time.Time) (int64, error) {
	endDateStr := endDate.Format("2006-01-02")
	result := db.WithContext(ctx).Exec(`
		UPDATE appointments SET status = 'SERIES_ENDED', updated_at = now()
		WHERE contract_id = ? AND appointment_date > ?::date AND status NOT IN ('CANCELLED', 'SERIES_ENDED', 'COMPLETED')
	`, contractID, endDateStr)
	return result.RowsAffected, result.Error
}

// CancelAppointmentsByContractFromDateIDs cancels appointments and returns the affected IDs.
func CancelAppointmentsByContractFromDateIDs(ctx context.Context, db *gorm.DB, contractID uuid.UUID, endDate time.Time) ([]uuid.UUID, error) {
	endDateStr := endDate.Format("2006-01-02")
	var rows []struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		UPDATE appointments
		SET status = 'SERIES_ENDED', updated_at = now()
		WHERE contract_id = ? AND appointment_date > ?::date AND status NOT IN ('CANCELLED', 'SERIES_ENDED', 'COMPLETED')
		RETURNING id
	`, contractID, endDateStr).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	return ids, nil
}

// CancelAppointmentsByContract marks as CANCELLED all appointments linked to the contract (except already completed).
func CancelAppointmentsByContract(ctx context.Context, db *gorm.DB, contractID uuid.UUID) (int64, error) {
	result := db.WithContext(ctx).Exec(`
		UPDATE appointments SET status = 'CANCELLED', updated_at = now()
		WHERE contract_id = ? AND status NOT IN ('CANCELLED', 'COMPLETED')
	`, contractID)
	return result.RowsAffected, result.Error
}

// CancelAppointmentsByContractIDs marks as CANCELLED and returns the affected IDs.
func CancelAppointmentsByContractIDs(ctx context.Context, db *gorm.DB, contractID uuid.UUID) ([]uuid.UUID, error) {
	var rows []struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		UPDATE appointments
		SET status = 'CANCELLED', updated_at = now()
		WHERE contract_id = ? AND status NOT IN ('CANCELLED', 'COMPLETED')
		RETURNING id
	`, contractID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	return ids, nil
}

// UpdateAppointmentsStatusByContract updates appointment status for a contract (e.g. PRE_AGENDADO -> AGENDADO on sign).
func UpdateAppointmentsStatusByContract(ctx context.Context, db *gorm.DB, contractID uuid.UUID, newStatus string) (int64, error) {
	result := db.WithContext(ctx).Exec(`
		UPDATE appointments SET status = ?, updated_at = now()
		WHERE contract_id = ? AND status NOT IN ('CANCELLED', 'COMPLETED', 'SERIES_ENDED')
	`, newStatus, contractID)
	return result.RowsAffected, result.Error
}

// CreateAppointmentsFromContractRules creates appointments from contract rules, from startDate to endDate (inclusive).
// Appointments are created with contract_id set (required for cancellation when ending contract).
// professionalID and clinicID come from the contract; durationMinutes is used for end_time (start + duration).
// maxAppointments: if > 0, creates at most that many appointments; 0 = no limit.
func CreateAppointmentsFromContractRules(ctx context.Context, db *gorm.DB, contractID, clinicID, professionalID, patientID uuid.UUID, startDate, endDate time.Time, durationMinutes int, maxAppointments int) error {
	return CreateAppointmentsFromContractRulesWithStatus(ctx, db, contractID, clinicID, professionalID, patientID, startDate, endDate, durationMinutes, maxAppointments, "AGENDADO")
}

// CreateAppointmentsFromContractRulesWithStatus creates appointments from contract rules with the given status.
// Used when sending contract (PRE_AGENDADO) or in flows that need another status.
func CreateAppointmentsFromContractRulesWithStatus(ctx context.Context, db *gorm.DB, contractID, clinicID, professionalID, patientID uuid.UUID, startDate, endDate time.Time, durationMinutes int, maxAppointments int, status string) error {
	if contractID == uuid.Nil {
		return fmt.Errorf("contract_id is required")
	}
	rules, err := ListContractScheduleRules(ctx, db, contractID)
	if err != nil || len(rules) == 0 {
		return err
	}
	if durationMinutes <= 0 {
		durationMinutes = 50
	}
	created := 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		if maxAppointments > 0 && created >= maxAppointments {
			break
		}
		weekday := int(d.Weekday())
		for _, r := range rules {
			if maxAppointments > 0 && created >= maxAppointments {
				break
			}
			if r.DayOfWeek != weekday {
				continue
			}
			startTime, errParse := ParseSlotTimeOnDate(r.SlotTime, d)
			if errParse != nil {
				return errParse
			}
			endTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)
			_, err := CreateAppointment(ctx, db, clinicID, professionalID, patientID, &contractID, d, startTime, endTime, status, "")
			if err != nil {
				return err
			}
			created++
		}
	}
	return nil
}

// AppointmentReminderRow holds data to send one reminder (one guardian phone per row).
// StartTime is string; PostgreSQL TIME is returned as string by the driver.
type AppointmentReminderRow struct {
	AppointmentID   uuid.UUID
	PatientID       uuid.UUID
	PatientName     string
	AppointmentDate time.Time
	StartTime       string
	GuardianID      uuid.UUID
	GuardianPhone   string
}

// ListAppointmentsForReminder returns appointments on the given date with guardian phone, for reminder WhatsApp.
// Only status AGENDADO and CONFIRMADO; only guardians with non-empty phone; excludes soft-deleted guardians.
func ListAppointmentsForReminder(ctx context.Context, db *gorm.DB, date time.Time) ([]AppointmentReminderRow, error) {
	dateStr := date.Format("2006-01-02")
	var list []AppointmentReminderRow
	err := db.WithContext(ctx).Raw(`
		SELECT a.id as appointment_id, p.id as patient_id, COALESCE(p.full_name, '') as patient_name, a.appointment_date, a.start_time, g.id as guardian_id, TRIM(g.phone) as guardian_phone
		FROM appointments a
		JOIN patients p ON p.id = a.patient_id AND p.deleted_at IS NULL
		JOIN patient_guardians pg ON pg.patient_id = a.patient_id
		JOIN legal_guardians g ON g.id = pg.legal_guardian_id AND g.deleted_at IS NULL AND g.status != 'CANCELLED'
		WHERE a.appointment_date = ?::date
		  AND a.status IN ('AGENDADO', 'CONFIRMADO')
		  AND g.phone IS NOT NULL AND TRIM(g.phone) != ''
		ORDER BY a.start_time, g.full_name
	`, dateStr).Scan(&list).Error
	return list, err
}
