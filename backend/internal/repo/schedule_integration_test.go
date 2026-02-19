//go:build integration

package repo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func openPoolForScheduleTest(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
		return nil
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	return pool
}

func TestIntegration_UpdateAppointmentsStatusByContract(t *testing.T) {
	ctx := context.Background()
	pool := openPoolForScheduleTest(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	// Get existing clinic, professional and patient (seed)
	var clinicID, profID, patientID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT c.id, p.id, pt.id FROM clinics c
		JOIN professionals p ON p.clinic_id = c.id
		JOIN patients pt ON pt.clinic_id = c.id
		LIMIT 1`).Scan(&clinicID, &profID, &patientID)
	if err != nil {
		t.Skipf("seed has no data: %v", err)
		return
	}
	guardianID := uuid.Nil
	_ = pool.QueryRow(ctx, "SELECT legal_guardian_id FROM patient_guardians WHERE patient_id = $1 LIMIT 1", patientID).Scan(&guardianID)
	if guardianID == uuid.Nil {
		t.Skip("patient sem guardian")
		return
	}

	// Create contract (and template if missing)
	var templateID uuid.UUID
	_ = pool.QueryRow(ctx, "SELECT id FROM contract_templates WHERE clinic_id = $1 LIMIT 1", clinicID).Scan(&templateID)
	if templateID == uuid.Nil {
		templateID, err = CreateContractTemplate(ctx, pool, clinicID, &profID, "Tpl Test", "<p>x</p>", "", "")
		if err != nil {
			t.Fatalf("CreateContractTemplate: %v", err)
		}
	}
	contractID, err := CreateContract(ctx, pool, clinicID, patientID, guardianID, &profID, templateID, "Guardian", false, 1, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateContract: %v", err)
	}

	// Criar dois appointments PRE_AGENDADO vinculados ao contrato
	startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	endTime := startTime.Add(50 * time.Minute)
	for _, d := range []time.Time{
		time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 4, 8, 0, 0, 0, 0, time.UTC),
	} {
		_, err = CreateAppointment(ctx, pool, clinicID, profID, patientID, &contractID, d, startTime, endTime, "PRE_AGENDADO", "")
		if err != nil {
			t.Fatalf("CreateAppointment: %v", err)
		}
	}

	n, err := UpdateAppointmentsStatusByContract(ctx, pool, contractID, "AGENDADO")
	if err != nil {
		t.Fatalf("UpdateAppointmentsStatusByContract: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows updated, got %d", n)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM appointments WHERE contract_id = $1 AND status = 'AGENDADO'", contractID).Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 appointments with AGENDADO, got %d", count)
	}
}
