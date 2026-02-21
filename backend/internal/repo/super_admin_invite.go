package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SuperAdminInvite struct {
	ID        uuid.UUID
	Token     string
	Email     string
	FullName  string
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateSuperAdminInvite creates a PENDING invite; token is returned for the registration URL.
func CreateSuperAdminInvite(ctx context.Context, db *gorm.DB, email, fullName string, expiresAt time.Time) (*SuperAdminInvite, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	err = db.WithContext(ctx).Exec(`
		INSERT INTO super_admin_invites (id, token, email, full_name, status, expires_at)
		VALUES (?, ?, ?, ?, 'PENDING', ?)
	`, id, token, email, fullName, expiresAt).Error
	if err != nil {
		return nil, err
	}
	return &SuperAdminInvite{
		ID: id, Token: token, Email: email, FullName: fullName,
		Status: "PENDING", ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func GetSuperAdminInviteByToken(ctx context.Context, db *gorm.DB, token string) (*SuperAdminInvite, error) {
	var inv SuperAdminInvite
	err := db.WithContext(ctx).Raw(`
		SELECT id, token, email, full_name, status, expires_at, created_at
		FROM super_admin_invites WHERE token = ?
	`, token).Scan(&inv).Error
	if err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &inv, nil
}

// ListSuperAdminInvitesPaginated returns invites with limit/offset. If limit is 0, no limit.
func ListSuperAdminInvitesPaginated(ctx context.Context, db *gorm.DB, limit, offset int) ([]SuperAdminInvite, int, error) {
	var total int
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM super_admin_invites`).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	q := `
		SELECT id, token, email, full_name, status, expires_at, created_at
		FROM super_admin_invites
		ORDER BY created_at DESC
	`
	args := []interface{}{}
	if limit > 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	var list []SuperAdminInvite
	var err error
	if limit > 0 {
		err = db.WithContext(ctx).Raw(q, args...).Scan(&list).Error
	} else {
		err = db.WithContext(ctx).Raw(q).Scan(&list).Error
	}
	return list, total, err
}

// GetSuperAdminInviteByID returns an invite by id (for resend/delete).
func GetSuperAdminInviteByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*SuperAdminInvite, error) {
	var inv SuperAdminInvite
	err := db.WithContext(ctx).Raw(`
		SELECT id, token, email, full_name, status, expires_at, created_at
		FROM super_admin_invites WHERE id = ?
	`, id).Scan(&inv).Error
	if err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &inv, nil
}

// DeleteSuperAdminInvite removes an invite by id.
func DeleteSuperAdminInvite(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	return db.WithContext(ctx).Exec(`DELETE FROM super_admin_invites WHERE id = ?`, id).Error
}

// AcceptSuperAdminInvite creates a super admin and marks invite ACCEPTED. Runs in a transaction.
func AcceptSuperAdminInvite(ctx context.Context, db *gorm.DB, inviteID uuid.UUID, passwordHash string, fullName string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID       uuid.UUID
			Email    string
			FullName string
		}
		if err := tx.Raw(`
			SELECT id, email, full_name
			FROM super_admin_invites
			WHERE id = ? AND status = 'PENDING' AND expires_at > now()
		`, inviteID).Scan(&row).Error; err != nil || row.ID == uuid.Nil {
			return gorm.ErrRecordNotFound
		}
		useName := fullName
		if useName == "" {
			useName = row.FullName
		}
		if err := tx.Exec(`
			INSERT INTO super_admins (email, password_hash, full_name, status)
			VALUES (?, ?, ?, 'ACTIVE')
		`, row.Email, passwordHash, useName).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE super_admin_invites SET status = 'ACCEPTED', updated_at = now() WHERE id = ?
		`, inviteID).Error
	})
}
