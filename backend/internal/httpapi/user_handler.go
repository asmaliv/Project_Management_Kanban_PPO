package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"ppo-kanban/internal/rbac"
	"ppo-kanban/internal/users"
)

type UserHandler struct {
	userService *users.Service
}

func NewUserHandler(userService *users.Service) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// ListUsers godoc
// GET /api/v1/users
// Admin: returns paginated full user details with meta.
// Other authenticated roles: returns user summary list (id, full_name, role).
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	sessionInfo := GetSessionInfo(r.Context())
	if sessionInfo == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	if sessionInfo.User.Role == rbac.RoleAdmin {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		userList, total, err := h.userService.ListUsers(r.Context(), int32(page), int32(pageSize))
		if err != nil {
			ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil daftar pengguna.", nil)
			return
		}

		JSONWithMeta(w, http.StatusOK, userList, &Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    int(total),
		})
		return
	}

	// Non-admin roles: summary view
	summaryList, err := h.userService.ListUsersSummary(r.Context())
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil ringkasan pengguna.", nil)
		return
	}

	JSON(w, http.StatusOK, summaryList)
}

// CreateUser godoc
// POST /api/v1/users (Admin only)
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req users.CreateUserRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload JSON tidak valid.", nil)
		return
	}

	created, err := h.userService.CreateUser(r.Context(), req)
	if err != nil {
		if errors.Is(err, users.ErrEmailAlreadyExists) {
			ErrorJSON(w, http.StatusConflict, "EMAIL_EXISTS", err.Error(), nil)
			return
		}
		if errors.Is(err, users.ErrInvalidRole) || errors.Is(err, users.ErrInvalidInput) {
			ErrorJSON(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal membuat pengguna.", nil)
		return
	}

	JSON(w, http.StatusCreated, created)
}

// GetUserByID godoc
// GET /api/v1/users/{id} (Admin only)
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	user, err := h.userService.GetUserByID(r.Context(), idStr)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Pengguna tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, users.ErrInvalidInput) {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil data pengguna.", nil)
		return
	}

	JSON(w, http.StatusOK, user)
}

// UpdateUser godoc
// PATCH /api/v1/users/{id} (Admin only)
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	sessionInfo := GetSessionInfo(r.Context())
	if sessionInfo == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	idStr := chi.URLParam(r, "id")

	var req users.UpdateUserRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload JSON tidak valid.", nil)
		return
	}

	updated, err := h.userService.UpdateUser(r.Context(), sessionInfo.User.ID, idStr, req)
	if err != nil {
		if errors.Is(err, users.ErrCannotDowngradeSelf) {
			ErrorJSON(w, http.StatusUnprocessableEntity, "CANNOT_DOWNGRADE_SELF", err.Error(), nil)
			return
		}
		if errors.Is(err, users.ErrUserNotFound) {
			ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Pengguna tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, users.ErrInvalidRole) || errors.Is(err, users.ErrInvalidInput) {
			ErrorJSON(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal memperbarui pengguna.", nil)
		return
	}

	JSON(w, http.StatusOK, updated)
}

// DeactivateUser godoc
// POST /api/v1/users/{id}/deactivate (Admin only)
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	sessionInfo := GetSessionInfo(r.Context())
	if sessionInfo == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	idStr := chi.URLParam(r, "id")

	updated, err := h.userService.SetUserActiveStatus(r.Context(), sessionInfo.User.ID, idStr, false)
	if err != nil {
		if errors.Is(err, users.ErrCannotDeactivateSelf) {
			ErrorJSON(w, http.StatusUnprocessableEntity, "CANNOT_DEACTIVATE_SELF", err.Error(), nil)
			return
		}
		if errors.Is(err, users.ErrUserNotFound) {
			ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Pengguna tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, users.ErrInvalidInput) {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal menonaktifkan pengguna.", nil)
		return
	}

	JSON(w, http.StatusOK, updated)
}

// ActivateUser godoc
// POST /api/v1/users/{id}/activate (Admin only)
func (h *UserHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	sessionInfo := GetSessionInfo(r.Context())
	if sessionInfo == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	idStr := chi.URLParam(r, "id")

	updated, err := h.userService.SetUserActiveStatus(r.Context(), sessionInfo.User.ID, idStr, true)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Pengguna tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, users.ErrInvalidInput) {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengaktifkan pengguna.", nil)
		return
	}

	JSON(w, http.StatusOK, updated)
}

type ResetPasswordPayload struct {
	TemporaryPassword string `json:"temporary_password"`
}

// ResetPassword godoc
// POST /api/v1/users/{id}/reset-password (Admin only)
func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	var req ResetPasswordPayload
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload JSON tidak valid.", nil)
		return
	}

	err := h.userService.ResetPassword(r.Context(), idStr, req.TemporaryPassword)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Pengguna tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, users.ErrInvalidInput) {
			ErrorJSON(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal me-reset password pengguna.", nil)
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "Password berhasil di-reset. Sesi pengguna telah dicabut.",
	})
}
