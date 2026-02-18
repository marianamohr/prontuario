package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScheduleConfig é a configuração de agenda de um dia da semana (day_of_week 0=domingo .. 6=sábado).
// Enabled: só dias com enabled=true aparecem para configurar e na agenda.
type ScheduleConfig struct {
	ClinicID                    uuid.UUID
	DayOfWeek                   int
	Enabled                     bool
	StartTime                   *time.Time
	EndTime                     *time.Time
	ConsultationDurationMinutes int
	IntervalMinutes             int
	LunchStart                  *time.Time
	LunchEnd                    *time.Time
}

func GetScheduleConfig(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, dayOfWeek int) (*ScheduleConfig, error) {
	var s ScheduleConfig
	var start, end, lunchStart, lunchEnd interface{}
	err := pool.QueryRow(ctx, `
		SELECT clinic_id, day_of_week, COALESCE(enabled, true), start_time, end_time, consultation_duration_minutes, interval_minutes, lunch_start, lunch_end
		FROM clinic_schedule_config WHERE clinic_id = $1 AND day_of_week = $2
	`, clinicID, dayOfWeek).Scan(&s.ClinicID, &s.DayOfWeek, &s.Enabled, &start, &end, &s.ConsultationDurationMinutes, &s.IntervalMinutes, &lunchStart, &lunchEnd)
	if err != nil {
		return nil, err
	}
	if t, ok := start.(time.Time); ok {
		s.StartTime = &t
	}
	if t, ok := end.(time.Time); ok {
		s.EndTime = &t
	}
	if t, ok := lunchStart.(time.Time); ok {
		s.LunchStart = &t
	}
	if t, ok := lunchEnd.(time.Time); ok {
		s.LunchEnd = &t
	}
	return &s, nil
}

