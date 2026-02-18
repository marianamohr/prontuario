package reminder

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/reminder/pkg/repo"
	"github.com/prontuario/reminder/pkg/whatsapp"
)

const auditActionReminderSent = "APPOINTMENT_REMINDER_SENT"
const auditSourceSystem = "SYSTEM"

// WhatsAppSender sends a reminder to a phone number.
// rescheduleURL: link para confirmar/remarcar; pode ser vazio.
type WhatsAppSender interface {
	SendReminder(phone, patientName, dateStr, timeStr, rescheduleURL string) error
}

// AppointmentLister returns appointments for reminder on a given date.
type AppointmentLister interface {
	ListAppointmentsForReminder(ctx context.Context, pool *pgxpool.Pool, date time.Time, professionalID *uuid.UUID) ([]repo.AppointmentReminderRow, error)
}

// SendAppointmentReminders loads appointments for the given date (tomorrow in practice), then sends
// one WhatsApp reminder per (appointment, guardian with phone).
// appPublicURL: base URL do frontend para links de remarcação (ex: https://app.example.com).
// If professionalID is non-nil, only that professional's appointments are sent.
func SendAppointmentReminders(ctx context.Context, pool *pgxpool.Pool, date time.Time, sender WhatsAppSender, professionalID *uuid.UUID, appPublicURL string) (sent int, skipped int) {
	return SendAppointmentRemindersWithLister(ctx, pool, date, sender, nil, professionalID, appPublicURL)
}

// SendAppointmentRemindersWithLister is like SendAppointmentReminders but accepts an optional lister for tests.
func SendAppointmentRemindersWithLister(ctx context.Context, pool *pgxpool.Pool, date time.Time, sender WhatsAppSender, lister AppointmentLister, professionalID *uuid.UUID, appPublicURL string) (sent int, skipped int) {
	if pool == nil && lister == nil {
		log.Printf("[reminder] pool is nil and no lister, skipping")
		return 0, 0
	}
	var rows []repo.AppointmentReminderRow
	var err error
	if lister != nil {
		rows, err = lister.ListAppointmentsForReminder(ctx, pool, date, professionalID)
	} else {
		rows, err = repo.ListAppointmentsForReminder(ctx, pool, date, professionalID)
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
		var rescheduleURL string
		if pool != nil && appPublicURL != "" {
			if token, err := repo.CreateReminderToken(ctx, pool, r.AppointmentID, r.GuardianID); err == nil {
				rescheduleURL = strings.TrimSuffix(appPublicURL, "/") + "/remarcar?token=" + token
			}
		}
		if err := sender.SendReminder(r.GuardianPhone, r.PatientName, dateStr, timeStr, rescheduleURL); err != nil {
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
