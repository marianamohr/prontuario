//go:build integration

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/seed"
	"github.com/prontuario/backend/internal/testutil"
	"gorm.io/gorm"
)

func openDBForScheduleTest(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	db, _ := testutil.OpenDB(ctx)
	if db == nil {
		t.Skip("DATABASE_URL not set")
		return nil
	}
	if err := testutil.MustMigrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Run(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func TestIntegration_UpdateAppointmentsStatusByContract(t *testing.T) {
	ctx := context.Background()
	db := openDBForScheduleTest(t)
	if db == nil {
		return
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	var row struct {
		ClinicID    uuid.UUID
		ProfID      uuid.UUID
		PatientID   uuid.UUID
	}
	if err := db.WithContext(ctx).Raw(`SELECT c.id as clinic_id, p.id as prof_id, pt.id as patient_id FROM clinics c
		JOIN professionals p ON p.clinic_id = c.id
		JOIN patients pt ON pt.clinic_id = c.id
		LIMIT 1`).Scan(&row).Error; err != nil || row.ClinicID == uuid.Nil {
		t.Skipf("seed has no data: %v", err)
		return
	}
	clinicID, profID, patientID := row.ClinicID, row.ProfID, row.PatientID

	var guardianID uuid.UUID
	_ = db.WithContext(ctx).Raw("SELECT legal_guardian_id FROM patient_guardians WHERE patient_id = ? LIMIT 1", patientID).Scan(&guardianID)
	if guardianID == uuid.Nil {
		t.Skip("patient sem guardian")
		return
	}

	var templateID uuid.UUID
	_ = db.WithContext(ctx).Raw("SELECT id FROM contract_templates WHERE clinic_id = ? LIMIT 1", clinicID).Scan(&templateID)
	if templateID == uuid.Nil {
		var err error
		templateID, err = CreateContractTemplate(ctx, db, clinicID, &profID, "Tpl Test", "<p>x</p>", "", "")
		if err != nil {
			t.Fatalf("CreateContractTemplate: %v", err)
		}
	}
	contractID, err := CreateContract(ctx, db, clinicID, patientID, guardianID, &profID, templateID, "Guardian", false, 1, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateContract: %v", err)
	}

	startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	endTime := startTime.Add(50 * time.Minute)
	for _, d := range []time.Time{
		time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 4, 8, 0, 0, 0, 0, time.UTC),
	} {
		_, err = CreateAppointment(ctx, db, clinicID, profID, patientID, &contractID, d, startTime, endTime, "PRE_AGENDADO", "")
		if err != nil {
			t.Fatalf("CreateAppointment: %v", err)
		}
	}

	n, err := UpdateAppointmentsStatusByContract(ctx, db, contractID, "AGENDADO")
	if err != nil {
		t.Fatalf("UpdateAppointmentsStatusByContract: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows updated, got %d", n)
	}

	var count int
	err = db.WithContext(ctx).Raw("SELECT COUNT(*) FROM appointments WHERE contract_id = ? AND status = 'AGENDADO'", contractID).Scan(&count).Error
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 appointments with AGENDADO, got %d", count)
	}
}
