package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/reminder/pkg/config"
	"github.com/prontuario/reminder/pkg/reminder"
)

// Server runs the reminder HTTP API.
type Server struct {
	pool   *pgxpool.Pool
	cfg    *config.Config
	sender reminder.WhatsAppSender
}

// New creates a server.
func New(pool *pgxpool.Pool, cfg *config.Config) *Server {
	sender := reminder.DefaultWhatsAppSender(cfg.TwilioAccountSid, cfg.TwilioAuthToken, cfg.TwilioWhatsAppFrom)
	return &Server{pool: pool, cfg: cfg, sender: sender}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /trigger", s.trigger)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) trigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.APIKey != "" {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}
		if key != s.cfg.APIKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}
	var professionalID *uuid.UUID
	if idStr := r.URL.Query().Get("professional_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, `{"error":"professional_id inválido"}`, http.StatusBadRequest)
			return
		}
		professionalID = &id
	}
	tzName := os.Getenv("REMINDER_CRON_TZ")
	if tzName == "" {
		tzName = "America/Sao_Paulo"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	targetDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, s.cfg.ReminderDaysAhead)
	sent, skipped := reminder.SendAppointmentReminders(r.Context(), s.pool, targetDate, s.sender, professionalID, s.cfg.AppPublicURL, s.cfg.AutoConfirm)
	log.Printf("[reminder] trigger done: sent=%d skipped=%d date=%s professional_id=%v", sent, skipped, targetDate.Format("2006-01-02"), professionalID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sent":    sent,
		"skipped": skipped,
		"date":    targetDate.Format("2006-01-02"),
	})
}
