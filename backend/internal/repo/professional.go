package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func ProfessionalByEmail(ctx context.Context, db *gorm.DB, email string) (*Professional, error) {
	var p Professional
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, email, password_hash, full_name, trade_name, status, signature_image_data
		FROM professionals WHERE lower(email) = lower(?) AND status != 'CANCELLED'
	`, email).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func ProfessionalByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Professional, error) {
	var p Professional
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, email, password_hash, full_name, trade_name, status, signature_image_data
		FROM professionals WHERE id = ?
	`, id).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
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

func ProfessionalProfileByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*ProfessionalProfile, error) {
	var p ProfessionalProfile
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, email, full_name, trade_name, birth_date::text, address_id, marital_status
		FROM professionals WHERE id = ?
	`, id).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func UpdateProfessionalProfile(ctx context.Context, db *gorm.DB, id uuid.UUID, fullName string, tradeName *string, birthDate *string, addressID *uuid.UUID, maritalStatus *string) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE professionals
		SET full_name = ?,
		    trade_name = CASE WHEN ?::text IS NULL THEN NULL ELSE NULLIF(?::text, '') END,
		    birth_date = CASE WHEN ?::text IS NULL THEN birth_date ELSE NULLIF(?::text, '')::date END,
		    address_id = ?,
		    marital_status = CASE WHEN ?::text IS NULL THEN marital_status ELSE NULLIF(?::text, '') END,
		    updated_at = now()
		WHERE id = ?
	`, fullName, tradeName, tradeName, birthDate, birthDate, addressID, maritalStatus, maritalStatus, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ProfessionalAdminByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*ProfessionalAdminView, error) {
	var p ProfessionalAdminView
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, email, full_name, trade_name, status, signature_image_data,
		       birth_date::text, cpf_hash, cpf_encrypted, cpf_nonce, cpf_key_version, address_id, marital_status
		FROM professionals WHERE id = ?
	`, id).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func UpdateProfessionalSignature(ctx context.Context, db *gorm.DB, professionalID uuid.UUID, signatureImageData *string) error {
	return db.WithContext(ctx).Exec(`UPDATE professionals SET signature_image_data = ?, updated_at = now() WHERE id = ?`, signatureImageData, professionalID).Error
}

// UpdateProfessionalAdmin atualiza dados do profissional (backoffice).
func UpdateProfessionalAdmin(
	ctx context.Context,
	db *gorm.DB,
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
	result := db.WithContext(ctx).Exec(`
		UPDATE professionals
		SET
			email = COALESCE(?, email),
			full_name = COALESCE(?, full_name),
			trade_name = CASE WHEN ?::text IS NULL THEN trade_name ELSE NULLIF(?::text, '') END,
			clinic_id = COALESCE(?, clinic_id),
			status = COALESCE(?, status),
			birth_date = CASE WHEN ?::text IS NULL THEN birth_date ELSE NULLIF(?::text, '')::date END,
			address_id = COALESCE(?, address_id),
			marital_status = CASE WHEN ?::text IS NULL THEN marital_status ELSE NULLIF(?::text, '') END,
			cpf_hash = CASE WHEN ?::text IS NULL THEN cpf_hash ELSE NULLIF(?::text, '') END,
			cpf_encrypted = CASE WHEN ?::bytea IS NULL THEN cpf_encrypted ELSE ? END,
			cpf_nonce = CASE WHEN ?::bytea IS NULL THEN cpf_nonce ELSE ? END,
			cpf_key_version = CASE WHEN ?::text IS NULL THEN cpf_key_version ELSE ?::text END,
			password_hash = COALESCE(?, password_hash),
			signature_image_data = CASE WHEN ?::text IS NULL THEN signature_image_data ELSE NULLIF(?::text, '') END,
			updated_at = now()
		WHERE id = ?
	`, email, fullName, tradeName, tradeName, clinicID, status, birthDate, addressID, maritalStatus, maritalStatus, cpfHash, cpfHash, cpfEncrypted, cpfEncrypted, cpfNonce, cpfNonce, cpfKeyVersion, cpfKeyVersion, passwordHash, signatureImageData, signatureImageData, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ProfessionalByIDAndClinic(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) (*Professional, error) {
	var p Professional
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, email, password_hash, full_name, status, signature_image_data
		FROM professionals WHERE id = ? AND clinic_id = ?
	`, id, clinicID).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}
