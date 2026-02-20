package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/seed"
	"github.com/prontuario/backend/internal/testutil"
)

// TestTenantIsolation exige DATABASE_URL. Rode: go test -v -run TestTenantIsolation ./internal/repo
func TestTenantIsolation(t *testing.T) {
	ctx := context.Background()
	db, _ := testutil.OpenDB(ctx)
	if db == nil {
		t.Skip("DATABASE_URL not set")
		return
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	if err := testutil.MustMigrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Run(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var clinicA, clinicB uuid.UUID
	if err := db.WithContext(ctx).Raw("SELECT id FROM clinics LIMIT 1").Scan(&clinicA).Error; err != nil {
		t.Fatalf("need clinics (run seed): %v", err)
	}
	if err := db.WithContext(ctx).Raw("SELECT id FROM clinics OFFSET 1 LIMIT 1").Scan(&clinicB).Error; err != nil {
		t.Fatal("need at least 2 clinics from seed")
	}
	if clinicA == clinicB {
		t.Fatal("need at least 2 distinct clinics from seed")
	}

	patientA, err := CreatePatient(ctx, db, clinicA, "Paciente Isolamento Test", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreatePatient: %v", err)
	}
	listB, err := PatientsByClinic(ctx, db, clinicB)
	if err != nil {
		t.Fatalf("PatientsByClinic B: %v", err)
	}
	for _, p := range listB {
		if p.ID == patientA {
			t.Errorf("patient from clinic A must not appear when listing clinic B (tenant isolation)")
		}
	}
	listA, err := PatientsByClinic(ctx, db, clinicA)
	if err != nil {
		t.Fatalf("PatientsByClinic A: %v", err)
	}
	found := false
	for _, p := range listA {
		if p.ID == patientA {
			found = true
			break
		}
	}
	if !found {
		t.Error("patient must appear when listing its own clinic")
	}
}
