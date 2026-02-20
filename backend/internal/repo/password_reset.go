package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreatePasswordResetToken(ctx context.Context, db *gorm.DB, userType string, userID uuid.UUID, exp time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	expiresAt := time.Now().Add(exp)
	return token, db.WithContext(ctx).Exec(`
		INSERT INTO password_reset_tokens (token, user_type, user_id, expires_at)
		VALUES (?, ?, ?, ?)
	`, token, userType, userID, expiresAt).Error
}

func ConsumePasswordResetToken(ctx context.Context, db *gorm.DB, token string) (userType string, userID uuid.UUID, err error) {
	var res struct {
		UserType string
		UserID   uuid.UUID
	}
	err = db.WithContext(ctx).Raw(`
		UPDATE password_reset_tokens SET used_at = now() WHERE token = ? AND used_at IS NULL AND expires_at > now()
		RETURNING user_type, user_id
	`, token).Scan(&res).Error
	return res.UserType, res.UserID, err
}

func GetPasswordResetToken(ctx context.Context, db *gorm.DB, token string) (userType string, userID uuid.UUID, expiresAt time.Time, usedAt *time.Time, err error) {
	var res struct {
		UserType   string
		UserID     uuid.UUID
		ExpiresAt  time.Time
		UsedAt     *time.Time
	}
	err = db.WithContext(ctx).Raw(`
		SELECT user_type, user_id, expires_at, used_at FROM password_reset_tokens WHERE token = ?
	`, token).Scan(&res).Error
	return res.UserType, res.UserID, res.ExpiresAt, res.UsedAt, err
}
