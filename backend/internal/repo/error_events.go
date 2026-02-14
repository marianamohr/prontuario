package repo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ErrorEvent struct {
	RequestID             *string
	Source                string
	Severity              string
	ClinicID              *uuid.UUID
	ActorType             *string
	ActorID               *uuid.UUID
	IsImpersonated        bool
	ImpersonationSessionID *uuid.UUID
	HTTPMethod            *string
	Path                  *string
	ActionName            *string
	Kind                  *string
	Message               *string
	Stack                 *string
	PGCode                *string
	PGMessage             *string
	Metadata              interface{}
}

func CreateErrorEvent(ctx context.Context, pool *pgxpool.Pool, ev ErrorEvent) error {
	var meta []byte
	if ev.Metadata != nil {
		meta, _ = json.Marshal(ev.Metadata)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO error_events (
			request_id, source, severity,
			clinic_id, actor_type, actor_id, is_impersonated, impersonation_session_id,
			http_method, path, action_name,
			kind, message, stack, pg_code, pg_message, metadata
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16, $17
		)
	`, ev.RequestID, ev.Source, ev.Severity,
		ev.ClinicID, ev.ActorType, ev.ActorID, ev.IsImpersonated, ev.ImpersonationSessionID,
		ev.HTTPMethod, ev.Path, ev.ActionName,
		ev.Kind, ev.Message, ev.Stack, ev.PGCode, ev.PGMessage, meta)
	return err
}

