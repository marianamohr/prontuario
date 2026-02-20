package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LegalGuardian struct {
	ID            uuid.UUID
	Email         string
	GoogleSub     *string
	PasswordHash  *string
	FullName      string
	CPFEncrypted  []byte
	CPFNonce      []byte
	CPFKeyVersion *string
	CPFHash       *string
	AddressID     *uuid.UUID
	BirthDate     *string
	Phone         *string
	AuthProvider  string
	Status        string
}

func LegalGuardianByEmail(ctx context.Context, db *gorm.DB, email string) (*LegalGuardian, error) {
	var g LegalGuardian
	err := db.WithContext(ctx).Raw(`
		SELECT id, email, google_sub, password_hash, full_name,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date::text, phone, auth_provider::text, status
		FROM legal_guardians WHERE email = ? AND status != 'CANCELLED' AND deleted_at IS NULL
	`, email).Scan(&g).Error
	if err != nil {
		return nil, err
	}
	if g.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &g, nil
}

func LegalGuardianByGoogleSub(ctx context.Context, db *gorm.DB, sub string) (*LegalGuardian, error) {
	var g LegalGuardian
	err := db.WithContext(ctx).Raw(`
		SELECT id, email, google_sub, password_hash, full_name,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date::text, phone, auth_provider::text, status
		FROM legal_guardians WHERE google_sub = ? AND status != 'CANCELLED' AND deleted_at IS NULL
	`, sub).Scan(&g).Error
	if err != nil {
		return nil, err
	}
	if g.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &g, nil
}

func LegalGuardianByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*LegalGuardian, error) {
	var g LegalGuardian
	err := db.WithContext(ctx).Raw(`
		SELECT id, email, google_sub, password_hash, full_name,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date::text, phone, auth_provider::text, status
		FROM legal_guardians WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&g).Error
	if err != nil {
		return nil, err
	}
	if g.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &g, nil
}

func CreateLegalGuardian(ctx context.Context, db *gorm.DB, g *LegalGuardian) error {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		INSERT INTO legal_guardians (email, google_sub, password_hash, full_name, cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date, phone, auth_provider, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::auth_provider_enum, ?)
		RETURNING id
	`, g.Email, g.GoogleSub, g.PasswordHash, g.FullName, g.CPFEncrypted, g.CPFNonce, g.CPFKeyVersion, g.CPFHash, g.AddressID, g.BirthDate, g.Phone, g.AuthProvider, g.Status).Scan(&res).Error
	if err != nil {
		return err
	}
	g.ID = res.ID
	return nil
}

func UpdateLegalGuardianCPF(ctx context.Context, db *gorm.DB, id uuid.UUID, cpfEncrypted, cpfNonce []byte, cpfKeyVersion string, cpfHash string) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE legal_guardians
		SET cpf_encrypted = ?,
		    cpf_nonce = ?,
		    cpf_key_version = ?::text,
		    cpf_hash = ?::text,
		    updated_at = now()
		WHERE id = ? AND deleted_at IS NULL
	`, cpfEncrypted, cpfNonce, cpfKeyVersion, cpfHash, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GuardianInfo is a minimal view for listing guardians of a patient.
type GuardianInfo struct {
	ID       uuid.UUID
	FullName string
	Email    string
	Phone    *string
	Relation string
}

func GuardiansByPatient(ctx context.Context, db *gorm.DB, patientID uuid.UUID) ([]GuardianInfo, error) {
	var list []GuardianInfo
	err := db.WithContext(ctx).Raw(`
		SELECT g.id, g.full_name, g.email, g.phone, pg.relation
		FROM legal_guardians g
		JOIN patient_guardians pg ON pg.legal_guardian_id = g.id
		WHERE pg.patient_id = ? AND g.status != 'CANCELLED' AND g.deleted_at IS NULL
		ORDER BY g.full_name
	`, patientID).Scan(&list).Error
	return list, err
}

// UpdateLegalGuardian atualiza full_name, email, address_id, birth_date, phone. Se cpfHash != nil, atualiza cpf_hash.
func UpdateLegalGuardian(ctx context.Context, db *gorm.DB, id uuid.UUID, fullName, email string, addressID *uuid.UUID, birthDate, phone *string, cpfHash *string) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE legal_guardians SET full_name = ?, email = ?, address_id = ?, birth_date = ?, phone = COALESCE(?, phone), cpf_hash = COALESCE(?, cpf_hash), updated_at = now()
		WHERE id = ?
	`, fullName, email, addressID, birthDate, phone, cpfHash, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateLegalGuardianAdmin atualiza cadastro do responsável legal (backoffice).
func UpdateLegalGuardianAdmin(
	ctx context.Context,
	db *gorm.DB,
	id uuid.UUID,
	fullName *string,
	email *string,
	addressID *uuid.UUID,
	birthDate *string,
	phone *string,
	status *string,
	passwordHash *string,
	authProvider *string,
	cpfEncrypted []byte,
	cpfNonce []byte,
	cpfKeyVersion *string,
	cpfHash *string,
) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE legal_guardians
		SET
			full_name = COALESCE(?, full_name),
			email = COALESCE(?, email),
			address_id = COALESCE(?, address_id),
			birth_date = CASE WHEN ?::text IS NULL THEN birth_date ELSE NULLIF(?::text, '')::date END,
			phone = CASE WHEN ?::text IS NULL THEN phone ELSE NULLIF(?::text, '') END,
			status = COALESCE(?, status),
			password_hash = COALESCE(?, password_hash),
			auth_provider = COALESCE(?::auth_provider_enum, auth_provider),
			cpf_encrypted = CASE WHEN ?::bytea IS NULL THEN cpf_encrypted ELSE ? END,
			cpf_nonce = CASE WHEN ?::bytea IS NULL THEN cpf_nonce ELSE ? END,
			cpf_key_version = CASE WHEN ?::text IS NULL THEN cpf_key_version ELSE ?::text END,
			cpf_hash = CASE WHEN ?::text IS NULL THEN cpf_hash ELSE ?::text END,
			updated_at = now()
		WHERE id = ? AND deleted_at IS NULL
	`, fullName, email, addressID, birthDate, birthDate, phone, status, passwordHash, authProvider, cpfEncrypted, cpfEncrypted, cpfNonce, cpfNonce, cpfKeyVersion, cpfKeyVersion, cpfHash, cpfHash, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func SoftDeleteLegalGuardian(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE legal_guardians SET deleted_at = now(), updated_at = now() WHERE id = ? AND deleted_at IS NULL
	`, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
