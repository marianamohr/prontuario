package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreatePasswordResetToken(ctx context.Context, pool *pgxpool.Pool, userType string, userID uuid.UUID, exp time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	expiresAt := time.Now().Add(exp)
	_, err = pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (token, user_type, user_id, expires_at)
		VALUES ($1, $2, $3, $4)
	`, token, userType, userID, expiresAt)
	return token, err
}

func ConsumePasswordResetToken(ctx context.Context, pool *pgxpool.Pool, token string) (userType string, userID uuid.UUID, err error) {
	err = pool.QueryRow(ctx, `
		UPDATE password_reset_tokens SET used_at = now() WHERE token = $1 AND used_at IS NULL AND expires_at > now()
		RETURNING user_type, user_id
	`, token).Scan(&userType, &userID)
	return
}

func GetPasswordResetToken(ctx context.Context, pool *pgxpool.Pool, token string) (userType string, userID uuid.UUID, expiresAt time.Time, usedAt *time.Time, err error) {
	err = pool.QueryRow(ctx, `
		SELECT user_type, user_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1
	`, token).Scan(&userType, &userID, &expiresAt, &usedAt)
	return
}
