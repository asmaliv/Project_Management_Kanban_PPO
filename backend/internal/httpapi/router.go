package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/project"
	"ppo-kanban/internal/task"
)

type RouterParams struct {
	Logger         *slog.Logger
	AuthService    *auth.Service
	AuditService   *auditlog.Service
	ProjectService *project.Service
	TaskService    *task.Service
	SessionSecret  string
	CookieSecure   bool
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
		
		// Audit logs endpoint (admin and audit roles only)
		if params.AuditService != nil {
			auditHandler := NewAuditLogHandler(params.AuditService)

			r.Group(func(r chi.Router) {
				r.Use(RequireAuth)
				r.Use(RequireRoles(db.UserRoleAdmin, db.UserRoleAudit))

				r.Get("/audit-logs", auditHandler.HandleListAuditLogs)
			})
		}

		// Project, Member, and Board Column endpoints
		if params.ProjectService != nil {
			projectHandler := NewProjectHandler(params.ProjectService)

			r.Group(func(r chi.Router) {
				r.Use(RequireAuth)

				// Projects
				r.Get("/projects", projectHandler.HandleListProjects)
				r.With(RequireRoles(db.UserRoleProjectManager)).Post("/projects", projectHandler.HandleCreateProject)

				r.Route("/projects/{id}", func(r chi.Router) {
					r.Get("/", projectHandler.HandleGetProject)
					r.Patch("/", projectHandler.HandleUpdateProject)
					r.Put("/", projectHandler.HandleUpdateProject)
					r.Post("/archive", projectHandler.HandleArchiveProject)
					r.With(RequireRoles(db.UserRoleAdmin)).Post("/transfer-owner", projectHandler.HandleTransferOwner)

					// Members
					r.Get("/members", projectHandler.HandleListMembers)
					r.Post("/members", projectHandler.HandleAddMember)
					r.Delete("/members/{userId}", projectHandler.HandleRemoveMember)

					// Columns on project
					r.Get("/columns", projectHandler.HandleListColumns)
					r.Get("/board", projectHandler.HandleListColumns) // alias for board view
					r.Post("/columns", projectHandler.HandleCreateColumn)
				})

				// Columns standalone routes
				r.Route("/columns/{columnId}", func(r chi.Router) {
					r.Patch("/", projectHandler.HandleUpdateColumn)
					r.Put("/", projectHandler.HandleUpdateColumn)
					r.Delete("/", projectHandler.HandleDeleteColumn)
				})
			})
		}

		// Tasks and Comments endpoints
		if params.TaskService != nil {
			taskHandler := NewTaskHandler(params.TaskService)

			r.Group(func(r chi.Router) {
				r.Use(RequireAuth)

				// Cross-project filtered tasks
				r.Get("/tasks", taskHandler.HandleListTasks)

				// Project tasks: GET /projects/{id}/tasks, POST /projects/{id}/tasks
				r.Route("/projects/{id}/tasks", func(r chi.Router) {
					r.Get("/", taskHandler.HandleListBoardTasks)
					r.Post("/", taskHandler.HandleCreateTask)
				})

				// Task operations: GET /tasks/{id}, PATCH /tasks/{id}, POST /tasks/{id}/move, DELETE /tasks/{id}
				r.Route("/tasks/{id}", func(r chi.Router) {
					r.Get("/", taskHandler.HandleGetTask)
					r.Patch("/", taskHandler.HandleUpdateTask)
					r.Put("/", taskHandler.HandleUpdateTask)
					r.Post("/move", taskHandler.HandleMoveTask)
					r.Delete("/", taskHandler.HandleDeleteTask)

					// Comments
					r.Get("/comments", taskHandler.HandleListComments)
					r.Post("/comments", taskHandler.HandleCreateComment)
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
