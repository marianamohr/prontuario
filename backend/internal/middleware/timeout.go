package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout returns a middleware that enforces a maximum duration for each request.
// When the timeout is reached, the request context is cancelled and the handler may stop.
// If timeoutSec is 0 or negative, the middleware is a no-op.
func Timeout(timeoutSec int) func(http.Handler) http.Handler {
	if timeoutSec <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	timeout := time.Duration(timeoutSec) * time.Second
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
