package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfessionalInvite struct {
	ID        uuid.UUID
	Token     string
	Email     string
	FullName  string
	ClinicID  uuid.UUID
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateProfessionalInvite creates a PENDING invite; token is returned for the registration URL.
func CreateProfessionalInvite(ctx context.Context, pool *pgxpool.Pool, email, fullName string, clinicID uuid.UUID, expiresAt time.Time) (*ProfessionalInvite, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO professional_invites (id, token, email, full_name, clinic_id, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, 'PENDING', $6)
	`, id, token, email, fullName, clinicID, expiresAt)
	if err != nil {
		return nil, err
	}
	return &ProfessionalInvite{
		ID: id, Token: token, Email: email, FullName: fullName, ClinicID: clinicID,
		Status: "PENDING", ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func GetProfessionalInviteByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*ProfessionalInvite, error) {
	var inv ProfessionalInvite
	err := pool.QueryRow(ctx, `
		SELECT id, token, email, full_name, clinic_id, status, expires_at, created_at
		FROM professional_invites WHERE token = $1
	`, token).Scan(&inv.ID, &inv.Token, &inv.Email, &inv.FullName, &inv.ClinicID, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListProfessionalInvites returns all invites ordered by created_at desc (for backoffice).
func ListProfessionalInvites(ctx context.Context, pool *pgxpool.Pool) ([]ProfessionalInvite, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, token, email, full_name, clinic_id, status, expires_at, created_at
		FROM professional_invites
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ProfessionalInvite
	for rows.Next() {
		var inv ProfessionalInvite
		if err := rows.Scan(&inv.ID, &inv.Token, &inv.Email, &inv.FullName, &inv.ClinicID, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}

// GetProfessionalInviteByID returns an invite by id (for resend/delete).
func GetProfessionalInviteByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*ProfessionalInvite, error) {
	var inv ProfessionalInvite
	err := pool.QueryRow(ctx, `
		SELECT id, token, email, full_name, clinic_id, status, expires_at, created_at
		FROM professional_invites WHERE id = $1
	`, id).Scan(&inv.ID, &inv.Token, &inv.Email, &inv.FullName, &inv.ClinicID, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// DeleteProfessionalInvite removes an invite by id. Does not delete the clinic.
func DeleteProfessionalInvite(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM professional_invites WHERE id = $1`, id)
	return err
}

func AcceptProfessionalInvite(ctx context.Context, pool *pgxpool.Pool, inviteID uuid.UUID, passwordHash string, fullName string, tradeName string, birthDate *string, cpfEncrypted, cpfNonce []byte, cpfKeyVersion *string, cpfHash *string, addressID *uuid.UUID, maritalStatus *string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var email, invFullName string
	var clinicID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT email, full_name, clinic_id FROM professional_invites
		WHERE id = $1 AND status = 'PENDING' AND expires_at > now()
	`, inviteID).Scan(&email, &invFullName, &clinicID)
	if err != nil {
		return err
	}
	if fullName == "" {
		fullName = invFullName
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO professionals (clinic_id, email, password_hash, full_name, trade_name, status, birth_date, cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, marital_status)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6, $7, $8, $9, $10, $11, $12)
	`, clinicID, email, passwordHash, fullName, tradeName, birthDate, cpfEncrypted, cpfNonce, cpfKeyVersion, cpfHash, addressID, maritalStatus)
	if err != nil {
		return err
	}

	// Atualiza o nome da clinic interna para refletir o nome fantasia (se informado).
	if tradeName != "" {
		_, errUpd := tx.Exec(ctx, `UPDATE clinics SET name = $1, updated_at = now() WHERE id = $2`, tradeName, clinicID)
		_ = errUpd
	}

	_, err = tx.Exec(ctx, `
		UPDATE professional_invites SET status = 'ACCEPTED', updated_at = now() WHERE id = $1
	`, inviteID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
