package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Patient struct {
	ID            uuid.UUID
	ClinicID      uuid.UUID
	FullName      string
	BirthDate     *string
	Email         *string
	AddressID     *uuid.UUID
	CPFEncrypted  []byte
	CPFNonce      []byte
	CPFKeyVersion *string
	CPFHash       *string
}

func PatientsByClinic(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) ([]Patient, error) {
	return PatientsByClinicPaginated(ctx, db, clinicID, 0, 0)
}

// PatientsByClinicPaginated returns patients for the clinic with limit and offset. If limit is 0, no limit is applied (all rows).
func PatientsByClinicPaginated(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, limit, offset int) ([]Patient, error) {
	q := `
		SELECT id, clinic_id, full_name, birth_date::text, email, address_id,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash
		FROM patients
		WHERE clinic_id = ? AND deleted_at IS NULL
		ORDER BY full_name
	`
	args := []interface{}{clinicID}
	if limit > 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	var list []Patient
	err := db.WithContext(ctx).Raw(q, args...).Scan(&list).Error
	return list, err
}

// PatientsCountByClinic returns the total number of patients for the clinic.
func PatientsCountByClinic(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) (int, error) {
	var n int
	err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM patients WHERE clinic_id = ? AND deleted_at IS NULL`, clinicID).Scan(&n).Error
	return n, err
}

func PatientByIDAndClinic(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) (*Patient, error) {
	var p Patient
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, full_name, birth_date::text, email, address_id,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash
		FROM patients
		WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, id, clinicID).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func PatientByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Patient, error) {
	var p Patient
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, full_name, birth_date::text, email, address_id,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash
		FROM patients
		WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func CreatePatient(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, fullName string, birthDate *string, email *string, addressID *uuid.UUID) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		INSERT INTO patients (clinic_id, full_name, birth_date, email, address_id) VALUES (?, ?, ?, ?, ?) RETURNING id
	`, clinicID, fullName, birthDate, email, addressID).Scan(&res).Error
	return res.ID, err
}

func UpdatePatient(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID, fullName string, birthDate *string, email *string, addressID *uuid.UUID) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE patients SET full_name = ?, birth_date = ?, email = ?, address_id = ?, updated_at = now()
		WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, fullName, birthDate, email, addressID, id, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func SoftDeletePatient(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE patients SET deleted_at = now(), updated_at = now() WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, id, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func SetPatientCPF(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID, cpfEnc, cpfNonce []byte, cpfKeyVersion string, cpfHash string) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE patients
		SET cpf_encrypted = ?,
		    cpf_nonce = ?,
		    cpf_key_version = ?::text,
		    cpf_hash = ?::text,
		    updated_at = now()
		WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, cpfEnc, cpfNonce, cpfKeyVersion, cpfHash, id, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ClearPatientCPF(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE patients
		SET cpf_encrypted = NULL,
		    cpf_nonce = NULL,
		    cpf_key_version = NULL,
		    cpf_hash = NULL,
		    updated_at = now()
		WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, id, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
