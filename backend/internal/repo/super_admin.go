package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SuperAdmin struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	Status       string
}

func SuperAdminByEmail(ctx context.Context, db *gorm.DB, email string) (*SuperAdmin, error) {
	var s SuperAdmin
	err := db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, full_name, status
		FROM super_admins WHERE email = ? AND status != 'CANCELLED'
	`, email).Scan(&s).Error
	if err != nil {
		return nil, err
	}
	if s.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &s, nil
}
