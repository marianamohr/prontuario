package config

import "os"

// Config holds env vars for the reminder service.
type Config struct {
	DatabaseURL        string
	TwilioAccountSid   string
	TwilioAuthToken    string
	TwilioWhatsAppFrom string
	Port               string
	APIKey             string
	AppPublicURL       string
}

// Load reads config from environment.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	return &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		TwilioAccountSid:   os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioWhatsAppFrom: os.Getenv("TWILIO_WHATSAPP_FROM"),
		Port:               port,
		APIKey:             os.Getenv("REMINDER_API_KEY"),
		AppPublicURL:       os.Getenv("APP_PUBLIC_URL"),
	}
}
