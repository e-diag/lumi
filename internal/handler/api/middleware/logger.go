// Пакет middleware содержит HTTP middleware для chi-роутера.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/freeway-vpn/backend/internal/logredact"
)

// RequestLogger логирует каждый HTTP-запрос через slog.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		slog.Info("http request",
			"method", r.Method,
			"path", logredact.HTTPPathForLog(r.URL.Path),
			"status", wrapped.status,
			"duration", time.Since(start).String(),
			"remote", ClientIP(r),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
