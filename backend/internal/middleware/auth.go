package middleware

import (
	"net/http"
	"strings"

	"github.com/prontuario/backend/internal/auth"
)

// RequireAuthMiddleware returns a mux-compatible middleware (func(http.Handler) http.Handler).
func RequireAuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuth(secret, next)
	}
}

func RequireAuth(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := extractBearer(r)
		if raw == "" {
			http.Error(w, `{"error":"missing or invalid authorization"}`, http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseJWT(secret, raw)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		r = r.WithContext(auth.WithClaims(r.Context(), claims))
		next.ServeHTTP(w, r)
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := auth.ClaimsFrom(r.Context())
			if c == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			for _, role := range roles {
				if c.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}

func RequireSuperAdmin(next http.Handler) http.Handler {
	return RequireRole(auth.RoleSuperAdmin)(next)
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}
