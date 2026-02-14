package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/config"
	"github.com/prontuario/backend/internal/migrate"
	"github.com/prontuario/backend/internal/reminder"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}
	if err := migrate.Run(ctx, pool, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	tzName := os.Getenv("REMINDER_CRON_TZ")
	if tzName == "" {
		tzName = "America/Sao_Paulo"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Printf("REMINDER_CRON_TZ=%s invalid, using UTC: %v", tzName, err)
		loc = time.UTC
	}
	now := time.Now().In(loc)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	sender := reminder.DefaultWhatsAppSender(cfg.TwilioAccountSid, cfg.TwilioAuthToken, cfg.TwilioWhatsAppFrom)
	sent, skipped := reminder.SendAppointmentReminders(ctx, pool, tomorrow, sender)
	log.Printf("[reminder] done: sent=%d skipped=%d date=%s", sent, skipped, tomorrow.Format("2006-01-02"))
	os.Exit(0)
}
