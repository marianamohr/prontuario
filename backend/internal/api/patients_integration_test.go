//go:build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/config"
	"github.com/prontuario/backend/internal/middleware"
	"github.com/prontuario/backend/internal/seed"
	"github.com/prontuario/backend/internal/testutil"
	"gorm.io/gorm"
)

func newAPIRouterForPatients(h *Handler, jwtSecret []byte) http.Handler {
	r := mux.NewRouter()
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.RequireAuthMiddleware(jwtSecret))
	protected.Handle("/patients", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.ListPatients))).Methods(http.MethodGet)
	protected.Handle("/patients", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.CreatePatient))).Methods(http.MethodPost)
	protected.Handle("/patients/{patientId}", middleware.RequireRole(auth.RoleProfessional, auth.RoleSuperAdmin)(http.HandlerFunc(h.GetPatient))).Methods(http.MethodGet)
	return middleware.RequestID(r)
}

func getClinicAndProfessionalID(ctx context.Context, db *gorm.DB, email string) (uuid.UUID, uuid.UUID) {
	var res struct {
		ID       uuid.UUID
		ClinicID uuid.UUID
	}
	_ = db.WithContext(ctx).Raw(`SELECT id, clinic_id FROM professionals WHERE lower(email)=lower(?) LIMIT 1`, email).Scan(&res)
	return res.ClinicID, res.ID
}

func authHeaderForProfessional(t *testing.T, secret []byte, profID uuid.UUID, clinicID uuid.UUID) string {
	t.Helper()
	cid := clinicID.String()
	tok, err := auth.BuildJWT(secret, profID.String(), auth.RoleProfessional, &cid, false, nil, 2*time.Hour)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return "Bearer " + tok
}

func TestIntegration_TenantIsolation_ListPatients(t *testing.T) {
	ctx := context.Background()
	db, url := testutil.OpenDB(ctx)
	if db == nil {
		t.Skip("DATABASE_URL not set for integration tests")
		return
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	_ = url
	if err := testutil.MustMigrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = seed.Run(ctx, db)

	cfg := config.Load()
	jwtSecret := []byte("test-jwt-secret-min-32-chars-xxxxxxxxxxxx")
	cfg.JWTSecret = jwtSecret
	h := &Handler{DB: db, Cfg: cfg}

	clinicA, profA := getClinicAndProfessionalID(ctx, db, "profa@clinica-a.local")
	clinicB, profB := getClinicAndProfessionalID(ctx, db, "profb@clinica-b.local")
	if clinicA == uuid.Nil || clinicB == uuid.Nil || profA == uuid.Nil || profB == uuid.Nil {
		t.Fatal("seed did not create expected professionals")
	}

	srv := newAPIRouterForPatients(h, jwtSecret)

	// cria paciente na clínica A
	body := map[string]interface{}{"full_name": "Paciente Isolamento Via HTTP"}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/patients", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeaderForProfessional(t, jwtSecret, profA, clinicA))
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created.ID == "" {
		t.Fatalf("expected id in response, got %s", rr.Body.String())
	}

	// lista pacientes como clínica B e garante que não aparece o paciente criado
	req2 := httptest.NewRequest(http.MethodGet, "/api/patients", nil)
	req2.Header.Set("Authorization", authHeaderForProfessional(t, jwtSecret, profB, clinicB))
	rr2 := httptest.NewRecorder()
	srv.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	if bytes.Contains(rr2.Body.Bytes(), []byte(created.ID)) {
		t.Fatalf("patient from clinic A must not appear in clinic B list")
	}
}

func TestIntegration_PatientCPFOptional_UniquePerClinic(t *testing.T) {
	ctx := context.Background()
	db, _ := testutil.OpenDB(ctx)
	if db == nil {
		t.Skip("DATABASE_URL not set for integration tests")
		return
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	_ = testutil.MustMigrate(ctx, db)
	_ = seed.Run(ctx, db)

	cfg := config.Load()
	jwtSecret := []byte("test-jwt-secret-min-32-chars-xxxxxxxxxxxx")
	cfg.JWTSecret = jwtSecret
	h := &Handler{DB: db, Cfg: cfg}
	srv := newAPIRouterForPatients(h, jwtSecret)

	clinicA, profA := getClinicAndProfessionalID(ctx, db, "profa@clinica-a.local")
	if clinicA == uuid.Nil || profA == uuid.Nil {
		t.Fatal("seed did not create professional A")
	}
	authz := authHeaderForProfessional(t, jwtSecret, profA, clinicA)

	// cria paciente com CPF
	body := map[string]interface{}{"full_name": "Paciente CPF", "patient_cpf": "529.982.247-25"}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/patients", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created.ID == "" {
		t.Fatalf("expected id, got %s", rr.Body.String())
	}

	// tenta criar outro com mesmo CPF na mesma clínica
	body2 := map[string]interface{}{"full_name": "Paciente CPF 2", "patient_cpf": "52998224725"}
	raw2, err := json.Marshal(body2)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/patients", bytes.NewReader(raw2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", authz)
	rr2 := httptest.NewRecorder()
	srv.ServeHTTP(rr2, req2)
	if rr2.Code == http.StatusCreated {
		t.Fatalf("expected non-201 for duplicated cpf, got 201 body=%s", rr2.Body.String())
	}

	// get retorna cpf (decriptado) quando existir
	req3 := httptest.NewRequest(http.MethodGet, "/api/patients/"+created.ID, nil)
	req3.Header.Set("Authorization", authz)
	rr3 := httptest.NewRecorder()
	srv.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr3.Code, rr3.Body.String())
	}
	if !bytes.Contains(rr3.Body.Bytes(), []byte(`"cpf"`)) {
		t.Fatalf("expected cpf field in response, got %s", rr3.Body.String())
	}
}
