package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
)

type timelineRow struct {
	Kind                   string          `json:"kind"`
	ID                     string          `json:"id"`
	Action                 string          `json:"action"`
	ActorType              string          `json:"actor_type"`
	ActorID                *string         `json:"actor_id,omitempty"`
	ClinicID               *string         `json:"clinic_id,omitempty"`
	RequestID              *string         `json:"request_id,omitempty"`
	IP                     *string         `json:"ip,omitempty"`
	UserAgent              *string         `json:"user_agent,omitempty"`
	ResourceType           *string         `json:"resource_type,omitempty"`
	ResourceID             *string         `json:"resource_id,omitempty"`
	PatientID              *string         `json:"patient_id,omitempty"`
	IsImpersonated         bool            `json:"is_impersonated"`
	ImpersonationSessionID *string         `json:"impersonation_session_id,omitempty"`
	Source                 string          `json:"source"`
	Severity               string          `json:"severity"`
	Metadata               json.RawMessage `json:"metadata,omitempty"`
	CreatedAt              string          `json:"created_at"`
}

// BackofficeTimeline retorna uma timeline unificada (audit_events + access_logs), ordenada por created_at desc.
func (h *Handler) BackofficeTimeline(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	q := r.URL.Query()
	limit := 50
	offset := 0
	if s := strings.TrimSpace(q.Get("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if s := strings.TrimSpace(q.Get("offset")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			offset = n
		}
	}

	// Filtros
	var from *time.Time
	var to *time.Time
	if s := strings.TrimSpace(q.Get("from")); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = &t
		} else if t2, err2 := time.Parse("2006-01-02", s); err2 == nil {
			from = &t2
		}
	}
	if s := strings.TrimSpace(q.Get("to")); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = &t
		} else if t2, err2 := time.Parse("2006-01-02", s); err2 == nil {
			// inclui o dia inteiro
			tt := t2.Add(24*time.Hour - time.Nanosecond)
			to = &tt
		}
	}
	requestID := strings.TrimSpace(q.Get("request_id"))
	severity := strings.ToUpper(strings.TrimSpace(q.Get("severity")))
	source := strings.ToUpper(strings.TrimSpace(q.Get("source")))

	// Filtra por actor_id / clinic_id / patient_id se vierem (UUIDs)
	var actorUUID *uuid.UUID
	if s := strings.TrimSpace(q.Get("actor_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			actorUUID = &id
		}
	}
	var clinicUUID *uuid.UUID
	if s := strings.TrimSpace(q.Get("clinic_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			clinicUUID = &id
		}
	}
	var patientUUID *uuid.UUID
	if s := strings.TrimSpace(q.Get("patient_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			patientUUID = &id
		}
	}
	resourceType := strings.TrimSpace(q.Get("resource_type"))
	var resourceUUID *uuid.UUID
	if s := strings.TrimSpace(q.Get("resource_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			resourceUUID = &id
		}
	}

	rows, err := h.Pool.Query(r.Context(), `
		WITH timeline AS (
			SELECT
				'AUDIT'::text AS kind,
				id::text AS id,
				action,
				actor_type,
				actor_id::text AS actor_id,
				clinic_id::text AS clinic_id,
				request_id,
				ip,
				user_agent,
				resource_type,
				resource_id::text AS resource_id,
				patient_id::text AS patient_id,
				COALESCE(is_impersonated, false) AS is_impersonated,
				impersonation_session_id::text AS impersonation_session_id,
				COALESCE(source, 'USER') AS source,
				COALESCE(severity, 'INFO') AS severity,
				COALESCE(metadata, '{}'::jsonb) AS metadata,
				created_at
			FROM audit_events
			WHERE ($1::timestamptz IS NULL OR created_at >= $1)
			  AND ($2::timestamptz IS NULL OR created_at <= $2)
			  AND ($3::text IS NULL OR request_id = $3)
			  AND ($4::text IS NULL OR UPPER(severity) = $4)
			  AND ($5::text IS NULL OR UPPER(source) = $5)
			  AND ($6::uuid IS NULL OR actor_id = $6)
			  AND ($7::uuid IS NULL OR clinic_id = $7)
			  AND ($8::uuid IS NULL OR patient_id = $8)
			  AND ($9::text IS NULL OR resource_type = $9)
			  AND ($10::uuid IS NULL OR resource_id = $10)

			UNION ALL

			SELECT
				'ACCESS'::text AS kind,
				id::text AS id,
				action,
				actor_type,
				actor_id::text AS actor_id,
				clinic_id::text AS clinic_id,
				request_id,
				ip,
				user_agent,
				resource_type,
				resource_id::text AS resource_id,
				patient_id::text AS patient_id,
				false AS is_impersonated,
				NULL::text AS impersonation_session_id,
				'USER'::text AS source,
				'INFO'::text AS severity,
				jsonb_build_object('action', action) AS metadata,
				created_at
			FROM access_logs
			WHERE ($1::timestamptz IS NULL OR created_at >= $1)
			  AND ($2::timestamptz IS NULL OR created_at <= $2)
			  AND ($3::text IS NULL OR request_id = $3)
			  AND ($6::uuid IS NULL OR actor_id = $6)
			  AND ($7::uuid IS NULL OR clinic_id = $7)
			  AND ($8::uuid IS NULL OR patient_id = $8)
			  AND ($9::text IS NULL OR resource_type = $9)
			  AND ($10::uuid IS NULL OR resource_id = $10)
		)
		SELECT kind, id, action, actor_type, actor_id, clinic_id, request_id, ip, user_agent, resource_type, resource_id, patient_id,
		       is_impersonated, impersonation_session_id, source, severity, metadata, created_at
		FROM timeline
		ORDER BY created_at DESC
		LIMIT $11 OFFSET $12
	`, from, to,
		nullIfEmpty(requestID),
		nullIfEmpty(severity),
		nullIfEmpty(source),
		actorUUID,
		clinicUUID,
		patientUUID,
		nullIfEmpty(resourceType),
		resourceUUID,
		limit, offset,
	)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []timelineRow
	for rows.Next() {
		var row timelineRow
		var actorID, clinicID, requestID, ip, ua, rType, rID, pID, impSID *string
		var meta []byte
		var created time.Time
		if err := rows.Scan(&row.Kind, &row.ID, &row.Action, &row.ActorType, &actorID, &clinicID, &requestID, &ip, &ua, &rType, &rID, &pID,
			&row.IsImpersonated, &impSID, &row.Source, &row.Severity, &meta, &created); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		row.ActorID = actorID
		row.ClinicID = clinicID
		row.RequestID = requestID
		row.IP = ip
		row.UserAgent = ua
		row.ResourceType = rType
		row.ResourceID = rID
		row.PatientID = pID
		row.ImpersonationSessionID = impSID
		row.Metadata = meta
		row.CreatedAt = created.Format(time.RFC3339)
		out = append(out, row)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": out, "limit": limit, "offset": offset})
}

