package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Patient struct {
	ID            uuid.UUID
	ClinicID      uuid.UUID
	FullName      string
	BirthDate     *string
	Email         *string
	CPFEncrypted  []byte
	CPFNonce      []byte
	CPFKeyVersion *string
	CPFHash       *string
}

func PatientsByClinic(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) ([]Patient, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, clinic_id, full_name, birth_date::text, email,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash
		FROM patients
		WHERE clinic_id = $1 AND deleted_at IS NULL
		ORDER BY full_name
	`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Patient
	for rows.Next() {
		var p Patient
		var birth *string
		var cpfKeyVer, cpfHash *string
		if err := rows.Scan(&p.ID, &p.ClinicID, &p.FullName, &birth, &p.Email, &p.CPFEncrypted, &p.CPFNonce, &cpfKeyVer, &cpfHash); err != nil {
			return nil, err
		}
		p.BirthDate = birth
		p.CPFKeyVersion = cpfKeyVer
		p.CPFHash = cpfHash
		list = append(list, p)
	}
	return list, rows.Err()
}

func PatientByIDAndClinic(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) (*Patient, error) {
	var p Patient
	var birth *string
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, full_name, birth_date::text, email,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash
		FROM patients
		WHERE id = $1 AND clinic_id = $2 AND deleted_at IS NULL
	`, id, clinicID).Scan(&p.ID, &p.ClinicID, &p.FullName, &birth, &p.Email, &p.CPFEncrypted, &p.CPFNonce, &p.CPFKeyVersion, &p.CPFHash)
	if err != nil {
		return nil, err
	}
	p.BirthDate = birth
	return &p, nil
}

func PatientByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Patient, error) {
	var p Patient
	var birth *string
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, full_name, birth_date::text, email,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash
		FROM patients
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&p.ID, &p.ClinicID, &p.FullName, &birth, &p.Email, &p.CPFEncrypted, &p.CPFNonce, &p.CPFKeyVersion, &p.CPFHash)
	if err != nil {
		return nil, err
	}
	p.BirthDate = birth
	return &p, nil
}

func CreatePatient(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, fullName string, birthDate *string, email *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO patients (clinic_id, full_name, birth_date, email) VALUES ($1, $2, $3, $4) RETURNING id
	`, clinicID, fullName, birthDate, email).Scan(&id)
	return id, err
}

func UpdatePatient(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID, fullName string, birthDate *string, email *string) error {
	result, err := pool.Exec(ctx, `
		UPDATE patients SET full_name = $1, birth_date = $2, email = $3, updated_at = now()
		WHERE id = $4 AND clinic_id = $5 AND deleted_at IS NULL
	`, fullName, birthDate, email, id, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func SoftDeletePatient(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) error {
	result, err := pool.Exec(ctx, `
		UPDATE patients SET deleted_at = now(), updated_at = now() WHERE id = $1 AND clinic_id = $2 AND deleted_at IS NULL
	`, id, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func SetPatientCPF(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID, cpfEnc, cpfNonce []byte, cpfKeyVersion string, cpfHash string) error {
	result, err := pool.Exec(ctx, `
		UPDATE patients
		SET cpf_encrypted = $1,
		    cpf_nonce = $2,
		    cpf_key_version = $3::text,
		    cpf_hash = $4::text,
		    updated_at = now()
		WHERE id = $5 AND clinic_id = $6 AND deleted_at IS NULL
	`, cpfEnc, cpfNonce, cpfKeyVersion, cpfHash, id, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func ClearPatientCPF(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) error {
	result, err := pool.Exec(ctx, `
		UPDATE patients
		SET cpf_encrypted = NULL,
		    cpf_nonce = NULL,
		    cpf_key_version = NULL,
		    cpf_hash = NULL,
		    updated_at = now()
		WHERE id = $1 AND clinic_id = $2 AND deleted_at IS NULL
	`, id, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
