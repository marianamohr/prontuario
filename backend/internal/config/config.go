package config

import (
	"os"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          []byte
	CORSOrigins        []string
	DataEncryptionKeys string
	CurrentDataKeyVer  string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPass           string
	SMTPFromName       string
	SMTPFromEmail      string
	AppPublicURL       string
	BackendPublicURL   string
	// WhatsApp (Twilio) para lembretes de consulta
	TwilioAccountSid   string
	TwilioAuthToken    string
	TwilioWhatsAppFrom string
	// Serviço reminder (cron separado) – para proxy do backoffice
	ReminderServiceURL string
	ReminderAPIKey     string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		jwtSecret = "default-secret-min-32-chars-required!!"
	}
	cors := os.Getenv("CORS_ORIGINS")
	if cors == "" {
		cors = "http://localhost:5173"
	}
	var origins []string
	for _, o := range strings.Split(cors, ",") {
		if t := strings.TrimSpace(o); t != "" {
			origins = append(origins, t)
		}
	}
	return &Config{
		Port:               port,
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          []byte(jwtSecret),
		CORSOrigins:        origins,
		DataEncryptionKeys: getEnv("DATA_ENCRYPTION_KEYS", "v1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
		CurrentDataKeyVer:  getEnv("CURRENT_DATA_KEY_VERSION", "v1"),
		SMTPHost:           getEnv("SMTP_HOST", "localhost"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPass:           os.Getenv("SMTP_PASS"),
		SMTPFromName:       getEnv("SMTP_FROM_NAME", "Prontuário Saúde"),
		SMTPFromEmail:      getEnv("SMTP_FROM_EMAIL", "noreply@localhost"),
		AppPublicURL:       getEnv("APP_PUBLIC_URL", "http://localhost:5173"),
		BackendPublicURL:   getEnv("BACKEND_PUBLIC_URL", "http://localhost:8080"),
		TwilioAccountSid:     os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:      os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioWhatsAppFrom:   os.Getenv("TWILIO_WHATSAPP_FROM"),
		ReminderServiceURL:   os.Getenv("REMINDER_SERVICE_URL"),
		ReminderAPIKey:       os.Getenv("REMINDER_API_KEY"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
