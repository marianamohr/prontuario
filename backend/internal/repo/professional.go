package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Professional struct {
	ID                 uuid.UUID
	ClinicID           uuid.UUID
	Email              string
	PasswordHash       string
	FullName           string
	TradeName          *string
	Status             string
	SignatureImageData *string
}

// ProfessionalAdminView é uma visão completa do cadastro do profissional (para backoffice).
type ProfessionalAdminView struct {
	ID                 uuid.UUID
	ClinicID           uuid.UUID
	Email              string
	FullName           string
	TradeName          *string
	Status             string
	SignatureImageData *string
	BirthDate          *string
	CPFHash            *string
	CPFEncrypted       []byte
	CPFNonce           []byte
	CPFKeyVersion      *string
	AddressID          *uuid.UUID
	MaritalStatus      *string
}

func ProfessionalByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*Professional, error) {
	var p Professional
	var sig *string
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, email, password_hash, full_name, trade_name, status, signature_image_data
		FROM professionals WHERE lower(email) = lower($1) AND status != 'CANCELLED'
	`, email).Scan(&p.ID, &p.ClinicID, &p.Email, &p.PasswordHash, &p.FullName, &p.TradeName, &p.Status, &sig)
	if err != nil {
		return nil, err
	}
	p.SignatureImageData = sig
	return &p, nil
}

func ProfessionalByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Professional, error) {
	var p Professional
	var sig *string
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, email, password_hash, full_name, trade_name, status, signature_image_data
		FROM professionals WHERE id = $1
	`, id).Scan(&p.ID, &p.ClinicID, &p.Email, &p.PasswordHash, &p.FullName, &p.TradeName, &p.Status, &sig)
	if err != nil {
		return nil, err
	}
	p.SignatureImageData = sig
	return &p, nil
}

type ProfessionalProfile struct {
	ID            uuid.UUID
	ClinicID      uuid.UUID
	Email         string
	FullName      string
	TradeName     *string
	BirthDate     *string
	AddressID     *uuid.UUID
	MaritalStatus *string
}

func ProfessionalProfileByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*ProfessionalProfile, error) {
	var p ProfessionalProfile
	var tradeName, birthDate, maritalStatus *string
	var addrID *uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, email, full_name, trade_name, birth_date::text, address_id, marital_status
		FROM professionals WHERE id = $1
	`, id).Scan(&p.ID, &p.ClinicID, &p.Email, &p.FullName, &tradeName, &birthDate, &addrID, &maritalStatus)
	if err != nil {
		return nil, err
	}
	p.TradeName = tradeName
	p.BirthDate = birthDate
	p.AddressID = addrID
	p.MaritalStatus = maritalStatus
	return &p, nil
}

func UpdateProfessionalProfile(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, fullName string, tradeName *string, birthDate *string, addressID *uuid.UUID, maritalStatus *string) error {
	result, err := pool.Exec(ctx, `
		UPDATE professionals
		SET full_name = $1,
		    trade_name = CASE WHEN $2::text IS NULL THEN NULL ELSE NULLIF($2::text, '') END,
		    birth_date = CASE WHEN $3::text IS NULL THEN birth_date ELSE NULLIF($3::text, '')::date END,
		    address_id = $4,
		    marital_status = CASE WHEN $5::text IS NULL THEN marital_status ELSE NULLIF($5::text, '') END,
		    updated_at = now()
		WHERE id = $6
	`, fullName, tradeName, birthDate, addressID, maritalStatus, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func ProfessionalAdminByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*ProfessionalAdminView, error) {
	var p ProfessionalAdminView
	var sig *string
	var tradeName, birth, cpfHash, marital *string
	var addrID *uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, email, full_name, trade_name, status, signature_image_data,
		       birth_date::text, cpf_hash, cpf_encrypted, cpf_nonce, cpf_key_version, address_id, marital_status
		FROM professionals WHERE id = $1
	`, id).Scan(&p.ID, &p.ClinicID, &p.Email, &p.FullName, &tradeName, &p.Status, &sig, &birth, &cpfHash, &p.CPFEncrypted, &p.CPFNonce, &p.CPFKeyVersion, &addrID, &marital)
	if err != nil {
		return nil, err
	}
	p.SignatureImageData = sig
	p.TradeName = tradeName
	p.BirthDate = birth
	p.CPFHash = cpfHash
	p.AddressID = addrID
	p.MaritalStatus = marital
	return &p, nil
}

func UpdateProfessionalSignature(ctx context.Context, pool *pgxpool.Pool, professionalID uuid.UUID, signatureImageData *string) error {
	_, err := pool.Exec(ctx, `UPDATE professionals SET signature_image_data = $1, updated_at = now() WHERE id = $2`, signatureImageData, professionalID)
	return err
}

// UpdateProfessionalAdmin atualiza dados do profissional (backoffice).
// Campos opcionais: se nil, mantém. addressID é UUID do endereço em addresses.
func UpdateProfessionalAdmin(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	email, fullName *string,
	tradeName *string,
	clinicID *uuid.UUID,
	status *string,
	birthDate *string,
	addressID *uuid.UUID,
	maritalStatus *string,
	cpfHash *string,
	cpfEncrypted []byte,
	cpfNonce []byte,
	cpfKeyVersion *string,
	passwordHash *string,
	signatureImageData *string,
) error {
	result, err := pool.Exec(ctx, `
		UPDATE professionals
		SET
			email = COALESCE($1, email),
			full_name = COALESCE($2, full_name),
			trade_name = CASE WHEN $3::text IS NULL THEN trade_name ELSE NULLIF($3::text, '') END,
			clinic_id = COALESCE($4, clinic_id),
			status = COALESCE($5, status),
			birth_date = CASE WHEN $6::text IS NULL THEN birth_date ELSE NULLIF($6::text, '')::date END,
			address_id = COALESCE($7, address_id),
			marital_status = CASE WHEN $8::text IS NULL THEN marital_status ELSE NULLIF($8::text, '') END,
			cpf_hash = CASE WHEN $9::text IS NULL THEN cpf_hash ELSE NULLIF($9::text, '') END,
			cpf_encrypted = CASE WHEN $10::bytea IS NULL THEN cpf_encrypted ELSE $10 END,
			cpf_nonce = CASE WHEN $11::bytea IS NULL THEN cpf_nonce ELSE $11 END,
			cpf_key_version = CASE WHEN $12::text IS NULL THEN cpf_key_version ELSE $12::text END,
			password_hash = COALESCE($13, password_hash),
			signature_image_data = CASE WHEN $14::text IS NULL THEN signature_image_data ELSE NULLIF($14::text, '') END,
			updated_at = now()
		WHERE id = $15
	`, email, fullName, tradeName, clinicID, status, birthDate, addressID, maritalStatus, cpfHash, cpfEncrypted, cpfNonce, cpfKeyVersion, passwordHash, signatureImageData, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func ProfessionalByIDAndClinic(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) (*Professional, error) {
	var p Professional
	var sig *string
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, email, password_hash, full_name, status, signature_image_data
		FROM professionals WHERE id = $1 AND clinic_id = $2
	`, id, clinicID).Scan(&p.ID, &p.ClinicID, &p.Email, &p.PasswordHash, &p.FullName, &p.Status, &sig)
	if err != nil {
		return nil, err
	}
	p.SignatureImageData = sig
	return &p, nil
}
