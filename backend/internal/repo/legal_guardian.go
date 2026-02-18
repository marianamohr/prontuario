package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	Phone         *string // E.164 para WhatsApp
	AuthProvider  string
	Status        string
}

func LegalGuardianByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*LegalGuardian, error) {
	var g LegalGuardian
	var googleSub, passHash, cpfKeyVer, cpfHash *string
	var addrID *uuid.UUID
	var birth, phone *string
	err := pool.QueryRow(ctx, `
		SELECT id, email, google_sub, password_hash, full_name,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date::text, phone, auth_provider::text, status
		FROM legal_guardians WHERE email = $1 AND status != 'CANCELLED' AND deleted_at IS NULL
	`, email).Scan(&g.ID, &g.Email, &googleSub, &passHash, &g.FullName,
		&g.CPFEncrypted, &g.CPFNonce, &cpfKeyVer, &cpfHash, &addrID, &birth, &phone, &g.AuthProvider, &g.Status)
	if err != nil {
		return nil, err
	}
	g.GoogleSub = googleSub
	g.PasswordHash = passHash
	g.CPFKeyVersion = cpfKeyVer
	g.CPFHash = cpfHash
	g.AddressID = addrID
	g.BirthDate = birth
	g.Phone = phone
	return &g, nil
}

func LegalGuardianByGoogleSub(ctx context.Context, pool *pgxpool.Pool, sub string) (*LegalGuardian, error) {
	var g LegalGuardian
	var googleSub, passHash, cpfKeyVer, cpfHash *string
	var addrID *uuid.UUID
	var birth, phone *string
	err := pool.QueryRow(ctx, `
		SELECT id, email, google_sub, password_hash, full_name,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date::text, phone, auth_provider::text, status
		FROM legal_guardians WHERE google_sub = $1 AND status != 'CANCELLED' AND deleted_at IS NULL
	`, sub).Scan(&g.ID, &g.Email, &googleSub, &passHash, &g.FullName,
		&g.CPFEncrypted, &g.CPFNonce, &cpfKeyVer, &cpfHash, &addrID, &birth, &phone, &g.AuthProvider, &g.Status)
	if err != nil {
		return nil, err
	}
	g.GoogleSub = googleSub
	g.PasswordHash = passHash
	g.CPFKeyVersion = cpfKeyVer
	g.CPFHash = cpfHash
	g.AddressID = addrID
	g.BirthDate = birth
	g.Phone = phone
	return &g, nil
}

func LegalGuardianByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*LegalGuardian, error) {
	var g LegalGuardian
	var googleSub, passHash, cpfKeyVer, cpfHash *string
	var addrID *uuid.UUID
	var birth, phone *string
	err := pool.QueryRow(ctx, `
		SELECT id, email, google_sub, password_hash, full_name,
		       cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date::text, phone, auth_provider::text, status
		FROM legal_guardians WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&g.ID, &g.Email, &googleSub, &passHash, &g.FullName,
		&g.CPFEncrypted, &g.CPFNonce, &cpfKeyVer, &cpfHash, &addrID, &birth, &phone, &g.AuthProvider, &g.Status)
	if err != nil {
		return nil, err
	}
	g.GoogleSub = googleSub
	g.PasswordHash = passHash
	g.CPFKeyVersion = cpfKeyVer
	g.CPFHash = cpfHash
	g.AddressID = addrID
	g.BirthDate = birth
	g.Phone = phone
	return &g, nil
}

func CreateLegalGuardian(ctx context.Context, pool *pgxpool.Pool, g *LegalGuardian) error {
	return pool.QueryRow(ctx, `
		INSERT INTO legal_guardians (email, google_sub, password_hash, full_name, cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, birth_date, phone, auth_provider, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::auth_provider_enum, $13)
		RETURNING id
	`, g.Email, g.GoogleSub, g.PasswordHash, g.FullName, g.CPFEncrypted, g.CPFNonce, g.CPFKeyVersion, g.CPFHash, g.AddressID, g.BirthDate, g.Phone, g.AuthProvider, g.Status).Scan(&g.ID)
}

func UpdateLegalGuardianCPF(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, cpfEncrypted, cpfNonce []byte, cpfKeyVersion string, cpfHash string) error {
	result, err := pool.Exec(ctx, `
		UPDATE legal_guardians
		SET cpf_encrypted = $1,
		    cpf_nonce = $2,
		    cpf_key_version = $3::text,
		    cpf_hash = $4::text,
		    updated_at = now()
		WHERE id = $5 AND deleted_at IS NULL
	`, cpfEncrypted, cpfNonce, cpfKeyVersion, cpfHash, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
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

func GuardiansByPatient(ctx context.Context, pool *pgxpool.Pool, patientID uuid.UUID) ([]GuardianInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.id, g.full_name, g.email, g.phone, pg.relation
		FROM legal_guardians g
		JOIN patient_guardians pg ON pg.legal_guardian_id = g.id
		WHERE pg.patient_id = $1 AND g.status != 'CANCELLED' AND g.deleted_at IS NULL
		ORDER BY g.full_name
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []GuardianInfo
	for rows.Next() {
		var gi GuardianInfo
		if err := rows.Scan(&gi.ID, &gi.FullName, &gi.Email, &gi.Phone, &gi.Relation); err != nil {
			return nil, err
		}
		list = append(list, gi)
	}
	return list, rows.Err()
}

// UpdateLegalGuardian atualiza full_name, email, address_id, birth_date, phone. Se cpfHash != nil, atualiza cpf_hash.
func UpdateLegalGuardian(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, fullName, email string, addressID *uuid.UUID, birthDate, phone *string, cpfHash *string) error {
	result, err := pool.Exec(ctx, `
		UPDATE legal_guardians SET full_name = $1, email = $2, address_id = $3, birth_date = $4, phone = COALESCE($5, phone), cpf_hash = COALESCE($6, cpf_hash), updated_at = now()
		WHERE id = $7
	`, fullName, email, addressID, birthDate, phone, cpfHash, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpdateLegalGuardianAdmin atualiza cadastro do responsável legal (backoffice).
// Campos opcionais: se nil, mantém. addressID é UUID do endereço em addresses.
func UpdateLegalGuardianAdmin(
	ctx context.Context,
	pool *pgxpool.Pool,
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
	result, err := pool.Exec(ctx, `
		UPDATE legal_guardians
		SET
			full_name = COALESCE($1, full_name),
			email = COALESCE($2, email),
			address_id = COALESCE($3, address_id),
			birth_date = CASE WHEN $4::text IS NULL THEN birth_date ELSE NULLIF($4::text, '')::date END,
			phone = CASE WHEN $5::text IS NULL THEN phone ELSE NULLIF($5::text, '') END,
			status = COALESCE($6, status),
			password_hash = COALESCE($7, password_hash),
			auth_provider = COALESCE($8::auth_provider_enum, auth_provider),
			cpf_encrypted = CASE WHEN $9::bytea IS NULL THEN cpf_encrypted ELSE $9 END,
			cpf_nonce = CASE WHEN $10::bytea IS NULL THEN cpf_nonce ELSE $10 END,
			cpf_key_version = CASE WHEN $11::text IS NULL THEN cpf_key_version ELSE $11::text END,
			cpf_hash = CASE WHEN $12::text IS NULL THEN cpf_hash ELSE $12::text END,
			updated_at = now()
		WHERE id = $13 AND deleted_at IS NULL
	`, fullName, email, addressID, birthDate, phone, status, passwordHash, authProvider, cpfEncrypted, cpfNonce, cpfKeyVersion, cpfHash, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func SoftDeleteLegalGuardian(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	result, err := pool.Exec(ctx, `
		UPDATE legal_guardians SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
