package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const ImpersonationTTL = 15 * time.Minute

func StartImpersonation(ctx context.Context, db *gorm.DB, adminID uuid.UUID, targetUserType string, targetUserID uuid.UUID, clinicID *uuid.UUID, reason string) (sessionID uuid.UUID, err error) {
	sessionID = uuid.New()
	return sessionID, db.WithContext(ctx).Exec(`
		INSERT INTO impersonation_sessions (id, admin_id, target_user_type, target_user_id, clinic_id, reason)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sessionID, adminID, targetUserType, targetUserID, clinicID, reason).Error
}

func EndImpersonation(ctx context.Context, db *gorm.DB, sessionID uuid.UUID) error {
	return db.WithContext(ctx).Exec(`UPDATE impersonation_sessions SET ended_at = now() WHERE id = ?`, sessionID).Error
}

func GetActiveImpersonation(ctx context.Context, db *gorm.DB, sessionID string) (adminID, targetUserID uuid.UUID, targetUserType string, clinicID *uuid.UUID, err error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", nil, err
	}
	var res struct {
		AdminID       uuid.UUID
		TargetUserID  uuid.UUID
		TargetUserType string
		ClinicID      *uuid.UUID
	}
	err = db.WithContext(ctx).Raw(`
		SELECT admin_id, target_user_id, target_user_type, clinic_id
		FROM impersonation_sessions
		WHERE id = ? AND ended_at IS NULL AND started_at > ?
	`, sid, time.Now().Add(-ImpersonationTTL)).Scan(&res).Error
	if err != nil {
		return uuid.Nil, uuid.Nil, "", nil, err
	}
	return res.AdminID, res.TargetUserID, res.TargetUserType, res.ClinicID, nil
}
