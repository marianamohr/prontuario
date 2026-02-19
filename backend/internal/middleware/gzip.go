package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// gzipWriter wraps http.ResponseWriter to compress the body with gzip.
type gzipWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func (g *gzipWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	g.ResponseWriter.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
	g.gz = gzip.NewWriter(g.ResponseWriter)
}

func (g *gzipWriter) Write(p []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	return g.gz.Write(p)
}

func (g *gzipWriter) Close() error {
	if g.gz == nil {
		return nil
	}
	return g.gz.Close()
}

// Gzip compresses responses with gzip when the client sends Accept-Encoding: gzip.
// Handlers should not set Content-Length; the middleware removes it when compressing.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.Close()
		next.ServeHTTP(gw, r)
	})
}

