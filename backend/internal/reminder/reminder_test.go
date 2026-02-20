package reminder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/repo"
	"gorm.io/gorm"
)

func TestSendAppointmentReminders_DBNil(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	sent, skipped := SendAppointmentReminders(ctx, nil, date, nil)
	if sent != 0 || skipped != 0 {
		t.Errorf("db nil: got sent=%d skipped=%d, want 0,0", sent, skipped)
	}
}

func TestSendAppointmentRemindersWithLister_DBAndListerNil(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	sent, skipped := SendAppointmentRemindersWithLister(ctx, nil, date, nil, nil)
	if sent != 0 || skipped != 0 {
		t.Errorf("db and lister nil: got sent=%d skipped=%d, want 0,0", sent, skipped)
	}
}

func TestSendAppointmentRemindersWithLister_ListerReturnsError(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	errMock := errors.New("db error")
	lister := &mockLister{err: errMock}
	sender := &mockSender{}
	sent, skipped := SendAppointmentRemindersWithLister(ctx, nil, date, sender, lister)
	if sent != 0 || skipped != 0 {
		t.Errorf("lister error: got sent=%d skipped=%d, want 0,0", sent, skipped)
	}
}

func TestSendAppointmentRemindersWithLister_SenderNil_CountsSkipped(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	rows := []repo.AppointmentReminderRow{
		{AppointmentID: uuid.New(), PatientName: "Maria", GuardianPhone: "+5511999990000", StartTime: "10:00:00"},
		{AppointmentID: uuid.New(), PatientName: "João", GuardianPhone: "+5511888880000", StartTime: "11:00:00"},
	}
	lister := &mockLister{rows: rows}
	sent, skipped := SendAppointmentRemindersWithLister(ctx, nil, date, nil, lister)
	if sent != 0 || skipped != 2 {
		t.Errorf("sender nil: got sent=%d skipped=%d, want 0,2", sent, skipped)
	}
}

func TestSendAppointmentRemindersWithLister_AllSent(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	rows := []repo.AppointmentReminderRow{
		{AppointmentID: uuid.New(), PatientID: uuid.New(), PatientName: "Maria", GuardianID: uuid.New(), GuardianPhone: "+5511999990000", StartTime: "14:30:00"},
		{AppointmentID: uuid.New(), PatientID: uuid.New(), PatientName: "João", GuardianID: uuid.New(), GuardianPhone: "+5511888880000", StartTime: "09:00:00"},
	}
	lister := &mockLister{rows: rows}
	sender := &mockSender{failIndex: -1} // nenhuma falha
	sent, skipped := SendAppointmentRemindersWithLister(ctx, nil, date, sender, lister)
	if sent != 2 || skipped != 0 {
		t.Errorf("all sent: got sent=%d skipped=%d, want 2,0", sent, skipped)
	}
	if len(sender.calls) != 2 {
		t.Errorf("sender calls: got %d, want 2", len(sender.calls))
	}
	// Formato da data no reminder: 02/01/2006
	wantDateStr := "12/02/2025"
	for i, c := range sender.calls {
		if c.dateStr != wantDateStr {
			t.Errorf("call %d dateStr: got %q, want %q", i, c.dateStr, wantDateStr)
		}
		if c.patientName != rows[i].PatientName || c.phone != rows[i].GuardianPhone {
			t.Errorf("call %d: phone=%q patient=%q", i, c.phone, c.patientName)
		}
	}
}

func TestSendAppointmentRemindersWithLister_PartialFail(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	rows := []repo.AppointmentReminderRow{
		{AppointmentID: uuid.New(), PatientName: "Maria", GuardianPhone: "+5511999990000", StartTime: "10:00:00"},
		{AppointmentID: uuid.New(), PatientName: "João", GuardianPhone: "+5511888880000", StartTime: "11:00:00"},
		{AppointmentID: uuid.New(), PatientName: "Pedro", GuardianPhone: "+5511777770000", StartTime: "12:00:00"},
	}
	lister := &mockLister{rows: rows}
	// Falha na segunda chamada (índice 1)
	sender := &mockSender{failIndex: 1}
	sent, skipped := SendAppointmentRemindersWithLister(ctx, nil, date, sender, lister)
	if sent != 2 || skipped != 1 {
		t.Errorf("partial fail: got sent=%d skipped=%d, want 2,1", sent, skipped)
	}
}

func TestDefaultWhatsAppSender_NilWhenEmpty(t *testing.T) {
	if DefaultWhatsAppSender("", "token", "from") != nil {
		t.Error("expected nil when accountSid empty")
	}
	if DefaultWhatsAppSender("sid", "", "from") != nil {
		t.Error("expected nil when authToken empty")
	}
	if DefaultWhatsAppSender("sid", "token", "") != nil {
		t.Error("expected nil when from empty")
	}
}

func TestDefaultWhatsAppSender_NonNilWhenConfigured(t *testing.T) {
	c := DefaultWhatsAppSender("sid", "token", "whatsapp:+15551234567")
	if c == nil {
		t.Error("expected non-nil client when all params set")
	}
}

// mockLister implementa AppointmentLister para testes.
type mockLister struct {
	rows []repo.AppointmentReminderRow
	err  error
}

func (m *mockLister) ListAppointmentsForReminder(_ context.Context, _ *gorm.DB, _ time.Time) ([]repo.AppointmentReminderRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

// mockSender implementa WhatsAppSender e grava as chamadas.
type mockSender struct {
	calls     []sendCall
	failIndex int // índice da chamada que deve falhar (-1 = nenhuma)
}

type sendCall struct {
	phone, patientName, dateStr, timeStr string
}

func (m *mockSender) SendReminder(phone, patientName, dateStr, timeStr string) error {
	m.calls = append(m.calls, sendCall{phone, patientName, dateStr, timeStr})
	if m.failIndex >= 0 && len(m.calls)-1 == m.failIndex {
		return errors.New("mock send error")
	}
	return nil
}
