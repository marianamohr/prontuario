package config

import (
	"os"
	"strconv"
)

// Config holds env vars for the reminder service.
type Config struct {
	DatabaseURL        string
	TwilioAccountSid   string
	TwilioAuthToken    string
	TwilioWhatsAppFrom string
	Port               string
	APIKey             string
	AppPublicURL       string
	ReminderDaysAhead  int  // Days ahead to send reminder (minimum 2).
	AutoConfirm        bool // If true, when sending reminder updates AGENDADO -> CONFIRMADO.
}

// Load reads config from environment.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	daysAhead := 2
	if v := os.Getenv("REMINDER_DAYS_AHEAD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 2 {
				n = 2
			}
			daysAhead = n
		}
	}
	autoConfirm := true
	switch os.Getenv("REMINDER_AUTO_CONFIRM") {
	case "false", "0", "no":
		autoConfirm = false
	default:
		if v := os.Getenv("REMINDER_AUTO_CONFIRM"); v == "true" || v == "1" || v == "yes" {
			autoConfirm = true
		}
	}
	return &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		TwilioAccountSid:   os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioWhatsAppFrom: os.Getenv("TWILIO_WHATSAPP_FROM"),
		Port:               port,
		APIKey:             os.Getenv("REMINDER_API_KEY"),
		AppPublicURL:       os.Getenv("APP_PUBLIC_URL"),
		ReminderDaysAhead:  daysAhead,
		AutoConfirm:        autoConfirm,
	}
}
