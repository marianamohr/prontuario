package middleware

import (
	"net/http"

	"github.com/prontuario/backend/internal/auth"
)

// OptionalAuth tenta ler o Bearer token, mas não bloqueia se estiver ausente/inválido.
// Se válido, injeta claims no context para o handler usar.
func OptionalAuth(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := extractBearer(r)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		claims, err := auth.ParseJWT(secret, raw)
		if err == nil && claims != nil {
			r = r.WithContext(auth.WithClaims(r.Context(), claims))
		}
		next.ServeHTTP(w, r)
	})
}

func OptionalAuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return OptionalAuth(secret, next) }
}

