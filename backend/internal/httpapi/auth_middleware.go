package httpapi

import (
	"context"
	"net/http"

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
)

type contextKey string

const (
	sessionContextKey contextKey = "ppo_auth_session"
)

// SessionMiddleware extracts and verifies the session cookie on each request.
func SessionMiddleware(authService *auth.Service, cookieSecure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			authInfo, err := authService.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				// Expired or invalid session: clear cookie
				http.SetCookie(w, &http.Cookie{
					Name:     auth.CookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					Secure:   cookieSecure,
					SameSite: http.SameSiteLaxMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), sessionContextKey, authInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth enforces that a valid session exists. Returns 401 otherwise.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo := GetSessionInfo(r.Context())
		if authInfo == nil {
			ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid atau telah berakhir.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePasswordChanged blocks users with must_change_password=true from accessing normal resources.
func RequirePasswordChanged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo := GetSessionInfo(r.Context())
		if authInfo != nil && authInfo.User.MustChangePassword {
			ErrorJSON(w, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "Password wajib diganti sebelum dapat mengakses fitur lain.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetSessionInfo retrieves the authenticated session info from the request context.
func GetSessionInfo(ctx context.Context) *auth.SessionAuthInfo {
	val := ctx.Value(sessionContextKey)
	if val == nil {
		return nil
	}
	info, ok := val.(*auth.SessionAuthInfo)
	if !ok {
		return nil
	}
	return info
}

// RequireRoles enforces that the authenticated user has one of the allowed roles.
// Returns 403 if the user's role is not in the allowed list.
// Must be used after RequireAuth middleware.
func RequireRoles(allowedRoles ...db.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authInfo := GetSessionInfo(r.Context())
			if authInfo == nil {
				ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid atau telah berakhir.", nil)
				return
			}

			userRole := db.UserRole(authInfo.User.Role)
			allowed := false
			for _, role := range allowedRoles {
				if userRole == role {
					allowed = true
					break
				}
			}

			if !allowed {
				ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki izin untuk mengakses resource ini.", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
