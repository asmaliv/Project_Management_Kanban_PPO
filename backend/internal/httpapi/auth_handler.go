package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"ppo-kanban/internal/auth"
)

type AuthHandler struct {
	authService   *auth.Service
	sessionSecret string
	cookieSecure  bool
}

func NewAuthHandler(authService *auth.Service, sessionSecret string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		sessionSecret: sessionSecret,
		cookieSecure:  cookieSecure,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format data request tidak valid.", nil)
		return
	}

	fields := make(map[string]string)
	if req.Email == "" {
		fields["email"] = "Email wajib diisi."
	}
	if req.Password == "" {
		fields["password"] = "Password wajib diisi."
	}
	if len(fields) > 0 {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Email dan password wajib diisi.", fields)
		return
	}

	ip := extractIP(r)
	result, err := h.authService.Login(r.Context(), req.Email, req.Password, ip, r.UserAgent())
	if err != nil {
		if errors.Is(err, auth.ErrAccountLocked) {
			ErrorJSON(w, http.StatusLocked, "ACCOUNT_LOCKED", "Akun terkunci sementara karena terlalu banyak percobaan gagal. Silakan coba lagi setelah 15 menit.", nil)
			return
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Email atau password salah.", nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan internal pada server.", nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    result.RawSessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.AbsoluteTimeout.Seconds()),
	})

	JSON(w, http.StatusOK, map[string]any{
		"user":       result.User,
		"csrf_token": result.CSRFToken,
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	authInfo := GetSessionInfo(r.Context())
	if authInfo != nil {
		_ = h.authService.Logout(r.Context(), authInfo.RawSessionToken)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	JSON(w, http.StatusOK, map[string]string{
		"message": "Logout berhasil",
	})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	authInfo := GetSessionInfo(r.Context())
	if authInfo == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan.", nil)
		return
	}

	csrfToken := auth.GenerateCSRFToken(authInfo.RawSessionToken, h.sessionSecret)
	JSON(w, http.StatusOK, map[string]any{
		"user":       authInfo.User,
		"csrf_token": csrfToken,
	})
}

func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	authInfo := GetSessionInfo(r.Context())
	if authInfo == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan.", nil)
		return
	}

	var req ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format data request tidak valid.", nil)
		return
	}

	fields := make(map[string]string)
	if req.OldPassword == "" {
		fields["old_password"] = "Password lama wajib diisi."
	}
	if req.NewPassword == "" {
		fields["new_password"] = "Password baru wajib diisi."
	}
	if req.ConfirmPassword == "" {
		fields["confirm_password"] = "Konfirmasi password baru wajib diisi."
	}
	if len(fields) > 0 {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Semua kolom password wajib diisi.", fields)
		return
	}

	err := h.authService.ChangePassword(
		r.Context(),
		authInfo.RawSessionToken,
		req.OldPassword,
		req.NewPassword,
		req.ConfirmPassword,
	)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrPasswordMismatch),
			errors.Is(err, auth.ErrSameOldNewPassword),
			errors.Is(err, auth.ErrPasswordTooShort),
			errors.Is(err, auth.ErrPasswordSameAsEmail),
			errors.Is(err, auth.ErrPasswordCommon),
			errors.Is(err, auth.ErrOldPasswordIncorrect):
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		default:
			ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal memperbarui password.", nil)
		}
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "Password berhasil diubah",
	})
}

func decodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MB limit
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // Security rule from SECURITY.md section 4
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must only contain a single JSON object")
	}
	return nil
}
