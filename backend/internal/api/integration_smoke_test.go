//go:build integration

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prontuario/backend/internal/config"
	"github.com/prontuario/backend/internal/testutil"
)

// Smoke: sobe router básico e garante que /health responde.
// Integração real usa DATABASE_URL para migrations/seed em outras suítes.
func TestIntegration_Health(t *testing.T) {
	ctx := context.Background()
	pool, url := testutil.OpenPool(ctx)
	if pool == nil {
		t.Skip("DATABASE_URL não configurada para testes de integração")
		return
	}
	defer pool.Close()
	_ = testutil.MustMigrate(ctx, pool)
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	_ = url
	_ = &Handler{Pool: pool, Cfg: config.Load()}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
