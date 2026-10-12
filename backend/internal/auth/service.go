package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/db"
)

var (
	ErrInvalidCredentials     = errors.New("Email atau password salah.")
	ErrAccountLocked           = errors.New("Akun terkunci sementara, coba lagi nanti.")
	ErrSessionNotFound         = errors.New("Sesi tidak ditemukan atau sudah berakhir.")
	ErrSessionExpired          = errors.New("Sesi telah berakhir.")
	ErrPasswordMismatch        = errors.New("Konfirmasi password baru tidak cocok.")
	ErrOldPasswordIncorrect    = errors.New("Password lama tidak sesuai.")
	ErrSameOldNewPassword      = errors.New("Password baru harus berbeda dari password lama.")
	ErrPasswordChangeRequired  = errors.New("Password wajib diganti sebelum melanjutkan.")
)

type UserDTO struct {
	ID                 string    `json:"id"`
	Email              string    `json:"email"`
	FullName           string    `json:"full_name"`
	Role               string    `json:"role"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type LoginResult struct {
	User            UserDTO `json:"user"`
	CSRFToken       string  `json:"csrf_token"`
	RawSessionToken string  `json:"-"`
}

type SessionAuthInfo struct {
	SessionID       string
	RawSessionToken string
	TokenHash       string
	User            UserDTO
	IsActive        bool
}

type Service struct {
	queries       db.Querier
	sessionSecret string
	logger        *slog.Logger
	auditLog      *auditlog.Service
}

func NewService(queries db.Querier, sessionSecret string, logger *slog.Logger, auditLog *auditlog.Service) *Service {
	return &Service{
		queries:       queries,
		sessionSecret: sessionSecret,
		logger:        logger,
		auditLog:      auditLog,
	}
}

// Login authenticates a user by email and password, handling lockout and sessions.
func (s *Service) Login(ctx context.Context, email, password, ip, userAgent, requestID string) (*LoginResult, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	now := time.Now().UTC()

	// 1. Fetch user by email
	user, err := s.queries.GetUserByEmail(ctx, cleanEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Run a dummy argon2 verify to mitigate timing attacks
			_, _ = VerifyPassword("dummy-password-to-waste-time", "$argon2id$v=19$m=65536,t=1,p=4$c29tZXNhbHQxMjM0NTY3OA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGg")
			
			// Log failed login attempt with no actor
			if s.auditLog != nil {
				_ = s.auditLog.Log(ctx, auditlog.LogParams{
					ActorID:    pgtype.UUID{}, // null UUID
					ActorRole:  db.NullUserRole{},
					Action:     "auth.login_failed",
					EntityType: "user",
					EntityID:   "",
					After:      map[string]any{"email": cleanEmail, "reason": "user_not_found"},
					IP:         ip,
					RequestID:  requestID,
				})
			}
			
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	// 2. Inactive account returns generic error per requirements
	if !user.IsActive {
		s.logger.Warn("login attempt on inactive account", slog.String("email", cleanEmail))
		
		if s.auditLog != nil {
			_ = s.auditLog.Log(ctx, auditlog.LogParams{
				ActorID:    user.ID,
				ActorRole:  db.NullUserRole{UserRole: user.Role, Valid: true},
				Action:     "auth.login_failed",
				EntityType: "user",
				EntityID:   uuid.UUID(user.ID.Bytes).String(),
				After:      map[string]any{"email": cleanEmail, "reason": "account_inactive"},
				IP:         ip,
				RequestID:  requestID,
			})
		}
		
		return nil, ErrInvalidCredentials
	}

	// 3. Lockout check (5 failed attempts locks for 15 minutes)
	if user.LockedUntil.Valid && now.Before(user.LockedUntil.Time) {
		s.logger.Warn("login attempt on locked account",
			slog.String("email", cleanEmail),
			slog.Time("locked_until", user.LockedUntil.Time),
		)
		
		if s.auditLog != nil {
			_ = s.auditLog.Log(ctx, auditlog.LogParams{
				ActorID:    user.ID,
				ActorRole:  db.NullUserRole{UserRole: user.Role, Valid: true},
				Action:     "auth.login_locked",
				EntityType: "user",
				EntityID:   uuid.UUID(user.ID.Bytes).String(),
				After: map[string]any{
					"email":        cleanEmail,
					"locked_until": user.LockedUntil.Time.Format(time.RFC3339),
				},
				IP:        ip,
				RequestID: requestID,
			})
		}
		
		return nil, ErrAccountLocked
	}

	// 4. Verify password
	matched, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !matched {
		// Increment failed login count
		failedCount, incErr := s.queries.IncrementFailedLogin(ctx, user.ID)
		if incErr != nil {
			s.logger.Error("failed to increment login counter", slog.String("error", incErr.Error()))
		}

		if failedCount >= 5 {
			lockUntil := now.Add(15 * time.Minute)
			_ = s.queries.LockAccount(ctx, db.LockAccountParams{
				ID: user.ID,
				LockedUntil: pgtype.Timestamptz{
					Time:  lockUntil,
					Valid: true,
				},
			})
			s.logger.Warn("account locked after 5 failed attempts",
				slog.String("email", cleanEmail),
				slog.Time("locked_until", lockUntil),
			)
			
			if s.auditLog != nil {
				_ = s.auditLog.Log(ctx, auditlog.LogParams{
					ActorID:    user.ID,
					ActorRole:  db.NullUserRole{UserRole: user.Role, Valid: true},
					Action:     "auth.login_locked",
					EntityType: "user",
					EntityID:   uuid.UUID(user.ID.Bytes).String(),
					After: map[string]any{
						"email":         cleanEmail,
						"locked_until":  lockUntil.Format(time.RFC3339),
						"failed_count":  failedCount,
					},
					IP:        ip,
					RequestID: requestID,
				})
			}
			
			return nil, ErrAccountLocked
		}
		
		if s.auditLog != nil {
			_ = s.auditLog.Log(ctx, auditlog.LogParams{
				ActorID:    user.ID,
				ActorRole:  db.NullUserRole{UserRole: user.Role, Valid: true},
				Action:     "auth.login_failed",
				EntityType: "user",
				EntityID:   uuid.UUID(user.ID.Bytes).String(),
				After:      map[string]any{"email": cleanEmail, "reason": "wrong_password", "failed_count": failedCount},
				IP:         ip,
				RequestID:  requestID,
			})
		}

		return nil, ErrInvalidCredentials
	}

	// 5. Success: reset failed logins & touch last_login_at
	if err := s.queries.ResetFailedLoginAndTouchLastLogin(ctx, user.ID); err != nil {
		s.logger.Error("failed to reset failed login count", slog.String("error", err.Error()))
	}

	// 6. Create session (8h absolute timeout)
	rawToken, err := GenerateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}
	tokenHash := HashSessionToken(rawToken)
	expiresAt := now.Add(AbsoluteTimeout)

	session, err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Ip:        pgtype.Text{String: ip, Valid: ip != ""},
		UserAgent: pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("create session in db: %w", err)
	}

	csrfToken := GenerateCSRFToken(rawToken, s.sessionSecret)
	userUUID := uuid.UUID(user.ID.Bytes).String()
	sessionUUID := uuid.UUID(session.ID.Bytes).String()
	
	// Log successful login
	if s.auditLog != nil {
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    user.ID,
			ActorRole:  db.NullUserRole{UserRole: user.Role, Valid: true},
			Action:     "auth.login_success",
			EntityType: "session",
			EntityID:   sessionUUID,
			After: map[string]any{
				"user_id": userUUID,
				"email":   user.Email,
				"role":    string(user.Role),
			},
			IP:        ip,
			RequestID: requestID,
		})
	}

	return &LoginResult{
		User: UserDTO{
			ID:                 userUUID,
			Email:              user.Email,
			FullName:           user.FullName,
			Role:               string(user.Role),
			MustChangePassword: user.MustChangePassword,
			CreatedAt:          user.CreatedAt.Time,
			UpdatedAt:          user.UpdatedAt.Time,
		},
		CSRFToken:       csrfToken,
		RawSessionToken: rawToken,
	}, nil
}

// ValidateSession verifies session validity (timeouts, user active status) and updates last_seen_at.
func (s *Service) ValidateSession(ctx context.Context, rawSessionToken string) (*SessionAuthInfo, error) {
	if rawSessionToken == "" {
		return nil, ErrSessionNotFound
	}

	tokenHash := HashSessionToken(rawSessionToken)
	sessionRow, err := s.queries.GetSessionWithUser(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session with user: %w", err)
	}

	now := time.Now().UTC()
	expired, _ := IsSessionExpired(sessionRow.LastSeenAt.Time, sessionRow.ExpiresAt.Time, now)
	if expired {
		_ = s.queries.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, ErrSessionExpired
	}

	if !sessionRow.IsActive {
		_ = s.queries.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, ErrSessionNotFound
	}

	// Update last_seen_at for idle timeout extension
	_ = s.queries.TouchSessionLastSeen(ctx, sessionRow.SessionID)

	userUUID := uuid.UUID(sessionRow.UserID.Bytes).String()
	sessionUUID := uuid.UUID(sessionRow.SessionID.Bytes).String()

	return &SessionAuthInfo{
		SessionID:       sessionUUID,
		RawSessionToken: rawSessionToken,
		TokenHash:       tokenHash,
		User: UserDTO{
			ID:                 userUUID,
			Email:              sessionRow.Email,
			FullName:           sessionRow.FullName,
			Role:               string(sessionRow.Role),
			MustChangePassword: sessionRow.MustChangePassword,
		},
		IsActive: sessionRow.IsActive,
	}, nil
}

// Logout invalidates the active session by deleting it from the database.
func (s *Service) Logout(ctx context.Context, rawSessionToken, ip, requestID string) error {
	if rawSessionToken == "" {
		return nil
	}
	
	tokenHash := HashSessionToken(rawSessionToken)
	
	// Get session info before deletion for audit log
	sessionRow, err := s.queries.GetSessionWithUser(ctx, tokenHash)
	if err == nil && s.auditLog != nil {
		// Log logout
		userUUID := uuid.UUID(sessionRow.UserID.Bytes).String()
		sessionUUID := uuid.UUID(sessionRow.SessionID.Bytes).String()
		
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    sessionRow.UserID,
			ActorRole:  db.NullUserRole{UserRole: sessionRow.Role, Valid: true},
			Action:     "auth.logout",
			EntityType: "session",
			EntityID:   sessionUUID,
			After: map[string]any{
				"user_id": userUUID,
				"email":   sessionRow.Email,
			},
			IP:        ip,
			RequestID: requestID,
		})
	}
	
	return s.queries.DeleteSessionByTokenHash(ctx, tokenHash)
}

// GetMe returns current session details and a freshly computed CSRF token.
func (s *Service) GetMe(ctx context.Context, rawSessionToken string) (*LoginResult, error) {
	authInfo, err := s.ValidateSession(ctx, rawSessionToken)
	if err != nil {
		return nil, err
	}

	csrfToken := GenerateCSRFToken(rawSessionToken, s.sessionSecret)
	return &LoginResult{
		User:            authInfo.User,
		CSRFToken:       csrfToken,
		RawSessionToken: rawSessionToken,
	}, nil
}

// ChangePassword validates old password, checks policy, and updates password.
func (s *Service) ChangePassword(ctx context.Context, rawSessionToken, oldPassword, newPassword, confirmPassword, ip, requestID string) error {
	if newPassword != confirmPassword {
		return ErrPasswordMismatch
	}

	if oldPassword == newPassword {
		return ErrSameOldNewPassword
	}

	authInfo, err := s.ValidateSession(ctx, rawSessionToken)
	if err != nil {
		return err
	}

	// Validate against banking password policy
	if err := ValidatePasswordPolicy(newPassword, authInfo.User.Email); err != nil {
		return err
	}

	// Fetch current user password hash
	parsedUserID, err := uuid.Parse(authInfo.User.ID)
	if err != nil {
		return fmt.Errorf("parse user id: %w", err)
	}
	userUUID := pgtype.UUID{Bytes: parsedUserID, Valid: true}

	user, err := s.queries.GetUserByID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("get user for password change: %w", err)
	}

	matched, err := VerifyPassword(oldPassword, user.PasswordHash)
	if err != nil || !matched {
		return ErrOldPasswordIncorrect
	}

	// Hash new password
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	// Update in database and set must_change_password to false
	if err := s.queries.UpdatePassword(ctx, db.UpdatePasswordParams{
		ID:                 user.ID,
		PasswordHash:       newHash,
		MustChangePassword: false,
	}); err != nil {
		return fmt.Errorf("update password in db: %w", err)
	}

	s.logger.Info("password successfully changed", slog.String("user_id", authInfo.User.ID))
	
	// Log password change - NEVER include password values
	if s.auditLog != nil {
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    userUUID,
			ActorRole:  db.NullUserRole{UserRole: user.Role, Valid: true},
			Action:     "auth.password_changed",
			EntityType: "user",
			EntityID:   authInfo.User.ID,
			After: map[string]any{
				"email":                  user.Email,
				"must_change_password":   false,
			},
			IP:        ip,
			RequestID: requestID,
		})
	}
	
	return nil
}
