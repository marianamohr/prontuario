package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/middleware"
	"github.com/prontuario/backend/internal/repo"
)

type FrontendErrorIngestRequest struct {
	RequestID  *string                `json:"request_id"`
	Severity   string                 `json:"severity"` // WARN|ERROR
	Kind       string                 `json:"kind"`
	Message    string                 `json:"message"`
	Stack      *string                `json:"stack,omitempty"`
	HTTPMethod *string                `json:"http_method,omitempty"`
	Path       *string                `json:"path,omitempty"`
	Status     *int                   `json:"status,omitempty"`
	ActionName *string                `json:"action_name,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func (h *Handler) IngestFrontendError(w http.ResponseWriter, r *http.Request) {
	var req FrontendErrorIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	sev := strings.ToUpper(strings.TrimSpace(req.Severity))
	if sev != "WARN" && sev != "ERROR" {
		http.Error(w, `{"error":"severity inválida"}`, http.StatusBadRequest)
		return
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = "FRONTEND_ERROR"
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		msg = "frontend error"
	}

	// Contexto do actor (se houver JWT)
	var clinicID *uuid.UUID
	var actorType *string
	var actorID *uuid.UUID
	isImpersonated := false
	var sessionID *uuid.UUID

	if c := auth.ClaimsFrom(r.Context()); c != nil {
		at := c.Role
		actorType = &at
		if c.UserID != "" {
			if uid, err := uuid.Parse(c.UserID); err == nil {
				actorID = &uid
			}
		}
		if c.ClinicID != nil && strings.TrimSpace(*c.ClinicID) != "" {
			if cid, err := uuid.Parse(strings.TrimSpace(*c.ClinicID)); err == nil {
				clinicID = &cid
			}
		}
		isImpersonated = c.IsImpersonated
		if c.ImpersonationSessionID != nil && strings.TrimSpace(*c.ImpersonationSessionID) != "" {
			if sid, err := uuid.Parse(strings.TrimSpace(*c.ImpersonationSessionID)); err == nil {
				sessionID = &sid
			}
		}
	}

	rid := r.Header.Get("X-Request-ID")
	if req.RequestID != nil && strings.TrimSpace(*req.RequestID) != "" {
		rid = strings.TrimSpace(*req.RequestID)
	}
	if rid == "" {
		rid = middleware.RequestIDFromContext(r.Context())
	}
	var ridPtr *string
	if strings.TrimSpace(rid) != "" {
		ridPtr = &rid
	}

	var path *string
	if req.Path != nil && strings.TrimSpace(*req.Path) != "" {
		p := strings.TrimSpace(*req.Path)
		path = &p
	}
	var method *string
	if req.HTTPMethod != nil && strings.TrimSpace(*req.HTTPMethod) != "" {
		m := strings.ToUpper(strings.TrimSpace(*req.HTTPMethod))
		method = &m
	}
	var actionName *string
	if req.ActionName != nil && strings.TrimSpace(*req.ActionName) != "" {
		a := strings.TrimSpace(*req.ActionName)
		actionName = &a
	}
	stack := req.Stack

	// Metadata sem PII: aceitamos apenas o que o frontend mandar, mas o plano prevê não enviar PII.
	meta := map[string]interface{}{}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			meta[k] = v
		}
	}
	if req.Status != nil {
		meta["status"] = *req.Status
	}

	k := kind
	m := msg
	ev := repo.ErrorEvent{
		RequestID:             ridPtr,
		Source:                "FRONTEND",
		Severity:              sev,
		ClinicID:              clinicID,
		ActorType:             actorType,
		ActorID:               actorID,
		IsImpersonated:        isImpersonated,
		ImpersonationSessionID: sessionID,
		HTTPMethod:            method,
		Path:                  path,
		ActionName:            actionName,
		Kind:                  &k,
		Message:               &m,
		Stack:                 stack,
		Metadata:              meta,
	}
	_ = repo.CreateErrorEvent(r.Context(), h.Pool, ev)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

