package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func CreatePatientInvite(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, guardianEmail, guardianFullName string, expiresAt time.Time) (*PatientInvite, error) {
	token, err := generateInviteToken()
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	err = db.WithContext(ctx).Exec(`
		INSERT INTO patient_invites (id, token, clinic_id, guardian_email, guardian_full_name, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'PENDING', ?)
	`, id, token, clinicID, guardianEmail, guardianFullName, expiresAt).Error
	if err != nil {
		return nil, err
	}
	return &PatientInvite{
		ID: id, Token: token, ClinicID: clinicID,
		GuardianEmail: guardianEmail, GuardianFullName: guardianFullName,
		Status: "PENDING", ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func GetPatientInviteByToken(ctx context.Context, db *gorm.DB, token string) (*PatientInvite, error) {
	var inv PatientInvite
	err := db.WithContext(ctx).Raw(`
		SELECT id, token, clinic_id, guardian_email, guardian_full_name, status, expires_at, created_at
		FROM patient_invites WHERE token = ?
	`, token).Scan(&inv).Error
	if err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &inv, nil
}

// AcceptPatientInvite marks invite as ACCEPTED.
// The actual creation/update of patient/guardian is performed by the handler in a transaction.
func AcceptPatientInvite(ctx context.Context, db *gorm.DB, inviteID uuid.UUID) error {
	return db.WithContext(ctx).Exec(`
		UPDATE patient_invites SET status = 'ACCEPTED', updated_at = now()
		WHERE id = ?
	`, inviteID).Error
}