func ListScheduleConfig(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) ([]ScheduleConfig, error) {
	rows, err := pool.Query(ctx, `
		SELECT clinic_id, day_of_week, COALESCE(enabled, true), start_time, end_time, consultation_duration_minutes, interval_minutes, lunch_start, lunch_end
		FROM clinic_schedule_config WHERE clinic_id = $1 ORDER BY day_of_week
	`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ScheduleConfig
	for rows.Next() {
		var s ScheduleConfig
		var start, end, lunchStart, lunchEnd interface{}
		if err := rows.Scan(&s.ClinicID, &s.DayOfWeek, &s.Enabled, &start, &end, &s.ConsultationDurationMinutes, &s.IntervalMinutes, &lunchStart, &lunchEnd); err != nil {
			return nil, err
		}
		if t, ok := start.(time.Time); ok {
			s.StartTime = &t
		}
		if t, ok := end.(time.Time); ok {
			s.EndTime = &t
		}
		if t, ok := lunchStart.(time.Time); ok {
			s.LunchStart = &t
		}
		if t, ok := lunchEnd.(time.Time); ok {
			s.LunchEnd = &t
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func UpsertScheduleConfig(ctx context.Context, pool *pgxpool.Pool, s *ScheduleConfig) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO clinic_schedule_config (clinic_id, day_of_week, enabled, start_time, end_time, consultation_duration_minutes, interval_minutes, lunch_start, lunch_end, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (clinic_id, day_of_week) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time,
			consultation_duration_minutes = EXCLUDED.consultation_duration_minutes, interval_minutes = EXCLUDED.interval_minutes,
			lunch_start = EXCLUDED.lunch_start, lunch_end = EXCLUDED.lunch_end, updated_at = now()
	`, s.ClinicID, s.DayOfWeek, s.Enabled, s.StartTime, s.EndTime, s.ConsultationDurationMinutes, s.IntervalMinutes, s.LunchStart, s.LunchEnd)
	return err
}

func CopyScheduleConfigDay(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, fromDay, toDay int) error {
	fromC, err := GetScheduleConfig(ctx, pool, clinicID, fromDay)
	if err != nil {
		if err == pgx.ErrNoRows {
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
	return UpsertScheduleConfig(ctx, pool, toC)
}

// ContractScheduleRule é uma regra de pré-agendamento no contrato (ex.: dia 2 = terça, 15:00).
type ContractScheduleRule struct {
	ID         uuid.UUID
	ContractID uuid.UUID
	DayOfWeek  int
	SlotTime   time.Time
}

func CreateContractScheduleRules(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID, rules []ContractScheduleRule) error {
	for _, r := range rules {
		_, err := pool.Exec(ctx, `INSERT INTO contract_schedule_rules (contract_id, day_of_week, slot_time) VALUES ($1, $2, $3)`,
			contractID, r.DayOfWeek, r.SlotTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func ListContractScheduleRules(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID) ([]ContractScheduleRule, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, contract_id, day_of_week, slot_time FROM contract_schedule_rules WHERE contract_id = $1 ORDER BY day_of_week, slot_time
	`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ContractScheduleRule
	for rows.Next() {
		var r ContractScheduleRule
		if err := rows.Scan(&r.ID, &r.ContractID, &r.DayOfWeek, &r.SlotTime); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// Appointment é um compromisso na agenda.
type Appointment struct {
	ID              uuid.UUID
	ClinicID        uuid.UUID
	ProfessionalID  uuid.UUID
	PatientID       uuid.UUID
	ContractID      *uuid.UUID
	AppointmentDate time.Time
	StartTime       time.Time
	EndTime         time.Time
	Status          string
	Notes           *string
}

func CreateAppointment(ctx context.Context, pool *pgxpool.Pool, clinicID, professionalID, patientID uuid.UUID, contractID *uuid.UUID, appointmentDate time.Time, startTime, endTime time.Time, status, notes string) (uuid.UUID, error) {
	var id uuid.UUID
	var n *string
	if notes != "" {
		n = &notes
	}
	err := pool.QueryRow(ctx, `
		INSERT INTO appointments (clinic_id, professional_id, patient_id, contract_id, appointment_date, start_time, end_time, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id
	`, clinicID, professionalID, patientID, contractID, appointmentDate, startTime, endTime, status, n).Scan(&id)
	return id, err
}

func ListAppointmentsByClinicAndDateRange(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, from, to time.Time) ([]Appointment, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.id, a.clinic_id, a.professional_id, a.patient_id, a.contract_id, a.appointment_date, a.start_time, a.end_time, a.status, a.notes
		FROM appointments a
		WHERE a.clinic_id = $1 AND a.appointment_date >= $2 AND a.appointment_date <= $3 AND a.status NOT IN ('CANCELLED', 'SERIES_ENDED')
		ORDER BY a.appointment_date, a.start_time
	`, clinicID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAppointments(rows)
}

// AppointmentWithPatientName é um compromisso com o nome do paciente (para exibição na agenda).
type AppointmentWithPatientName struct {
	Appointment
	PatientName string
}

func ListAppointmentsByClinicAndDateRangeWithPatientName(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, from, to time.Time) ([]AppointmentWithPatientName, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.id, a.clinic_id, a.professional_id, a.patient_id, a.contract_id, a.appointment_date, a.start_time, a.end_time, a.status, a.notes, COALESCE(p.full_name, '')
		FROM appointments a
		LEFT JOIN patients p ON p.id = a.patient_id
		WHERE a.clinic_id = $1 AND a.appointment_date >= $2 AND a.appointment_date <= $3 AND a.status NOT IN ('CANCELLED', 'SERIES_ENDED')
		ORDER BY a.appointment_date, a.start_time
	`, clinicID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AppointmentWithPatientName
	for rows.Next() {
		var a AppointmentWithPatientName
		var contractID *uuid.UUID
		if err := rows.Scan(&a.ID, &a.ClinicID, &a.ProfessionalID, &a.PatientID, &contractID, &a.AppointmentDate, &a.StartTime, &a.EndTime, &a.Status, &a.Notes, &a.PatientName); err != nil {
			return nil, err
		}
		a.ContractID = contractID
		list = append(list, a)
	}
	return list, rows.Err()
}

func AppointmentByIDAndClinic(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) (*Appointment, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, clinic_id, professional_id, patient_id, contract_id, appointment_date, start_time, end_time, status, notes
		FROM appointments WHERE id = $1 AND clinic_id = $2
	`, id, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanAppointments(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func UpdateAppointment(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID, appointmentDate *time.Time, startTime, endTime *time.Time, status *string, notes *string) error {
	// Build dynamic update: only non-nil fields
	_, err := pool.Exec(ctx, `
		UPDATE appointments SET
			appointment_date = COALESCE($1, appointment_date),
			start_time = COALESCE($2, start_time),
			end_time = COALESCE($3, end_time),
			status = COALESCE($4, status),
			notes = COALESCE($5, notes),
			updated_at = now()
		WHERE id = $6 AND clinic_id = $7
	`, appointmentDate, startTime, endTime, status, notes, id, clinicID)
	return err
}

// CancelAppointmentsByContractFromDate cancela agendamentos com data APÓS a data de término.
// Ex.: encerramento em 15/02 → mantém 14/02 e 15/02, cancela 16/02 em diante.
func CancelAppointmentsByContractFromDate(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID, endDate time.Time) (int64, error) {
	endDateStr := endDate.Format("2006-01-02")
	result, err := pool.Exec(ctx, `
		UPDATE appointments SET status = 'SERIES_ENDED', updated_at = now()
		WHERE contract_id = $1 AND appointment_date > $2::date AND status NOT IN ('CANCELLED', 'SERIES_ENDED', 'COMPLETED')
	`, contractID, endDateStr)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// CancelAppointmentsByContractFromDateIDs cancela agendamentos e retorna os IDs alterados.
func CancelAppointmentsByContractFromDateIDs(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID, endDate time.Time) ([]uuid.UUID, error) {
	endDateStr := endDate.Format("2006-01-02")
	rows, err := pool.Query(ctx, `
		UPDATE appointments
		SET status = 'SERIES_ENDED', updated_at = now()
		WHERE contract_id = $1 AND appointment_date > $2::date AND status NOT IN ('CANCELLED', 'SERIES_ENDED', 'COMPLETED')
		RETURNING id
	`, contractID, endDateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CancelAppointmentsByContract marca como CANCELLED todos os agendamentos vinculados ao contrato (exceto já concluídos).
func CancelAppointmentsByContract(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID) (int64, error) {
	result, err := pool.Exec(ctx, `
		UPDATE appointments SET status = 'CANCELLED', updated_at = now()
		WHERE contract_id = $1 AND status NOT IN ('CANCELLED', 'COMPLETED')
	`, contractID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// CancelAppointmentsByContractIDs marca como CANCELLED e retorna os IDs alterados.
func CancelAppointmentsByContractIDs(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
		UPDATE appointments
		SET status = 'CANCELLED', updated_at = now()
		WHERE contract_id = $1 AND status NOT IN ('CANCELLED', 'COMPLETED')
		RETURNING id
	`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CreateAppointmentsFromContractRules gera os compromissos a partir das regras do contrato, do startDate ao endDate (inclusive).
// Os agendamentos são criados com contract_id preenchido (obrigatório para cancelamento ao encerrar contrato).
// professionalID e clinicID vêm do contrato; durationMinutos é usado para end_time (start + duration).
// maxAppointments: se > 0, cria no máximo esse número de agendamentos; 0 = sem limite.
func CreateAppointmentsFromContractRules(ctx context.Context, pool *pgxpool.Pool, contractID, clinicID, professionalID, patientID uuid.UUID, startDate, endDate time.Time, durationMinutes int, maxAppointments int) error {
	if contractID == uuid.Nil {
		return fmt.Errorf("contract_id é obrigatório para agendamentos criados na assinatura")
	}
	rules, err := ListContractScheduleRules(ctx, pool, contractID)
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
			startTime := r.SlotTime
			endTime := r.SlotTime.Add(time.Duration(durationMinutes) * time.Minute)
			_, err := CreateAppointment(ctx, pool, clinicID, professionalID, patientID, &contractID, d, startTime, endTime, "CONFIRMED", "")
			if err != nil {
				return err
			}
			created++
		}
	}
	return nil
}

func scanAppointments(rows pgx.Rows) ([]Appointment, error) {
	var list []Appointment
	for rows.Next() {
		var a Appointment
		var contractID *uuid.UUID
		if err := rows.Scan(&a.ID, &a.ClinicID, &a.ProfessionalID, &a.PatientID, &contractID, &a.AppointmentDate, &a.StartTime, &a.EndTime, &a.Status, &a.Notes); err != nil {
			return nil, err
		}
		a.ContractID = contractID
		list = append(list, a)
	}
	return list, rows.Err()
}

// AppointmentReminderRow holds data to send one reminder (one guardian phone per row).
type AppointmentReminderRow struct {
	AppointmentID   uuid.UUID
	PatientID       uuid.UUID
	PatientName     string
	AppointmentDate time.Time
	StartTime       time.Time
	GuardianID      uuid.UUID
	GuardianPhone   string
}

// ListAppointmentsForReminder returns appointments on the given date with guardian phone, for reminder WhatsApp.
// Only status CONFIRMED and PENDING_SIGNATURE; only guardians with non-empty phone; excludes soft-deleted guardians.
func ListAppointmentsForReminder(ctx context.Context, pool *pgxpool.Pool, date time.Time) ([]AppointmentReminderRow, error) {
	dateStr := date.Format("2006-01-02")
	rows, err := pool.Query(ctx, `
		SELECT a.id, p.id, COALESCE(p.full_name, ''), a.appointment_date, a.start_time, g.id, TRIM(g.phone)
		FROM appointments a
		JOIN patients p ON p.id = a.patient_id AND p.deleted_at IS NULL
		JOIN patient_guardians pg ON pg.patient_id = a.patient_id
		JOIN legal_guardians g ON g.id = pg.legal_guardian_id AND g.deleted_at IS NULL AND g.status != 'CANCELLED'
		WHERE a.appointment_date = $1::date
		  AND a.status IN ('CONFIRMED', 'PENDING_SIGNATURE')
		  AND g.phone IS NOT NULL AND TRIM(g.phone) != ''
		ORDER BY a.start_time, g.full_name
	`, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AppointmentReminderRow
	for rows.Next() {
		var r AppointmentReminderRow
		if err := rows.Scan(&r.AppointmentID, &r.PatientID, &r.PatientName, &r.AppointmentDate, &r.StartTime, &r.GuardianID, &r.GuardianPhone); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
