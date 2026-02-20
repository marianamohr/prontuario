package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
func CreateProfessionalInvite(ctx context.Context, db *gorm.DB, email, fullName string, clinicID uuid.UUID, expiresAt time.Time) (*ProfessionalInvite, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	err = db.WithContext(ctx).Exec(`
		INSERT INTO professional_invites (id, token, email, full_name, clinic_id, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'PENDING', ?)
	`, id, token, email, fullName, clinicID, expiresAt).Error
	if err != nil {
		return nil, err
	}
	return &ProfessionalInvite{
		ID: id, Token: token, Email: email, FullName: fullName, ClinicID: clinicID,
		Status: "PENDING", ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func GetProfessionalInviteByToken(ctx context.Context, db *gorm.DB, token string) (*ProfessionalInvite, error) {
	var inv ProfessionalInvite
	err := db.WithContext(ctx).Raw(`
		SELECT id, token, email, full_name, clinic_id, status, expires_at, created_at
		FROM professional_invites WHERE token = ?
	`, token).Scan(&inv).Error
	if err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &inv, nil
}

// ListProfessionalInvites returns all invites ordered by created_at desc (for backoffice).
func ListProfessionalInvites(ctx context.Context, db *gorm.DB) ([]ProfessionalInvite, error) {
	list, _, err := ListProfessionalInvitesPaginated(ctx, db, 0, 0)
	return list, err
}

// ListProfessionalInvitesPaginated returns invites with limit/offset. If limit is 0, no limit.
func ListProfessionalInvitesPaginated(ctx context.Context, db *gorm.DB, limit, offset int) ([]ProfessionalInvite, int, error) {
	var total int
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM professional_invites`).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	q := `
		SELECT id, token, email, full_name, clinic_id, status, expires_at, created_at
		FROM professional_invites
		ORDER BY created_at DESC
	`
	args := []interface{}{}
	if limit > 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	var list []ProfessionalInvite
	var err error
	if limit > 0 {
		err = db.WithContext(ctx).Raw(q, args...).Scan(&list).Error
	} else {
		err = db.WithContext(ctx).Raw(q).Scan(&list).Error
	}
	return list, total, err
}

// GetProfessionalInviteByID returns an invite by id (for resend/delete).
func GetProfessionalInviteByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*ProfessionalInvite, error) {
	var inv ProfessionalInvite
	err := db.WithContext(ctx).Raw(`
		SELECT id, token, email, full_name, clinic_id, status, expires_at, created_at
		FROM professional_invites WHERE id = ?
	`, id).Scan(&inv).Error
	if err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &inv, nil
}

// DeleteProfessionalInvite removes an invite by id. Does not delete the clinic.
func DeleteProfessionalInvite(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	return db.WithContext(ctx).Exec(`DELETE FROM professional_invites WHERE id = ?`, id).Error
}

// AcceptProfessionalInvite creates professional, optionally updates clinic name, and marks invite ACCEPTED. Runs in a transaction.
func AcceptProfessionalInvite(ctx context.Context, db *gorm.DB, inviteID uuid.UUID, passwordHash string, fullName string, tradeName string, birthDate *string, cpfEncrypted, cpfNonce []byte, cpfKeyVersion *string, cpfHash *string, addressID *uuid.UUID, maritalStatus *string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv struct {
			Email    string
			FullName string
			ClinicID uuid.UUID
		}
		if err := tx.Raw(`
			SELECT email, full_name, clinic_id FROM professional_invites
			WHERE id = ? AND status = 'PENDING' AND expires_at > now()
		`, inviteID).Scan(&inv).Error; err != nil || inv.ClinicID == uuid.Nil {
			return err
		}
		if fullName == "" {
			fullName = inv.FullName
		}
		if err := tx.Exec(`
			INSERT INTO professionals (clinic_id, email, password_hash, full_name, trade_name, status, birth_date, cpf_encrypted, cpf_nonce, cpf_key_version, cpf_hash, address_id, marital_status)
			VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?, ?, ?, ?)
		`, inv.ClinicID, inv.Email, passwordHash, fullName, tradeName, birthDate, cpfEncrypted, cpfNonce, cpfKeyVersion, cpfHash, addressID, maritalStatus).Error; err != nil {
			return err
		}
		if tradeName != "" {
			_ = tx.Exec(`UPDATE clinics SET name = ?, updated_at = now() WHERE id = ?`, tradeName, inv.ClinicID)
		}
		return tx.Exec(`
			UPDATE professional_invites SET status = 'ACCEPTED', updated_at = now() WHERE id = ?
		`, inviteID).Error
	})
}
