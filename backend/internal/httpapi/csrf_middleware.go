package httpapi

import (
	"net/http"

	"ppo-kanban/internal/auth"
)

// RequireCSRF validates the X-CSRF-Token header on write requests (POST, PATCH, PUT, DELETE).
func RequireCSRF(sessionSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read-only methods don't mutate state and do not require CSRF token
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			authInfo := GetSessionInfo(r.Context())
			if authInfo == nil {
				// If not authenticated, let RequireAuth handle the 401
				next.ServeHTTP(w, r)
				return
			}

			csrfToken := r.Header.Get(auth.CSRFHeader)
			if csrfToken == "" || !auth.VerifyCSRFToken(csrfToken, authInfo.RawSessionToken, sessionSecret) {
				ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Header X-CSRF-Token tidak valid atau tidak ditemukan.", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
