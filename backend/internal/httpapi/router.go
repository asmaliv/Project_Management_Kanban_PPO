package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the chi router with global middleware.
func NewRouter(logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack (order matches ARCHITECTURE.md)
	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)

	// Public endpoints (no auth required per SECURITY.md)
	r.Get("/healthz", HandleHealthz())

	// API v1 routes will be mounted here in later phases
	r.Route("/api/v1", func(r chi.Router) {
		// Phase 1+: auth, user, project, task, risk, finding, audit, dashboard
	})

	return r
}

// requestLogger logs each request with structured fields.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("request",
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

// securityHeaders adds baseline security headers to every response.
// Full CSP and HSTS will be configured in Phase 8.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		next.ServeHTTP(w, r)
	})
}
