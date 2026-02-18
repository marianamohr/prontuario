package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// Recover captura panics e retorna JSON consistente.
// O erro detalhado vai para log do servidor (stderr) via stack.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Log mínimo no stdout/stderr (Railway captura). Não inclui PII.
				// Inclui stack para debugging.
				log.Printf("[panic] request_id=%s path=%s err=%v\n%s", r.Header.Get("X-Request-ID"), r.URL.Path, rec, string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":      "internal",
					"request_id": r.Header.Get("X-Request-ID"),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
