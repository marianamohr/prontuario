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

	var timelineScan []struct {
		Kind                   string
		ID                     string
		Action                 string
		ActorType              string
		ActorID                *string
		ClinicID               *string
		RequestID              *string
		IP                     *string
		UserAgent              *string
		ResourceType           *string
		ResourceID             *string
		PatientID              *string
		IsImpersonated         bool
		ImpersonationSessionID *string
		Source                 string
		Severity               string
		Metadata               []byte
		CreatedAt              time.Time
	}
	err := h.DB.WithContext(r.Context()).Raw(`
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
			WHERE (?::timestamptz IS NULL OR created_at >= ?)
			  AND (?::timestamptz IS NULL OR created_at <= ?)
			  AND (?::text IS NULL OR request_id = ?)
			  AND (?::text IS NULL OR UPPER(severity) = ?)
			  AND (?::text IS NULL OR UPPER(source) = ?)
			  AND (?::uuid IS NULL OR actor_id = ?)
			  AND (?::uuid IS NULL OR clinic_id = ?)
			  AND (?::uuid IS NULL OR patient_id = ?)
			  AND (?::text IS NULL OR resource_type = ?)
			  AND (?::uuid IS NULL OR resource_id = ?)

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
			WHERE (?::timestamptz IS NULL OR created_at >= ?)
			  AND (?::timestamptz IS NULL OR created_at <= ?)
			  AND (?::text IS NULL OR request_id = ?)
			  AND (?::uuid IS NULL OR actor_id = ?)
			  AND (?::uuid IS NULL OR clinic_id = ?)
			  AND (?::uuid IS NULL OR patient_id = ?)
			  AND (?::text IS NULL OR resource_type = ?)
			  AND (?::uuid IS NULL OR resource_id = ?)
		)
		SELECT kind, id, action, actor_type, actor_id, clinic_id, request_id, ip, user_agent, resource_type, resource_id, patient_id,
		       is_impersonated, impersonation_session_id, source, severity, metadata, created_at
		FROM timeline
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, from, to, from, to,
		nullIfEmpty(requestID), nullIfEmpty(requestID),
		nullIfEmpty(severity), nullIfEmpty(severity),
		nullIfEmpty(source), nullIfEmpty(source),
		actorUUID, actorUUID,
		clinicUUID, clinicUUID,
		patientUUID, patientUUID,
		nullIfEmpty(resourceType), nullIfEmpty(resourceType),
		resourceUUID, resourceUUID,
		from, to, from, to,
		nullIfEmpty(requestID), nullIfEmpty(requestID),
		actorUUID, actorUUID,
		clinicUUID, clinicUUID,
		patientUUID, patientUUID,
		nullIfEmpty(resourceType), nullIfEmpty(resourceType),
		resourceUUID, resourceUUID,
		limit, offset,
	).Scan(&timelineScan).Error
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	out := make([]timelineRow, len(timelineScan))
	for i := range timelineScan {
		r := &timelineScan[i]
		out[i] = timelineRow{
			Kind: r.Kind, ID: r.ID, Action: r.Action, ActorType: r.ActorType,
			ActorID: r.ActorID, ClinicID: r.ClinicID, RequestID: r.RequestID,
			IP: r.IP, UserAgent: r.UserAgent, ResourceType: r.ResourceType, ResourceID: r.ResourceID, PatientID: r.PatientID,
			IsImpersonated: r.IsImpersonated, ImpersonationSessionID: r.ImpersonationSessionID,
			Source: r.Source, Severity: r.Severity, Metadata: r.Metadata, CreatedAt: r.CreatedAt.Format(time.RFC3339),
		}
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

	var scanRows []struct {
		ID         string
		CreatedAt  time.Time
		RequestID  *string
		Source     string
		Severity   string
		ClinicID   *string
		ActorType  *string
		ActorID    *string
		Path       *string
		Method     *string
		ActionName *string
		Kind       *string
		Message    *string
		Stack      *string
		PGCode     *string
		PGMessage  *string
		Metadata   []byte
	}
	err := h.DB.WithContext(r.Context()).Raw(`
		SELECT id::text, created_at, request_id, source, severity, clinic_id::text, actor_type, actor_id::text,
		       path, http_method, action_name, kind, message, stack, pg_code, pg_message,
		       COALESCE(metadata, '{}'::jsonb) AS metadata
		FROM error_events
		WHERE (?::timestamptz IS NULL OR created_at >= ?)
		  AND (?::timestamptz IS NULL OR created_at <= ?)
		  AND (?::text IS NULL OR request_id = ?)
		  AND (?::text IS NULL OR UPPER(severity) = ?)
		  AND (?::text IS NULL OR UPPER(source) = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, from, to, from, to, nullIfEmpty(requestID), nullIfEmpty(severity), nullIfEmpty(source), limit, offset).Scan(&scanRows).Error
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	out := make([]errorEventRow, len(scanRows))
	for i := range scanRows {
		r := &scanRows[i]
		out[i] = errorEventRow{
			ID: r.ID, CreatedAt: r.CreatedAt.Format(time.RFC3339), RequestID: r.RequestID, Source: r.Source, Severity: r.Severity,
			ClinicID: r.ClinicID, ActorType: r.ActorType, ActorID: r.ActorID, Path: r.Path,
			Method: r.Method, ActionName: r.ActionName, Kind: r.Kind, Message: r.Message, Stack: r.Stack,
			PGCode: r.PGCode, PGMessage: r.PGMessage, Metadata: r.Metadata,
		}
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
