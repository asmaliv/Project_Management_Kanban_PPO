package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/rbac"
)

var (
	ErrUserNotFound         = errors.New("Pengguna tidak ditemukan.")
	ErrEmailAlreadyExists   = errors.New("Email sudah terdaftar dalam sistem.")
	ErrCannotDeactivateSelf = errors.New("Admin tidak dapat menonaktifkan akun sendiri.")
	ErrCannotDowngradeSelf  = errors.New("Admin tidak dapat menurunkan peran (role) akun sendiri.")
	ErrInvalidRole          = errors.New("Peran pengguna tidak valid.")
	ErrInvalidInput         = errors.New("Input data pengguna tidak valid.")
)

type UserDetailDTO struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	FullName           string     `json:"full_name"`
	Role               string     `json:"role"`
	IsActive           bool       `json:"is_active"`
	MustChangePassword bool       `json:"must_change_password"`
	FailedLoginCount   int32      `json:"failed_login_count"`
	LockedUntil        *time.Time `json:"locked_until,omitempty"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UserSummaryDTO struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type CreateUserRequest struct {
	Email             string `json:"email"`
	FullName          string `json:"full_name"`
	Role              string `json:"role"`
	TemporaryPassword string `json:"temporary_password"`
}

type UpdateUserRequest struct {
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type Service struct {
	queries db.Querier
	logger  *slog.Logger
}

func NewService(queries db.Querier, logger *slog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger,
	}
}

// CreateUser handles administrative creation of a new user account with temporary credentials.
func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (*UserDetailDTO, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || len(email) > 255 {
		return nil, fmt.Errorf("%w: email wajib diisi dan maksimal 255 karakter", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("%w: format email tidak valid", ErrInvalidInput)
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" || len(fullName) > 255 {
		return nil, fmt.Errorf("%w: nama lengkap wajib diisi dan maksimal 255 karakter", ErrInvalidInput)
	}

	role := strings.TrimSpace(req.Role)
	if !rbac.IsValidRole(role) {
		return nil, fmt.Errorf("%w: '%s'", ErrInvalidRole, role)
	}

	if err := auth.ValidatePasswordPolicy(req.TemporaryPassword, email); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	// Check email uniqueness
	count, err := s.queries.CountUsersByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa ketersediaan email: %w", err)
	}
	if count > 0 {
		return nil, ErrEmailAlreadyExists
	}

	passwordHash, err := auth.HashPassword(req.TemporaryPassword)
	if err != nil {
		return nil, fmt.Errorf("gagal melakukan hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:              email,
		FullName:           fullName,
		PasswordHash:       passwordHash,
		Role:               db.UserRole(role),
		IsActive:           true,
		MustChangePassword: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat user di database: %w", err)
	}

	s.logger.Info("user created by admin",
		slog.String("user_id", uuid.UUID(user.ID.Bytes).String()),
		slog.String("email", user.Email),
		slog.String("role", string(user.Role)),
	)

	dto := toUserDetailDTO(user)
	return &dto, nil
}

// ListUsers returns paginated full user details for administrators.
func (s *Service) ListUsers(ctx context.Context, page, pageSize int32) ([]UserDetailDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := s.queries.ListUsers(ctx, db.ListUsersParams{
		Limit:  pageSize,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("gagal mengambil daftar pengguna: %w", err)
	}

	total, err := s.queries.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung total pengguna: %w", err)
	}

	result := make([]UserDetailDTO, 0, len(rows))
	for _, r := range rows {
		var lockedUntil *time.Time
		if r.LockedUntil.Valid {
			t := r.LockedUntil.Time
			lockedUntil = &t
		}
		var lastLoginAt *time.Time
		if r.LastLoginAt.Valid {
			t := r.LastLoginAt.Time
			lastLoginAt = &t
		}

		result = append(result, UserDetailDTO{
			ID:                 uuid.UUID(r.ID.Bytes).String(),
			Email:              r.Email,
			FullName:           r.FullName,
			Role:               string(r.Role),
			IsActive:           r.IsActive,
			MustChangePassword: r.MustChangePassword,
			FailedLoginCount:   r.FailedLoginCount,
			LockedUntil:        lockedUntil,
			LastLoginAt:        lastLoginAt,
			CreatedAt:          r.CreatedAt.Time,
			UpdatedAt:          r.UpdatedAt.Time,
		})
	}

	return result, total, nil
}

// ListUsersSummary returns active user summaries (id, full_name, role) for non-admin roles.
func (s *Service) ListUsersSummary(ctx context.Context) ([]UserSummaryDTO, error) {
	rows, err := s.queries.ListUsersSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil ringkasan pengguna: %w", err)
	}

	result := make([]UserSummaryDTO, 0, len(rows))
	for _, r := range rows {
		result = append(result, UserSummaryDTO{
			ID:       uuid.UUID(r.ID.Bytes).String(),
			FullName: r.FullName,
			Role:     string(r.Role),
		})
	}
	return result, nil
}

// GetUserByID returns full user details for a specific user.
func (s *Service) GetUserByID(ctx context.Context, idStr string) (*UserDetailDTO, error) {
	parsedUUID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("%w: id tidak valid", ErrInvalidInput)
	}

	user, err := s.queries.GetUserByID(ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data pengguna: %w", err)
	}

	dto := toUserDetailDTO(user)
	return &dto, nil
}

// UpdateUser modifies full name and role of an existing user.
func (s *Service) UpdateUser(ctx context.Context, callerIDStr, targetIDStr string, req UpdateUserRequest) (*UserDetailDTO, error) {
	callerUUID, err := uuid.Parse(callerIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: caller id tidak valid", ErrInvalidInput)
	}
	targetUUID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: target id tidak valid", ErrInvalidInput)
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" || len(fullName) > 255 {
		return nil, fmt.Errorf("%w: nama lengkap wajib diisi dan maksimal 255 karakter", ErrInvalidInput)
	}

	role := strings.TrimSpace(req.Role)
	if !rbac.IsValidRole(role) {
		return nil, fmt.Errorf("%w: '%s'", ErrInvalidRole, role)
	}

	// Security Rule: Admin cannot downgrade their own role
	if callerUUID == targetUUID && role != rbac.RoleAdmin {
		return nil, ErrCannotDowngradeSelf
	}

	targetPGUUID := pgtype.UUID{Bytes: targetUUID, Valid: true}

	existingUser, err := s.queries.GetUserByID(ctx, targetPGUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data pengguna: %w", err)
	}

	updated, err := s.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:       targetPGUUID,
		FullName: fullName,
		Role:     db.UserRole(role),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui data pengguna: %w", err)
	}

	// If role changed, revoke all active sessions of this user so new privileges take effect immediately
	if string(existingUser.Role) != role {
		_ = s.queries.DeleteSessionsByUserID(ctx, targetPGUUID)
		s.logger.Info("user role changed; active sessions revoked",
			slog.String("target_id", targetIDStr),
			slog.String("old_role", string(existingUser.Role)),
			slog.String("new_role", role),
		)
	}

	var lockedUntil *time.Time
	if updated.LockedUntil.Valid {
		t := updated.LockedUntil.Time
		lockedUntil = &t
	}
	var lastLoginAt *time.Time
	if updated.LastLoginAt.Valid {
		t := updated.LastLoginAt.Time
		lastLoginAt = &t
	}

	return &UserDetailDTO{
		ID:                 uuid.UUID(updated.ID.Bytes).String(),
		Email:              updated.Email,
		FullName:           updated.FullName,
		Role:               string(updated.Role),
		IsActive:           updated.IsActive,
		MustChangePassword: updated.MustChangePassword,
		FailedLoginCount:   updated.FailedLoginCount,
		LockedUntil:        lockedUntil,
		LastLoginAt:        lastLoginAt,
		CreatedAt:          updated.CreatedAt.Time,
		UpdatedAt:          updated.UpdatedAt.Time,
	}, nil
}

// SetUserActiveStatus activates or deactivates an account. Deactivating terminates all active sessions.
func (s *Service) SetUserActiveStatus(ctx context.Context, callerIDStr, targetIDStr string, isActive bool) (*UserDetailDTO, error) {
	callerUUID, err := uuid.Parse(callerIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: caller id tidak valid", ErrInvalidInput)
	}
	targetUUID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: target id tidak valid", ErrInvalidInput)
	}

	// Security Rule: Admin cannot deactivate self
	if callerUUID == targetUUID && !isActive {
		return nil, ErrCannotDeactivateSelf
	}

	targetPGUUID := pgtype.UUID{Bytes: targetUUID, Valid: true}

	updated, err := s.queries.SetUserActiveStatus(ctx, db.SetUserActiveStatusParams{
		ID:       targetPGUUID,
		IsActive: isActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("gagal mengubah status aktif pengguna: %w", err)
	}

	// If deactivating, invalidate all sessions immediately
	if !isActive {
		_ = s.queries.DeleteSessionsByUserID(ctx, targetPGUUID)
		s.logger.Info("user deactivated; all active sessions revoked",
			slog.String("target_id", targetIDStr),
		)
	}

	var lockedUntil *time.Time
	if updated.LockedUntil.Valid {
		t := updated.LockedUntil.Time
		lockedUntil = &t
	}
	var lastLoginAt *time.Time
	if updated.LastLoginAt.Valid {
		t := updated.LastLoginAt.Time
		lastLoginAt = &t
	}

	return &UserDetailDTO{
		ID:                 uuid.UUID(updated.ID.Bytes).String(),
		Email:              updated.Email,
		FullName:           updated.FullName,
		Role:               string(updated.Role),
		IsActive:           updated.IsActive,
		MustChangePassword: updated.MustChangePassword,
		FailedLoginCount:   updated.FailedLoginCount,
		LockedUntil:        lockedUntil,
		LastLoginAt:        lastLoginAt,
		CreatedAt:          updated.CreatedAt.Time,
		UpdatedAt:          updated.UpdatedAt.Time,
	}, nil
}

// ResetPassword resets a user's password with a new temporary password and revokes existing sessions.
func (s *Service) ResetPassword(ctx context.Context, targetIDStr, newTempPassword string) error {
	targetUUID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return fmt.Errorf("%w: id tidak valid", ErrInvalidInput)
	}

	targetPGUUID := pgtype.UUID{Bytes: targetUUID, Valid: true}

	user, err := s.queries.GetUserByID(ctx, targetPGUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("gagal mengambil data pengguna: %w", err)
	}

	if err := auth.ValidatePasswordPolicy(newTempPassword, user.Email); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	passwordHash, err := auth.HashPassword(newTempPassword)
	if err != nil {
		return fmt.Errorf("gagal hash password: %w", err)
	}

	err = s.queries.ResetUserPassword(ctx, db.ResetUserPasswordParams{
		ID:           targetPGUUID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("gagal reset password di database: %w", err)
	}

	// Revoke active sessions so user must login with new temp password and change it
	_ = s.queries.DeleteSessionsByUserID(ctx, targetPGUUID)

	s.logger.Info("user password reset by admin; sessions revoked",
		slog.String("target_id", targetIDStr),
		slog.String("email", user.Email),
	)

	return nil
}

func toUserDetailDTO(u db.User) UserDetailDTO {
	var lockedUntil *time.Time
	if u.LockedUntil.Valid {
		t := u.LockedUntil.Time
		lockedUntil = &t
	}
	var lastLoginAt *time.Time
	if u.LastLoginAt.Valid {
		t := u.LastLoginAt.Time
		lastLoginAt = &t
	}

	return UserDetailDTO{
		ID:                 uuid.UUID(u.ID.Bytes).String(),
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               string(u.Role),
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
		FailedLoginCount:   u.FailedLoginCount,
		LockedUntil:        lockedUntil,
		LastLoginAt:        lastLoginAt,
		CreatedAt:          u.CreatedAt.Time,
		UpdatedAt:          u.UpdatedAt.Time,
	}
}
