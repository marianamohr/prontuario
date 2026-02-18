package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PatientInvite struct {
	ID               uuid.UUID
	Token            string
	ClinicID         uuid.UUID
	GuardianEmail    string
	GuardianFullName string
	Status           string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

func generateInviteToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func CreatePatientInvite(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, guardianEmail, guardianFullName string, expiresAt time.Time) (*PatientInvite, error) {
	token, err := generateInviteToken()
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO patient_invites (id, token, clinic_id, guardian_email, guardian_full_name, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, 'PENDING', $6)
	`, id, token, clinicID, guardianEmail, guardianFullName, expiresAt)
	if err != nil {
		return nil, err
	}
	return &PatientInvite{
		ID: id, Token: token, ClinicID: clinicID,
		GuardianEmail: guardianEmail, GuardianFullName: guardianFullName,
		Status: "PENDING", ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func GetPatientInviteByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*PatientInvite, error) {
	var inv PatientInvite
	err := pool.QueryRow(ctx, `
		SELECT id, token, clinic_id, guardian_email, guardian_full_name, status, expires_at, created_at
		FROM patient_invites WHERE token = $1
	`, token).Scan(&inv.ID, &inv.Token, &inv.ClinicID, &inv.GuardianEmail, &inv.GuardianFullName, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// AcceptPatientInvite marks invite as ACCEPTED.
// The actual creation/update of patient/guardian is performed by the handler in a transaction.
func AcceptPatientInvite(ctx context.Context, pool *pgxpool.Pool, inviteID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE patient_invites SET status = 'ACCEPTED', updated_at = now()
		WHERE id = $1
	`, inviteID)
	return err
}
