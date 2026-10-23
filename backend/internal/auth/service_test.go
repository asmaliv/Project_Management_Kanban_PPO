package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/db"
)

type mockStore struct {
	mu       sync.Mutex
	users    map[string]db.User
	sessions map[string]db.Session
}

func newMockStore() *mockStore {
	return &mockStore{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
	}
}

func (m *mockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return 1, nil
		}
	}
	return 0, nil
}

func (m *mockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	u := db.User{
		ID:                 pgtype.UUID{Bytes: id, Valid: true},
		Email:              arg.Email,
		FullName:           arg.FullName,
		PasswordHash:       arg.PasswordHash,
		Role:               arg.Role,
		IsActive:           arg.IsActive,
		MustChangePassword: arg.MustChangePassword,
		CreatedAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.users[id.String()] = u
	return u, nil
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *mockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	if u, ok := m.users[uid]; ok {
		return u, nil
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *mockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return 0, pgx.ErrNoRows
	}
	u.FailedLoginCount++
	m.users[uid] = u
	return u.FailedLoginCount, nil
}

func (m *mockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(arg.ID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return pgx.ErrNoRows
	}
	u.LockedUntil = arg.LockedUntil
	m.users[uid] = u
	return nil
}

func (m *mockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return pgx.ErrNoRows
	}
	u.FailedLoginCount = 0
	u.LockedUntil = pgtype.Timestamptz{}
	u.LastLoginAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.users[uid] = u
	return nil
}

func (m *mockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	s := db.Session{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		UserID:     arg.UserID,
		TokenHash:  arg.TokenHash,
		CreatedAt:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		LastSeenAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ExpiresAt:  arg.ExpiresAt,
		Ip:         arg.Ip,
		UserAgent:  arg.UserAgent,
	}
	m.sessions[arg.TokenHash] = s
	return s, nil
}

func (m *mockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[tokenHash]
	if !ok {
		return db.GetSessionWithUserRow{}, pgx.ErrNoRows
	}
	uid := uuid.UUID(s.UserID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return db.GetSessionWithUserRow{}, pgx.ErrNoRows
	}

	return db.GetSessionWithUserRow{
		SessionID:          s.ID,
		UserID:             u.ID,
		TokenHash:          s.TokenHash,
		SessionCreatedAt:   s.CreatedAt,
		LastSeenAt:         s.LastSeenAt,
		ExpiresAt:          s.ExpiresAt,
		Ip:                 s.Ip,
		UserAgent:          s.UserAgent,
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
	}, nil
}

func (m *mockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, s := range m.sessions {
		if s.ID == id {
			s.LastSeenAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
			m.sessions[k] = s
			return nil
		}
	}
	return nil
}

func (m *mockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
}

