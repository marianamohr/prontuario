package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SuperAdmin struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	Status       string
}

func SuperAdminByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*SuperAdmin, error) {
	var s SuperAdmin
	err := pool.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, status
		FROM super_admins WHERE email = $1 AND status != 'CANCELLED'
	`, email).Scan(&s.ID, &s.Email, &s.PasswordHash, &s.FullName, &s.Status)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
