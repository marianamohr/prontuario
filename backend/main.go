package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/api"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/cache"
	"github.com/prontuario/backend/internal/config"
	"github.com/prontuario/backend/internal/email"
	"github.com/prontuario/backend/internal/middleware"
	"github.com/prontuario/backend/internal/migrate"
	"github.com/prontuario/backend/internal/seed"
)

func main() {
	cfg := config.Load()
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("config postgres: %v", err)
		}
		if cfg.DBMaxConns > 0 {
			poolConfig.MaxConns = int32(cfg.DBMaxConns)
		}
		if cfg.DBMinConns > 0 {
			poolConfig.MinConns = int32(cfg.DBMinConns)
		}
		if cfg.DBMaxConnLifetime > 0 {
			poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
		}
		pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			log.Fatalf("conexão postgres: %v", err)
		}
		defer pool.Close()
		if err := pool.Ping(context.Background()); err != nil {
			log.Fatalf("ping postgres: %v", err)
		}
		if err := migrate.Run(context.Background(), pool, "migrations"); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		if err := seed.Run(context.Background(), pool); err != nil {
			log.Printf("seed (ignored if already applied): %v", err)
		}
	}

	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"no database"}`))
			return
		}
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"db unhealthy"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)

	h := &api.Handler{Pool: pool, Cfg: cfg, Cache: cache.New(30 * time.Second)}
	h.SetHashPassword(auth.HashPassword)
	if cfg.AppPublicURL != "" {
		mailCfg := &email.Config{
			Host:     cfg.SMTPHost,
			Port:     email.PortFromString(cfg.SMTPPort),
			User:     cfg.SMTPUser,
			Pass:     cfg.SMTPPass,
			FromName: cfg.SMTPFromName,
			FromAddr: cfg.SMTPFromEmail,
		}
		mailCfg.LogConfigSummary()
		h.SetSendPasswordResetEmail(func(to, token string) error {
			resetURL := cfg.AppPublicURL + "/reset-password?token=" + token
			return mailCfg.SendPasswordReset(to, resetURL)
		})
		h.SetSendContractSignedEmail(func(to, name string, pdf []byte, verificationToken string) error {
			verURL := cfg.AppPublicURL + "/verify/" + verificationToken
			body := "Olá, " + name + ",\n\nSegue em anexo a cópia do contrato assinado.\nLink para verificação: " + verURL
			return mailCfg.SendWithAttachment(to, "Contrato assinado - Prontuário Saúde", body, "contrato-assinado.pdf", pdf)
		})
		h.SetSendInviteEmail(func(to, fullName, registerURL string) error {
			return mailCfg.SendInvite(to, fullName, registerURL)
		})
		h.SetSendPatientInviteEmail(func(to, fullName, registerURL string) error {
			return mailCfg.SendPatientInvite(to, fullName, registerURL)
		})
		h.SetSendContractToSignEmail(func(to, fullName, signURL string) error {
			return mailCfg.SendContractToSign(to, fullName, signURL)
		})
		h.SetSendContractCancelledEmail(func(to, fullName string) error {
			return mailCfg.SendContractCancelled(to, fullName)
		})
		h.SetSendContractEndedEmail(func(to, fullName, endDate string) error {
			return mailCfg.SendContractEnded(to, fullName, endDate)
		})
		if cfg.SMTPUser == "" {
			log.Printf("[email] SMTP configurado: %s:%s (sem autenticação). E-mails em dev: veja no MailHog http://localhost:8025", cfg.SMTPHost, cfg.SMTPPort)
		} else {
			log.Printf("[email] SMTP configurado: %s:%s (autenticação ativa)", cfg.SMTPHost, cfg.SMTPPort)
		}
	} else {
		log.Printf("[email] Envio de e-mail desativado: APP_PUBLIC_URL vazio. Defina APP_PUBLIC_URL para habilitar convites, reset de senha e contratos por e-mail.")
	}
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/auth/login", h.Login).Methods(http.MethodPost)
	apiRouter.HandleFunc("/auth/register/guardian", h.GuardianRegister).Methods(http.MethodPost)
	apiRouter.HandleFunc("/auth/login/guardian", h.GuardianLogin).Methods(http.MethodPost)
	apiRouter.HandleFunc("/auth/password/forgot", h.ForgotPassword).Methods(http.MethodPost)
	apiRouter.HandleFunc("/auth/password/reset", h.ResetPassword).Methods(http.MethodPost)
	apiRouter.HandleFunc("/contracts/by-token", h.GetContractByToken).Methods(http.MethodGet)
	apiRouter.HandleFunc("/contracts/sign", h.SignContract).Methods(http.MethodPost)
	r.HandleFunc("/api/contracts/verify/{token}", h.GetContractVerify).Methods(http.MethodGet)
	r.HandleFunc("/api/appointments/remarcar/{token}", h.GetRemarcarByToken).Methods(http.MethodGet)
	r.HandleFunc("/api/appointments/remarcar/{token}/confirm", h.ConfirmRemarcar).Methods(http.MethodPost)
	r.HandleFunc("/api/appointments/remarcar/{token}", h.RemarcarAppointment).Methods(http.MethodPatch)
	apiRouter.HandleFunc("/invites/by-token", h.GetInviteByToken).Methods(http.MethodGet)
	apiRouter.HandleFunc("/invites/accept", h.AcceptInvite).Methods(http.MethodPost)
	apiRouter.HandleFunc("/patient-invites/by-token", h.GetPatientInviteByToken).Methods(http.MethodGet)
	apiRouter.HandleFunc("/patient-invites/accept", h.AcceptPatientInvite).Methods(http.MethodPost)
	// Ingestão de erros do frontend (sem PII). Auth é opcional: se houver JWT, enriquece o contexto.
	apiRouter.Handle("/errors/frontend", middleware.OptionalAuthMiddleware(cfg.JWTSecret)(http.HandlerFunc(h.IngestFrontendError))).Methods(http.MethodPost)

	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.RequireAuthMiddleware(cfg.JWTSecret))
	protected.HandleFunc("/me", h.Me).Methods(http.MethodGet)
	protected.Handle("/me/signature", middleware.RequireRole(auth.RoleProfessional)(http.HandlerFunc(h.GetMySignature))).Methods(http.MethodGet)
	protected.Handle("/me/signature", middleware.RequireRole(auth.RoleProfessional)(http.HandlerFunc(h.PutMySignature))).Methods(http.MethodPut)
	protected.Handle("/me/branding", middleware.RequireRole(auth.RoleProfessional)(http.HandlerFunc(h.GetMyBranding))).Methods(http.MethodGet)
	protected.Handle("/me/branding", middleware.RequireRole(auth.RoleProfessional)(http.HandlerFunc(h.PutMyBranding))).Methods(http.MethodPut)
	protected.Handle("/me/profile", middleware.RequireRole(auth.RoleProfessional)(http.HandlerFunc(h.GetMyProfile))).Methods(http.MethodGet)
	protected.Handle("/me/profile", middleware.RequireRole(auth.RoleProfessional)(http.HandlerFunc(h.PatchMyProfile))).Methods(http.MethodPatch)
	protected.Handle("/patients", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListPatients))).Methods(http.MethodGet)
	protected.Handle("/patients/{patientId}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.GetPatient))).Methods(http.MethodGet)
	protected.Handle("/patients/{patientId}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.UpdatePatient))).Methods(http.MethodPatch)
	// Soft delete: permitido para SUPER_ADMIN e para SUPER_ADMIN em modo impersonate (token com Role=PROFESSIONAL).
	protected.Handle("/patients/{patientId}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.SoftDeletePatient))).Methods(http.MethodDelete)
	protected.Handle("/patients", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CreatePatient))).Methods(http.MethodPost)
	protected.Handle("/patient-invites", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CreatePatientInvite))).Methods(http.MethodPost)
	protected.Handle("/contract-templates", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListContractTemplates))).Methods(http.MethodGet)
	protected.Handle("/contract-templates", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CreateContractTemplate))).Methods(http.MethodPost)
	protected.Handle("/contract-templates/{id}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.GetContractTemplate))).Methods(http.MethodGet)
	protected.Handle("/contract-templates/{id}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.UpdateContractTemplate))).Methods(http.MethodPut)
	protected.Handle("/contract-templates/{id}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.DeleteContractTemplate))).Methods(http.MethodDelete)
	protected.Handle("/contracts", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListContracts))).Methods(http.MethodGet)
	protected.Handle("/contracts/pending", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListPendingContracts))).Methods(http.MethodGet)
	protected.Handle("/contracts/for-agenda", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListContractsForAgenda))).Methods(http.MethodGet)
	protected.Handle("/contracts", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CreateContract))).Methods(http.MethodPost)
	protected.HandleFunc("/patients/{patientId}/record-entries", h.ListRecordEntries).Methods(http.MethodGet)
	protected.HandleFunc("/patients/{patientId}/record-entries", h.CreateRecordEntry).Methods(http.MethodPost)
	protected.Handle("/patients/{patientId}/guardians", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin, auth.RoleLegalGuardian)(http.HandlerFunc(h.ListPatientGuardians))).Methods(http.MethodGet)
	// Soft delete: permitido para SUPER_ADMIN e para SUPER_ADMIN em modo impersonate (token com Role=PROFESSIONAL).
	protected.Handle("/patients/{patientId}/guardians/{guardianId}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.SoftDeleteGuardian))).Methods(http.MethodDelete)
	protected.Handle("/patients/{patientId}/contracts", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListPatientContracts))).Methods(http.MethodGet)
	protected.Handle("/patients/{patientId}/send-contract", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.SendContractForPatient))).Methods(http.MethodPost)
	protected.Handle("/patients/{patientId}/contract-preview", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.GetContractPreview))).Methods(http.MethodGet)
	protected.Handle("/patients/{patientId}/contracts/{contractId}/preview", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.GetContractPreviewByID))).Methods(http.MethodGet)
	protected.Handle("/patients/{patientId}/contracts/{contractId}/resend", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ResendContract))).Methods(http.MethodPost)
	protected.Handle("/patients/{patientId}/contracts/{contractId}/cancel", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CancelContract))).Methods(http.MethodPost)
	protected.Handle("/patients/{patientId}/contracts/{contractId}/end", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.EndContract))).Methods(http.MethodPut)
	// Soft delete: permitido para SUPER_ADMIN e para SUPER_ADMIN em modo impersonate (token com Role=PROFESSIONAL).
	protected.Handle("/patients/{patientId}/contracts/{contractId}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.SoftDeleteContract))).Methods(http.MethodDelete)
	protected.Handle("/me/schedule-config", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.GetScheduleConfig))).Methods(http.MethodGet)
	protected.Handle("/me/schedule-config", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.PutScheduleConfig))).Methods(http.MethodPut)
	protected.Handle("/me/schedule-config/copy", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CopyScheduleConfigDay))).Methods(http.MethodPost)
	protected.Handle("/appointments", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListAppointments))).Methods(http.MethodGet)
	protected.Handle("/appointments", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CreateAppointments))).Methods(http.MethodPost)
	protected.Handle("/appointments/{id}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.PatchAppointment))).Methods(http.MethodPatch)
	protected.Handle("/backoffice/users", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.ListUsersBackoffice))).Methods(http.MethodGet)
	protected.Handle("/backoffice/users/{type}/{id}", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.GetBackofficeUser))).Methods(http.MethodGet)
	protected.Handle("/backoffice/users/{type}/{id}", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.PatchBackofficeUser))).Methods(http.MethodPatch)
	protected.Handle("/backoffice/professionals/{id}/related", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.BackofficeProfessionalRelatedData))).Methods(http.MethodGet)
	protected.Handle("/backoffice/timeline", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.BackofficeTimeline))).Methods(http.MethodGet)
	protected.Handle("/backoffice/errors", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.BackofficeErrors))).Methods(http.MethodGet)
	protected.Handle("/backoffice/cleanup-orphan-addresses", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.CleanupOrphanAddresses))).Methods(http.MethodPost)
	protected.Handle("/backoffice/invites", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.ListInvites))).Methods(http.MethodGet)
	protected.Handle("/backoffice/invites", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.CreateInvite))).Methods(http.MethodPost)
	protected.Handle("/backoffice/invites/{id}", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.DeleteInvite))).Methods(http.MethodDelete)
	protected.Handle("/backoffice/invites/{id}/resend", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.ResendInvite))).Methods(http.MethodPost)
	protected.Handle("/backoffice/reminder/trigger", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.TriggerReminder))).Methods(http.MethodPost)
	protected.Handle("/backoffice/impersonate/start", middleware.RequireRole(auth.RoleSuperAdmin)(http.HandlerFunc(h.ImpersonateStart))).Methods(http.MethodPost)
	protected.HandleFunc("/backoffice/impersonate/end", h.ImpersonateEnd).Methods(http.MethodPost)

	chain := middleware.Recover(middleware.RequestID(middleware.Timeout(cfg.RequestTimeoutSec)(middleware.CORS(cfg.CORSOrigins)(middleware.Gzip(r)))))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      chain,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("backend listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("backend stopped")
}