func (m *mockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

func (m *mockStore) DeleteExpiredSessions(ctx context.Context) error {
	return nil
}

func (m *mockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(arg.ID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return pgx.ErrNoRows
	}
	u.PasswordHash = arg.PasswordHash
	u.MustChangePassword = arg.MustChangePassword
	m.users[uid] = u
	return nil
}

// InsertAuditLog is a no-op for tests
func (m *mockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) {
	return db.AuditLog{}, nil
}

// ListAuditLogs is a no-op for tests
func (m *mockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return []db.AuditLog{}, nil
}

// Phase 4 Querier stubs for auth tests
func (m *mockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error { return nil }
func (m *mockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *mockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) { return 0, nil }
func (m *mockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *mockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *mockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *mockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) { return false, nil }
func (m *mockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) { return nil, nil }
func (m *mockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) { return nil, nil }
func (m *mockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) { return nil, nil }
func (m *mockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) { return nil, nil }
func (m *mockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) { return nil, nil }
func (m *mockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) { return 0, nil }
func (m *mockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-0001", nil }
func (m *mockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error { return nil }
func (m *mockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) { return db.Project{}, nil }
func (m *mockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) { return db.Project{}, nil }

func setupTestService(t *testing.T) (*Service, *mockStore) {
	t.Helper()
	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	
	// For tests, we use a nil auditlog service and modify NewService to handle it
	svc := NewService(store, "test-session-secret-key-32bytes!!", logger, nil)
	return svc, store
}

func TestAuthService_Login(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	hash, err := HashPassword("CorrectPassword#2026")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user, err := store.CreateUser(ctx, db.CreateUserParams{
		Email:              "banker@bank.local",
		FullName:           "Banker John",
		PasswordHash:       hash,
		Role:               db.UserRoleAdmin,
		IsActive:           true,
		MustChangePassword: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	t.Run("successful login", func(t *testing.T) {
		res, err := svc.Login(ctx, "banker@bank.local", "CorrectPassword#2026", "127.0.0.1", "test-agent", "test-request-id")
		if err != nil {
			t.Fatalf("expected login success, got: %v", err)
		}
		if res.User.Email != "banker@bank.local" {
			t.Errorf("expected email banker@bank.local, got %s", res.User.Email)
		}
		if res.CSRFToken == "" || res.RawSessionToken == "" {
			t.Errorf("csrf and session token should be populated")
		}
		if !res.User.MustChangePassword {
			t.Errorf("expected must_change_password to be true")
		}
	})

	t.Run("wrong password returns generic error", func(t *testing.T) {
		_, err := svc.Login(ctx, "banker@bank.local", "WrongPassword#999", "127.0.0.1", "test-agent", "test-request-id")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("non-existent email returns generic error", func(t *testing.T) {
		_, err := svc.Login(ctx, "unknown@bank.local", "WrongPassword#999", "127.0.0.1", "test-agent", "test-request-id")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("inactive user returns generic error", func(t *testing.T) {
		_, _ = store.CreateUser(ctx, db.CreateUserParams{
			Email:              "inactive@bank.local",
			FullName:           "Inactive User",
			PasswordHash:       hash,
			Role:               db.UserRoleTeamLeader,
			IsActive:           false,
			MustChangePassword: false,
		})

		_, err := svc.Login(ctx, "inactive@bank.local", "CorrectPassword#2026", "127.0.0.1", "test-agent", "test-request-id")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("5 consecutive failed logins locks account for 15 minutes", func(t *testing.T) {
		// Reset user failed login counter first
		_ = store.ResetFailedLoginAndTouchLastLogin(ctx, user.ID)

		for i := 1; i <= 4; i++ {
			_, err := svc.Login(ctx, "banker@bank.local", "WrongPassword#999", "127.0.0.1", "test-agent", "test-request-id")
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("attempt %d: expected ErrInvalidCredentials, got: %v", i, err)
			}
		}

		// 5th attempt triggers lockout
		_, err := svc.Login(ctx, "banker@bank.local", "WrongPassword#999", "127.0.0.1", "test-agent", "test-request-id")
		if !errors.Is(err, ErrAccountLocked) {
			t.Fatalf("5th attempt: expected ErrAccountLocked, got: %v", err)
		}

		// Subsequent attempt with even correct password is now blocked
		_, err = svc.Login(ctx, "banker@bank.local", "CorrectPassword#2026", "127.0.0.1", "test-agent", "test-request-id")
		if !errors.Is(err, ErrAccountLocked) {
			t.Fatalf("subsequent attempt: expected ErrAccountLocked, got: %v", err)
		}
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	oldPlain := "InitialPassword#2026"
	hash, _ := HashPassword(oldPlain)

	_, err := store.CreateUser(ctx, db.CreateUserParams{
		Email:              "pm@bank.local",
		FullName:           "Project Manager",
		PasswordHash:       hash,
		Role:               db.UserRoleProjectManager,
		IsActive:           true,
		MustChangePassword: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	loginRes, err := svc.Login(ctx, "pm@bank.local", oldPlain, "127.0.0.1", "test-agent", "test-request-id")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	t.Run("rejects mismatched confirm password", func(t *testing.T) {
		err := svc.ChangePassword(ctx, loginRes.RawSessionToken, oldPlain, "NewValidPassword#2026", "DifferentPassword#2026", "127.0.0.1", "test-request-id")
		if !errors.Is(err, ErrPasswordMismatch) {
			t.Errorf("expected ErrPasswordMismatch, got: %v", err)
		}
	})

	t.Run("rejects incorrect old password", func(t *testing.T) {
		err := svc.ChangePassword(ctx, loginRes.RawSessionToken, "WrongOldPass#999", "NewValidPassword#2026", "NewValidPassword#2026", "127.0.0.1", "test-request-id")
		if !errors.Is(err, ErrOldPasswordIncorrect) {
			t.Errorf("expected ErrOldPasswordIncorrect, got: %v", err)
		}
	})

	t.Run("successful change password updates must_change_password to false", func(t *testing.T) {
		newPlain := "BrandNewSecurePassword#2026!"
		err := svc.ChangePassword(ctx, loginRes.RawSessionToken, oldPlain, newPlain, newPlain, "127.0.0.1", "test-request-id")
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}

		// Re-login with new password
		newLogin, err := svc.Login(ctx, "pm@bank.local", newPlain, "127.0.0.1", "test-agent", "test-request-id")
		if err != nil {
			t.Fatalf("login with new password failed: %v", err)
		}
		if newLogin.User.MustChangePassword {
			t.Errorf("must_change_password should now be false")
		}
	})
}
