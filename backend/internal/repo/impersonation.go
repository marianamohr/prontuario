package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ImpersonationTTL = 15 * time.Minute

func StartImpersonation(ctx context.Context, pool *pgxpool.Pool, adminID uuid.UUID, targetUserType string, targetUserID uuid.UUID, clinicID *uuid.UUID, reason string) (sessionID uuid.UUID, err error) {
	sessionID = uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO impersonation_sessions (id, admin_id, target_user_type, target_user_id, clinic_id, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, sessionID, adminID, targetUserType, targetUserID, clinicID, reason)
	return sessionID, err
}

func EndImpersonation(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE impersonation_sessions SET ended_at = now() WHERE id = $1`, sessionID)
	return err
}

func GetActiveImpersonation(ctx context.Context, pool *pgxpool.Pool, sessionID string) (adminID, targetUserID uuid.UUID, targetUserType string, clinicID *uuid.UUID, err error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", nil, err
	}
	var cid *uuid.UUID
	err = pool.QueryRow(ctx, `
		SELECT admin_id, target_user_id, target_user_type, clinic_id
		FROM impersonation_sessions
		WHERE id = $1 AND ended_at IS NULL AND started_at > $2
	`, sid, time.Now().Add(-ImpersonationTTL)).Scan(&adminID, &targetUserID, &targetUserType, &cid)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", nil, err
	}
	clinicID = cid
	return adminID, targetUserID, targetUserType, clinicID, nil
}
