package repo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEvent struct {
	Action                 string
	ActorType              string
	ActorID                *uuid.UUID
	ClinicID               *uuid.UUID
	RequestID              string
	IP                     string
	UserAgent              string
	ResourceType           *string
	ResourceID             *uuid.UUID
	PatientID              *uuid.UUID
	IsImpersonated         bool
	ImpersonationSessionID *uuid.UUID
	Source                 *string // USER|SYSTEM
	Severity               *string // INFO|WARN|ERROR
	Metadata               interface{}
}

func CreateAuditEventFull(ctx context.Context, pool *pgxpool.Pool, ev AuditEvent) error {
	var meta []byte
	if ev.Metadata != nil {
		var marshalErr error
		meta, marshalErr = json.Marshal(ev.Metadata)
		if marshalErr != nil {
			return marshalErr
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO audit_events (
			action, actor_type, actor_id, clinic_id, request_id, ip, user_agent,
			resource_type, resource_id, patient_id, is_impersonated, impersonation_session_id,
			source, severity, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15
		)
	`,
		ev.Action, ev.ActorType, ev.ActorID, ev.ClinicID, nullIfEmptyText(ev.RequestID), nullIfEmptyText(ev.IP), nullIfEmptyText(ev.UserAgent),
		ev.ResourceType, ev.ResourceID, ev.PatientID, ev.IsImpersonated, ev.ImpersonationSessionID,
		ev.Source, ev.Severity, meta,
	)
	return err
}

func nullIfEmptyText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func CreateAuditEvent(ctx context.Context, pool *pgxpool.Pool, action, actorType string, actorID *uuid.UUID, metadata interface{}) error {
	return CreateAuditEventFull(ctx, pool, AuditEvent{
		Action:    action,
		ActorType: actorType,
		ActorID:   actorID,
		Metadata:  metadata,
	})
}

func CreateAccessLog(ctx context.Context, pool *pgxpool.Pool, clinicID, actorID *uuid.UUID, actorType, action, resourceType string, resourceID, patientID *uuid.UUID, ip, userAgent, requestID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO access_logs (clinic_id, actor_type, actor_id, action, resource_type, resource_id, patient_id, ip, user_agent, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, clinicID, actorType, actorID, action, resourceType, resourceID, patientID, ip, userAgent, requestID)
	return err
}
