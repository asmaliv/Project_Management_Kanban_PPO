package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/rbac"
	"ppo-kanban/internal/users"
)

type RouterParams struct {
	Logger        *slog.Logger
	AuthService   *auth.Service
	UserService   *users.Service
	SessionSecret string
	CookieSecure  bool
}

// NewRouter creates and configures the chi router with global middleware and API endpoints.
func NewRouter(params RouterParams) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack (order matches ARCHITECTURE.md)
	r.Use(middleware.RequestID)
	r.Use(requestLogger(params.Logger))
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)

	// Session middleware: extracts session from cookie if present.
	// Must be registered before any routes (chi requirement).
	// No-op for unauthenticated requests (no cookie = pass through).
	if params.AuthService != nil {
		r.Use(SessionMiddleware(params.AuthService, params.CookieSecure))
	}

	// Public endpoints (no auth required per SECURITY.md)
	r.Get("/healthz", HandleHealthz())

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		if params.AuthService != nil {
			authHandler := NewAuthHandler(params.AuthService, params.SessionSecret, params.CookieSecure)

			r.Route("/auth", func(r chi.Router) {
				// Login is public but strictly rate-limited
				r.With(RateLimitLogin()).Post("/login", authHandler.HandleLogin)

				// Authenticated auth routes
				r.Group(func(r chi.Router) {
					r.Use(RequireAuth)

					r.Get("/me", authHandler.HandleMe)
					r.Post("/logout", authHandler.HandleLogout)

					// Change password requires active session and CSRF token
					r.With(RequireCSRF(params.SessionSecret)).Post("/change-password", authHandler.HandleChangePassword)
				})
			})
		}

		if params.UserService != nil {
			userHandler := NewUserHandler(params.UserService)

			r.Route("/users", func(r chi.Router) {
				// Base requirements: authenticated and password change completed
				r.Use(RequireAuth)
				r.Use(RequirePasswordChanged)

				// GET /users: Admin sees full list; other roles see summary per RBAC matrix
				r.Get("/", userHandler.ListUsers)

				// Admin-only management endpoints
				r.Group(func(r chi.Router) {
					r.Use(RequireRole(rbac.RoleAdmin))

					r.Get("/{id}", userHandler.GetUserByID)

					// State-changing endpoints require CSRF
					r.Group(func(r chi.Router) {
						r.Use(RequireCSRF(params.SessionSecret))

						r.Post("/", userHandler.CreateUser)
						r.Patch("/{id}", userHandler.UpdateUser)
						r.Post("/{id}/deactivate", userHandler.DeactivateUser)
						r.Post("/{id}/activate", userHandler.ActivateUser)
						r.Post("/{id}/reset-password", userHandler.ResetPassword)
					})
				})
			})
		}
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
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		next.ServeHTTP(w, r)
	})
}
