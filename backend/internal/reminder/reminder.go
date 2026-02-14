package reminder

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/repo"
	"github.com/prontuario/backend/internal/whatsapp"
)

const auditActionReminderSent = "APPOINTMENT_REMINDER_SENT"
const auditSourceSystem = "SYSTEM"

// WhatsAppSender sends a reminder to a phone number.
type WhatsAppSender interface {
	SendReminder(phone, patientName, dateStr, timeStr string) error
}

// AppointmentLister returns appointments for reminder on a given date. Used in tests with a mock; in production pass nil to use repo.
type AppointmentLister interface {
	ListAppointmentsForReminder(ctx context.Context, pool *pgxpool.Pool, date time.Time) ([]repo.AppointmentReminderRow, error)
}

// SendAppointmentReminders loads appointments for the given date (tomorrow in practice), then sends
// one WhatsApp reminder per (appointment, guardian with phone). Failures per recipient are logged
// and do not stop the rest. Uses repo.ListAppointmentsForReminder when lister is nil.
func SendAppointmentReminders(ctx context.Context, pool *pgxpool.Pool, date time.Time, sender WhatsAppSender) (sent int, skipped int) {
	return SendAppointmentRemindersWithLister(ctx, pool, date, sender, nil)
}

// SendAppointmentRemindersWithLister is like SendAppointmentReminders but accepts an optional lister for tests. If lister is nil, repo is used (and pool must be non-nil).
func SendAppointmentRemindersWithLister(ctx context.Context, pool *pgxpool.Pool, date time.Time, sender WhatsAppSender, lister AppointmentLister) (sent int, skipped int) {
	if pool == nil && lister == nil {
		log.Printf("[reminder] pool is nil and no lister, skipping")
		return 0, 0
	}
	var rows []repo.AppointmentReminderRow
	var err error
	if lister != nil {
		rows, err = lister.ListAppointmentsForReminder(ctx, pool, date)
	} else {
		rows, err = repo.ListAppointmentsForReminder(ctx, pool, date)
	}
	if err != nil {
		log.Printf("[reminder] ListAppointmentsForReminder: %v", err)
		return 0, 0
	}
	if sender == nil {
		log.Printf("[reminder] WhatsApp not configured, would send %d reminders", len(rows))
		return 0, len(rows)
	}
	dateStr := date.Format("02/01/2006")
	for _, r := range rows {
		timeStr := r.StartTime.Format("15:04")
		if err := sender.SendReminder(r.GuardianPhone, r.PatientName, dateStr, timeStr); err != nil {
			log.Printf("[reminder] send failed appointment=%s guardian=%s phone=%s: %v", r.AppointmentID, r.GuardianID, r.GuardianPhone, err)
			skipped++
			continue
		}
		sent++
		log.Printf("[reminder] sent appointment=%s to %s", r.AppointmentID, r.GuardianPhone)
		if pool != nil {
			_ = repo.CreateAuditEventFull(ctx, pool, repo.AuditEvent{
				Action:       auditActionReminderSent,
				ActorType:    auditSourceSystem,
				ResourceType: strPtr("APPOINTMENT"),
				ResourceID:   &r.AppointmentID,
				PatientID:    &r.PatientID,
				Source:       strPtr(auditSourceSystem),
				Metadata:     map[string]string{"appointment_id": r.AppointmentID.String(), "guardian_id": r.GuardianID.String()},
			})
		}
	}
	return sent, skipped
}

func strPtr(s string) *string { return &s }

// DefaultWhatsAppSender returns a whatsapp.Client from the given config, or nil if not configured.
func DefaultWhatsAppSender(accountSid, authToken, from string) WhatsAppSender {
	if accountSid == "" || authToken == "" || from == "" {
		return nil
	}
	return whatsapp.NewClient(whatsapp.Config{
		AccountSid: accountSid,
		AuthToken:  authToken,
		From:       from,
	})
}
