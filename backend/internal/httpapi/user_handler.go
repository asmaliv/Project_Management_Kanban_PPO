package httpapi

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
)

type UserHandler struct {
	queries  db.Querier
	auditLog *auditlog.Service
}

func NewUserHandler(queries db.Querier, auditLog *auditlog.Service) *UserHandler {
	return &UserHandler{
		queries:  queries,
		auditLog: auditLog,
	}
}

type UserResponseItem struct {
	ID                 string      `json:"id"`
	Username           string      `json:"username"`
	Email              string      `json:"email"`
	FullName           string      `json:"full_name"`
	Role               db.UserRole `json:"role"`
	IsActive           bool        `json:"is_active"`
	MustChangePassword bool        `json:"must_change_password"`
	CreatedAt          string      `json:"created_at"`
	UpdatedAt          string      `json:"updated_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
	Password string `json:"password"`
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]`)

func generateEmailFromName(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	parts := strings.Fields(clean)
	var handle string
	if len(parts) >= 2 {
		handle = parts[0] + "." + parts[1]
	} else if len(parts) == 1 {
		handle = parts[0]
	} else {
		handle = "user"
	}
	handle = nonAlphanumericRegex.ReplaceAllString(handle, "")
	if len(handle) < 2 {
		handle = "user"
	}
	return fmt.Sprintf("%s_%d@ppo.bank.co.id", handle, time.Now().Unix()%10000)
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal mengambil daftar pengguna.", nil)
		return
	}

	result := make([]UserResponseItem, 0, len(users))
	for _, u := range users {
		var uid string
		if u.ID.Valid {
			uid = uuid.UUID(u.ID.Bytes).String()
		}
		var createdAt, updatedAt string
		if u.CreatedAt.Valid {
			createdAt = u.CreatedAt.Time.Format(time.RFC3339)
		}
		if u.UpdatedAt.Valid {
			updatedAt = u.UpdatedAt.Time.Format(time.RFC3339)
		}

		result = append(result, UserResponseItem{
			ID:                 uid,
			Username:           strings.SplitN(u.Email, "@", 2)[0],
			Email:              u.Email,
			FullName:           u.FullName,
			Role:               u.Role,
			IsActive:           u.IsActive,
			MustChangePassword: u.MustChangePassword,
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
		})
	}

	JSON(w, http.StatusOK, result)
}

func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	var req CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format data request tidak valid.", nil)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.Username = normalizeUsername(req.Username)
	req.Role = strings.TrimSpace(req.Role)
	req.Password = strings.TrimSpace(req.Password)

	fieldErrors := make(map[string]string)
	if req.FullName == "" {
		fieldErrors["full_name"] = "Nama pengguna wajib diisi."
	}
	if req.Username == "" {
		req.Username = strings.SplitN(generateEmailFromName(req.FullName), "@", 2)[0]
	}
	if !regexp.MustCompile(`^[a-z0-9_]{3,50}$`).MatchString(req.Username) {
		fieldErrors["username"] = "Username harus 3–50 karakter dan hanya boleh berisi huruf kecil, angka, atau underscore."
	}

	var userRole db.UserRole
	switch req.Role {
	case "project_manager", "team_leader", "risk_management", "audit", "admin":
		userRole = db.UserRole(req.Role)
	default:
		fieldErrors["role"] = "Role tidak valid. Pilihan: project_manager, team_leader, risk_management, audit."
	}

	if len(req.Password) < 12 {
		fieldErrors["password"] = "Password sementara minimal 12 karakter."
	}

	if len(fieldErrors) > 0 {
		ErrorJSON(w, http.StatusBadRequest, "VALIDATION_FAILED", "Data pengguna tidak valid.", fieldErrors)
		return
	}

	email := req.Username + "@ppo.bank.co.id"

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal memproses password.", nil)
		return
	}

	newUser, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Email:              email,
		FullName:           req.FullName,
		PasswordHash:       passwordHash,
		Role:               userRole,
		IsActive:           true,
		MustChangePassword: true,
	})
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal menyimpan data pengguna ke database.", nil)
		return
	}

	var newUserID string
	if newUser.ID.Valid {
		newUserID = uuid.UUID(newUser.ID.Bytes).String()
	}

	// Audit Log
	if h.auditLog != nil {
		ip := extractIP(r)
		requestID := middleware.GetReqID(r.Context())
		var actorUUID pgtype.UUID
		if session.User.ID != "" {
			_ = actorUUID.Scan(session.User.ID)
		}
		_ = h.auditLog.Log(r.Context(), auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: db.UserRole(session.User.Role), Valid: true},
			Action:     "user.created",
			EntityType: "user",
			EntityID:   newUserID,
			After: map[string]any{
				"full_name": req.FullName,
				"username":  req.Username,
				"role":      string(userRole),
			},
			IP:        ip,
			RequestID: requestID,
		})
	}

	var createdAt, updatedAt string
	if newUser.CreatedAt.Valid {
		createdAt = newUser.CreatedAt.Time.Format(time.RFC3339)
	}
	if newUser.UpdatedAt.Valid {
		updatedAt = newUser.UpdatedAt.Time.Format(time.RFC3339)
	}

	JSON(w, http.StatusCreated, UserResponseItem{
		ID:                 newUserID,
		Username:           strings.SplitN(newUser.Email, "@", 2)[0],
		Email:              newUser.Email,
		FullName:           newUser.FullName,
		Role:               newUser.Role,
		IsActive:           newUser.IsActive,
		MustChangePassword: newUser.MustChangePassword,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	})
}

