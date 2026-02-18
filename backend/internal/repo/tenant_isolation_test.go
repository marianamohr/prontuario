package repo

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestTenantIsolation exige DATABASE_URL. Rode: go test -v -run TestTenantIsolation ./internal/repo
func TestTenantIsolation(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	ctx := context.Background()

	var clinicA, clinicB uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT id FROM clinics LIMIT 1").Scan(&clinicA); err != nil {
		t.Fatalf("need clinics (run seed): %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT id FROM clinics OFFSET 1 LIMIT 1").Scan(&clinicB); err != nil {
		t.Fatal("need at least 2 clinics from seed")
	}
	if clinicA == clinicB {
		t.Fatal("need at least 2 distinct clinics from seed")
	}

	// Cria paciente na clínica A
	patientA, err := CreatePatient(ctx, pool, clinicA, "Paciente Isolamento Test", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreatePatient: %v", err)
	}
	listB, err := PatientsByClinic(ctx, pool, clinicB)
	if err != nil {
		t.Fatalf("PatientsByClinic B: %v", err)
	}
	for _, p := range listB {
		if p.ID == patientA {
			t.Errorf("patient from clinic A must not appear when listing clinic B (tenant isolation)")
		}
	}
	listA, err := PatientsByClinic(ctx, pool, clinicA)
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