type errorEventRow struct {
	ID         string          `json:"id"`
	CreatedAt  string          `json:"created_at"`
	RequestID  *string         `json:"request_id,omitempty"`
	Source     string          `json:"source"`
	Severity   string          `json:"severity"`
	ClinicID   *string         `json:"clinic_id,omitempty"`
	ActorType  *string         `json:"actor_type,omitempty"`
	ActorID    *string         `json:"actor_id,omitempty"`
	Path       *string         `json:"path,omitempty"`
	Method     *string         `json:"http_method,omitempty"`
	ActionName *string         `json:"action_name,omitempty"`
	Kind       *string         `json:"kind,omitempty"`
	Message    *string         `json:"message,omitempty"`
	Stack      *string         `json:"stack,omitempty"`
	PGCode     *string         `json:"pg_code,omitempty"`
	PGMessage  *string         `json:"pg_message,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

func (h *Handler) BackofficeErrors(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	q := r.URL.Query()
	limit := 50
	offset := 0
	if s := strings.TrimSpace(q.Get("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if s := strings.TrimSpace(q.Get("offset")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			offset = n
		}
	}

	var from *time.Time
	var to *time.Time
	if s := strings.TrimSpace(q.Get("from")); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = &t
		} else if t2, err2 := time.Parse("2006-01-02", s); err2 == nil {
			from = &t2
		}
	}
	if s := strings.TrimSpace(q.Get("to")); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = &t
		} else if t2, err2 := time.Parse("2006-01-02", s); err2 == nil {
			tt := t2.Add(24*time.Hour - time.Nanosecond)
			to = &tt
		}
	}
	requestID := strings.TrimSpace(q.Get("request_id"))
	severity := strings.ToUpper(strings.TrimSpace(q.Get("severity")))
	source := strings.ToUpper(strings.TrimSpace(q.Get("source")))

	rows, err := h.Pool.Query(r.Context(), `
		SELECT id::text, created_at, request_id, source, severity, clinic_id::text, actor_type, actor_id::text,
		       path, http_method, action_name, kind, message, stack, pg_code, pg_message,
		       COALESCE(metadata, '{}'::jsonb) AS metadata
		FROM error_events
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		  AND ($2::timestamptz IS NULL OR created_at <= $2)
		  AND ($3::text IS NULL OR request_id = $3)
		  AND ($4::text IS NULL OR UPPER(severity) = $4)
		  AND ($5::text IS NULL OR UPPER(source) = $5)
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7
	`, from, to, nullIfEmpty(requestID), nullIfEmpty(severity), nullIfEmpty(source), limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []errorEventRow
	for rows.Next() {
		var row errorEventRow
		var created time.Time
		var meta []byte
		if err := rows.Scan(&row.ID, &created, &row.RequestID, &row.Source, &row.Severity, &row.ClinicID, &row.ActorType, &row.ActorID,
			&row.Path, &row.Method, &row.ActionName, &row.Kind, &row.Message, &row.Stack, &row.PGCode, &row.PGMessage, &meta); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		row.CreatedAt = created.Format(time.RFC3339)
		row.Metadata = meta
		out = append(out, row)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": out, "limit": limit, "offset": offset})
}

func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	t := strings.TrimSpace(s)
	return &t
}
