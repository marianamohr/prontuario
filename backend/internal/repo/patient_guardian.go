package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PatientGuardian struct {
	ID                   uuid.UUID
	PatientID            uuid.UUID
	LegalGuardianID      uuid.UUID
	Relation             string
	CanViewMedicalRecord bool
	CanViewContracts     bool
}

func PatientGuardianByPatientAndGuardian(ctx context.Context, db *gorm.DB, patientID, guardianID uuid.UUID) (*PatientGuardian, error) {
	var pg PatientGuardian
	err := db.WithContext(ctx).Raw(`
		SELECT id, patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts
		FROM patient_guardians WHERE patient_id = ? AND legal_guardian_id = ?
	`, patientID, guardianID).Scan(&pg).Error
	if err != nil {
		return nil, err
	}
	if pg.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &pg, nil
}

func GuardianCanViewMedicalRecord(ctx context.Context, db *gorm.DB, guardianID, patientID uuid.UUID) (bool, error) {
	var res struct{ CanViewMedicalRecord bool }
	err := db.WithContext(ctx).Raw(`
		SELECT can_view_medical_record FROM patient_guardians WHERE legal_guardian_id = ? AND patient_id = ?
	`, guardianID, patientID).Scan(&res).Error
	return res.CanViewMedicalRecord, err
}

func GuardianCanViewContracts(ctx context.Context, db *gorm.DB, guardianID, patientID uuid.UUID) (bool, error) {
	var res struct{ CanViewContracts bool }
	err := db.WithContext(ctx).Raw(`
		SELECT can_view_contracts FROM patient_guardians WHERE legal_guardian_id = ? AND patient_id = ?
	`, guardianID, patientID).Scan(&res).Error
	return res.CanViewContracts, err
}

func CreatePatientGuardian(ctx context.Context, db *gorm.DB, patientID, legalGuardianID uuid.UUID, relation string, canViewMedicalRecord, canViewContracts bool) error {
	return db.WithContext(ctx).Exec(`
		INSERT INTO patient_guardians (patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts)
		VALUES (?, ?, ?, ?, ?)
	`, patientID, legalGuardianID, relation, canViewMedicalRecord, canViewContracts).Error
}