func (h *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	targetUserIDStr := chi.URLParam(r, "id")
	parsedUUID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_ID", "ID pengguna tidak valid.", nil)
		return
	}

	var targetUUID pgtype.UUID
	_ = targetUUID.Scan(parsedUUID.String())

	var req UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format data request tidak valid.", nil)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.Role = strings.TrimSpace(req.Role)

	fieldErrors := make(map[string]string)
	if req.FullName == "" {
		fieldErrors["full_name"] = "Nama pengguna tidak boleh kosong."
	}

	var userRole db.UserRole
	switch req.Role {
	case "project_manager", "team_leader", "risk_management", "audit", "admin":
		userRole = db.UserRole(req.Role)
	default:
		fieldErrors["role"] = "Role tidak valid."
	}

	if len(fieldErrors) > 0 {
		ErrorJSON(w, http.StatusBadRequest, "VALIDATION_FAILED", "Data pengguna tidak valid.", fieldErrors)
		return
	}

	updatedUser, err := h.queries.UpdateUser(r.Context(), db.UpdateUserParams{
		ID:       targetUUID,
		FullName: req.FullName,
		Role:     userRole,
		IsActive: req.IsActive,
	})
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal memperbarui data pengguna.", nil)
		return
	}

	if req.Password != "" {
		if len(req.Password) < 12 {
			ErrorJSON(w, http.StatusBadRequest, "VALIDATION_FAILED", "Data pengguna tidak valid.", map[string]string{"password": "Password baru minimal 12 karakter."})
			return
		}
		passwordHash, hashErr := auth.HashPassword(req.Password)
		if hashErr != nil {
			ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal memproses password.", nil)
			return
		}
		if updateErr := h.queries.UpdatePassword(r.Context(), db.UpdatePasswordParams{
			ID:                 targetUUID,
			PasswordHash:       passwordHash,
			MustChangePassword: true,
		}); updateErr != nil {
			ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal mereset password pengguna.", nil)
			return
		}
	}

	if h.auditLog != nil {
		ip := extractIP(r)
		requestID := middleware.GetReqID(r.Context())
		var actorUUID pgtype.UUID
		if session.User.ID != "" {
			_ = actorUUID.Scan(session.User.ID)
		}
		_ = h.auditLog.Log(r.Context(), auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: db.UserRole(session.User.Role), Valid: true},
			Action:     "user.updated",
			EntityType: "user",
			EntityID:   targetUserIDStr,
			After: map[string]any{
				"full_name":      req.FullName,
				"username":       strings.SplitN(updatedUser.Email, "@", 2)[0],
				"role":           string(userRole),
				"is_active":      req.IsActive,
				"password_reset": req.Password != "",
			},
			IP:        ip,
			RequestID: requestID,
		})
	}

	var createdAt, updatedAt string
	if updatedUser.CreatedAt.Valid {
		createdAt = updatedUser.CreatedAt.Time.Format(time.RFC3339)
	}
	if updatedUser.UpdatedAt.Valid {
		updatedAt = updatedUser.UpdatedAt.Time.Format(time.RFC3339)
	}

	JSON(w, http.StatusOK, UserResponseItem{
		ID:                 targetUserIDStr,
		Username:           strings.SplitN(updatedUser.Email, "@", 2)[0],
		Email:              updatedUser.Email,
		FullName:           updatedUser.FullName,
		Role:               updatedUser.Role,
		IsActive:           updatedUser.IsActive,
		MustChangePassword: updatedUser.MustChangePassword,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	})
}

func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	targetUserIDStr := chi.URLParam(r, "id")
	parsedUUID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_ID", "ID pengguna tidak valid.", nil)
		return
	}

	// Cegah admin menghapus akunnya sendiri
	if session.User.ID == targetUserIDStr {
		ErrorJSON(w, http.StatusBadRequest, "SELF_DELETE_FORBIDDEN", "Anda tidak dapat menghapus akun Anda sendiri.", nil)
		return
	}

	var targetUUID pgtype.UUID
	_ = targetUUID.Scan(parsedUUID.String())

	// Hapus sesi pengguna dulu jika ada
	_ = h.queries.DeleteSessionsByUserID(r.Context(), targetUUID)

	if err := h.queries.DeleteUser(r.Context(), targetUUID); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal menghapus pengguna (kemungkinan ada data terkait). Anda bisa menonaktifkan akun sebagai alternatif.", nil)
		return
	}

	if h.auditLog != nil {
		ip := extractIP(r)
		requestID := middleware.GetReqID(r.Context())
		var actorUUID pgtype.UUID
		if session.User.ID != "" {
			_ = actorUUID.Scan(session.User.ID)
		}
		_ = h.auditLog.Log(r.Context(), auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: db.UserRole(session.User.Role), Valid: true},
			Action:     "user.deleted",
			EntityType: "user",
			EntityID:   targetUserIDStr,
			IP:         ip,
			RequestID:  requestID,
		})
	}

	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
