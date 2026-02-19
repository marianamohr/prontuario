package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateReminderToken creates a token for confirm/reschedule (valid 7 days).
func CreateReminderToken(ctx context.Context, pool *pgxpool.Pool, appointmentID, guardianID uuid.UUID) (string, error) {
	token := uuid.New().String()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err := pool.Exec(ctx, `
		INSERT INTO appointment_reminder_tokens (appointment_id, guardian_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, appointmentID, guardianID, token, expiresAt)
	return token, err
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
	ClinicID        uuid.UUID
	Status          string
}

// ListAppointmentsForReminder returns appointments on the given date with guardian phone.
// If professionalID is non-nil, filters by that professional only.
// Only status AGENDADO and CONFIRMADO; only guardians with non-empty phone.
func ListAppointmentsForReminder(ctx context.Context, pool *pgxpool.Pool, date time.Time, professionalID *uuid.UUID) ([]AppointmentReminderRow, error) {
	dateStr := date.Format("2006-01-02")
	query := `
		SELECT a.id, p.id, COALESCE(p.full_name, ''), a.appointment_date, a.start_time, g.id, TRIM(g.phone), a.clinic_id, a.status
		FROM appointments a
		JOIN patients p ON p.id = a.patient_id AND p.deleted_at IS NULL
		JOIN patient_guardians pg ON pg.patient_id = a.patient_id
		JOIN legal_guardians g ON g.id = pg.legal_guardian_id AND g.deleted_at IS NULL AND g.status != 'CANCELLED'
		WHERE a.appointment_date = $1::date
		  AND a.status IN ('AGENDADO', 'CONFIRMADO')
		  AND g.phone IS NOT NULL AND TRIM(g.phone) != ''
	`
	args := []interface{}{dateStr}
	if professionalID != nil {
		query += ` AND a.professional_id = $2`
		args = append(args, professionalID)
	}
	query += ` ORDER BY a.start_time, g.full_name`
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AppointmentReminderRow
	for rows.Next() {
		var r AppointmentReminderRow
		if err := rows.Scan(&r.AppointmentID, &r.PatientID, &r.PatientName, &r.AppointmentDate, &r.StartTime, &r.GuardianID, &r.GuardianPhone, &r.ClinicID, &r.Status); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// ConfirmAppointment sets appointment status to CONFIRMADO if it is currently AGENDADO.
func ConfirmAppointment(ctx context.Context, pool *pgxpool.Pool, appointmentID, clinicID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE appointments SET status = 'CONFIRMADO', updated_at = now()
		WHERE id = $1 AND clinic_id = $2 AND status = 'AGENDADO'
	`, appointmentID, clinicID)
	return err
}

// AuditEvent for creating audit entries.
type AuditEvent struct {
	Action       string
	ActorType    string
	ResourceType *string
	ResourceID   *uuid.UUID
	PatientID    *uuid.UUID
	Source       *string
	Metadata     interface{}
}

// CreateAuditEventFull inserts an audit event.
func CreateAuditEventFull(ctx context.Context, pool *pgxpool.Pool, ev AuditEvent) error {
	var meta []byte
	if ev.Metadata != nil {
		var err error
		meta, err = json.Marshal(ev.Metadata)
		if err != nil {
			return err
		}
	}
	source := "SYSTEM"
	if ev.Source != nil {
		source = *ev.Source
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO audit_events (
			action, actor_type, actor_id, clinic_id, request_id, ip, user_agent,
			resource_type, resource_id, patient_id, is_impersonated, impersonation_session_id,
			source, severity, metadata
		) VALUES (
			$1, $2, NULL, NULL, '', '', '',
			$3, $4, $5, false, NULL,
			$6, 'INFO', $7
		)
	`,
		ev.Action, ev.ActorType,
		ev.ResourceType, ev.ResourceID, ev.PatientID,
		source, meta,
	)
	return err
}
