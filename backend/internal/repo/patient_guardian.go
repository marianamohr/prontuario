package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PatientGuardian struct {
	ID                  uuid.UUID
	PatientID           uuid.UUID
	LegalGuardianID     uuid.UUID
	Relation            string
	CanViewMedicalRecord bool
	CanViewContracts    bool
}

func PatientGuardianByPatientAndGuardian(ctx context.Context, pool *pgxpool.Pool, patientID, guardianID uuid.UUID) (*PatientGuardian, error) {
	var pg PatientGuardian
	err := pool.QueryRow(ctx, `
		SELECT id, patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts
		FROM patient_guardians WHERE patient_id = $1 AND legal_guardian_id = $2
	`, patientID, guardianID).Scan(&pg.ID, &pg.PatientID, &pg.LegalGuardianID, &pg.Relation, &pg.CanViewMedicalRecord, &pg.CanViewContracts)
	if err != nil {
		return nil, err
	}
	return &pg, nil
}

func GuardianCanViewMedicalRecord(ctx context.Context, pool *pgxpool.Pool, guardianID, patientID uuid.UUID) (bool, error) {
	var can bool
	err := pool.QueryRow(ctx, `
		SELECT can_view_medical_record FROM patient_guardians WHERE legal_guardian_id = $1 AND patient_id = $2
	`, guardianID, patientID).Scan(&can)
	if err != nil {
		return false, err
	}
	return can, nil
}

func GuardianCanViewContracts(ctx context.Context, pool *pgxpool.Pool, guardianID, patientID uuid.UUID) (bool, error) {
	var can bool
	err := pool.QueryRow(ctx, `
		SELECT can_view_contracts FROM patient_guardians WHERE legal_guardian_id = $1 AND patient_id = $2
	`, guardianID, patientID).Scan(&can)
	if err != nil {
		return false, err
	}
	return can, nil
}

func CreatePatientGuardian(ctx context.Context, pool *pgxpool.Pool, patientID, legalGuardianID uuid.UUID, relation string, canViewMedicalRecord, canViewContracts bool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO patient_guardians (patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts)
		VALUES ($1, $2, $3, $4, $5)
	`, patientID, legalGuardianID, relation, canViewMedicalRecord, canViewContracts)
	return err
}
