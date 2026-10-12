package users

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/rbac"
)

type mockUserStore struct {
	mu       sync.Mutex
	users    map[string]db.User
	sessions map[string]db.Session
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
	}
}

func (m *mockUserStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := uuid.New()
	u := db.User{
		ID:                 pgtype.UUID{Bytes: uid, Valid: true},
		Email:              arg.Email,
		FullName:           arg.FullName,
		PasswordHash:       arg.PasswordHash,
		Role:               arg.Role,
		IsActive:           arg.IsActive,
		MustChangePassword: arg.MustChangePassword,
		CreatedAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.users[uid.String()] = u
	return u, nil
}

func (m *mockUserStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.users {
		if u.Email == email {
			return 1, nil
		}
	}
	return 0, nil
}

func (m *mockUserStore) CountUsers(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.users)), nil
}

func (m *mockUserStore) ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows := make([]db.ListUsersRow, 0, len(m.users))
	for _, u := range m.users {
		rows = append(rows, db.ListUsersRow{
			ID:                 u.ID,
			Email:              u.Email,
			FullName:           u.FullName,
			Role:               u.Role,
			IsActive:           u.IsActive,
			MustChangePassword: u.MustChangePassword,
			FailedLoginCount:   u.FailedLoginCount,
			LockedUntil:        u.LockedUntil,
			LastLoginAt:        u.LastLoginAt,
			CreatedAt:          u.CreatedAt,
			UpdatedAt:          u.UpdatedAt,
		})
	}
	return rows, nil
}

func (m *mockUserStore) ListUsersSummary(ctx context.Context) ([]db.ListUsersSummaryRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows := make([]db.ListUsersSummaryRow, 0)
	for _, u := range m.users {
		if u.IsActive {
			rows = append(rows, db.ListUsersSummaryRow{
				ID:       u.ID,
				FullName: u.FullName,
				Role:     u.Role,
			})
		}
	}
	return rows, nil
}

func (m *mockUserStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := uuid.UUID(id.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockUserStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *mockUserStore) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.UpdateUserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := uuid.UUID(arg.ID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return db.UpdateUserRow{}, pgx.ErrNoRows
	}
	u.FullName = arg.FullName
	u.Role = arg.Role
	u.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.users[uid] = u

	return db.UpdateUserRow{
		ID:                 u.ID,
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
		FailedLoginCount:   u.FailedLoginCount,
		LockedUntil:        u.LockedUntil,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}, nil
}

func (m *mockUserStore) SetUserActiveStatus(ctx context.Context, arg db.SetUserActiveStatusParams) (db.SetUserActiveStatusRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := uuid.UUID(arg.ID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return db.SetUserActiveStatusRow{}, pgx.ErrNoRows
	}
	u.IsActive = arg.IsActive
	u.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.users[uid] = u

	return db.SetUserActiveStatusRow{
		ID:                 u.ID,
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
		FailedLoginCount:   u.FailedLoginCount,
		LockedUntil:        u.LockedUntil,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}, nil
}

func (m *mockUserStore) ResetUserPassword(ctx context.Context, arg db.ResetUserPasswordParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := uuid.UUID(arg.ID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return pgx.ErrNoRows
	}
	u.PasswordHash = arg.PasswordHash
	u.MustChangePassword = true
	u.FailedLoginCount = 0
	u.LockedUntil = pgtype.Timestamptz{Valid: false}
	m.users[uid] = u
	return nil
}

func (m *mockUserStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

// Unused Querier methods for mockUserStore
func (m *mockUserStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
	return db.Session{}, nil
}
func (m *mockUserStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *mockUserStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
func (m *mockUserStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	return db.GetSessionWithUserRow{}, nil
}
func (m *mockUserStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *mockUserStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error {
	return nil
}
func (m *mockUserStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *mockUserStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockUserStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	return nil
}

func TestUserService(t *testing.T) {
	store := newMockUserStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(store, logger)
	ctx := context.Background()

	// Seed an Admin
	adminHash, _ := auth.HashPassword("AdminStr0ngPass#2026")
	adminID := uuid.New()
	store.users[adminID.String()] = db.User{
		ID:                 pgtype.UUID{Bytes: adminID, Valid: true},
		Email:              "admin@bank.co.id",
		FullName:           "Master Administrator",
		PasswordHash:       adminHash,
		Role:               db.UserRoleAdmin,
		IsActive:           true,
		MustChangePassword: false,
	}

	t.Run("CreateUser successfully sets must_change_password=true and active", func(t *testing.T) {
		req := CreateUserRequest{
			Email:             "pm@bank.co.id",
			FullName:          "Project Manager One",
			Role:              rbac.RoleProjectManager,
			TemporaryPassword: "TempStr0ngPass#2026",
		}
		user, err := svc.CreateUser(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error creating user: %v", err)
		}
		if user.Email != "pm@bank.co.id" {
			t.Errorf("expected email pm@bank.co.id, got %s", user.Email)
		}
		if !user.MustChangePassword {
			t.Errorf("expected MustChangePassword=true for new user")
		}
		if !user.IsActive {
			t.Errorf("expected IsActive=true for new user")
		}
		if user.Role != rbac.RoleProjectManager {
			t.Errorf("expected role project_manager, got %s", user.Role)
		}
	})

	t.Run("CreateUser rejects duplicate email", func(t *testing.T) {
		req := CreateUserRequest{
			Email:             "pm@bank.co.id",
			FullName:          "Duplicate PM",
			Role:              rbac.RoleProjectManager,
			TemporaryPassword: "TempStr0ngPass#2026",
		}
		_, err := svc.CreateUser(ctx, req)
		if err == nil || err != ErrEmailAlreadyExists {
			t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
		}
	})

	t.Run("CreateUser rejects invalid role", func(t *testing.T) {
		req := CreateUserRequest{
			Email:             "hacker@bank.co.id",
			FullName:          "Super Hacker",
			Role:              "superuser",
			TemporaryPassword: "TempStr0ngPass#2026",
		}
		_, err := svc.CreateUser(ctx, req)
		if err == nil {
			t.Fatalf("expected error for invalid role, got nil")
		}
	})

	t.Run("Admin cannot deactivate self", func(t *testing.T) {
		_, err := svc.SetUserActiveStatus(ctx, adminID.String(), adminID.String(), false)
		if err == nil || err != ErrCannotDeactivateSelf {
			t.Fatalf("expected ErrCannotDeactivateSelf, got %v", err)
		}
	})

	t.Run("Admin cannot downgrade own role", func(t *testing.T) {
		_, err := svc.UpdateUser(ctx, adminID.String(), adminID.String(), UpdateUserRequest{
			FullName: "Admin Name Changed",
			Role:     rbac.RoleTeamLeader,
		})
		if err == nil || err != ErrCannotDowngradeSelf {
			t.Fatalf("expected ErrCannotDowngradeSelf, got %v", err)
		}
	})

	t.Run("Deactivating target user revokes active sessions", func(t *testing.T) {
		// Create TL user
		tlReq := CreateUserRequest{
			Email:             "tl@bank.co.id",
			FullName:          "Team Leader One",
			Role:              rbac.RoleTeamLeader,
			TemporaryPassword: "TempStr0ngPass#2026",
		}
		tl, err := svc.CreateUser(ctx, tlReq)
		if err != nil {
			t.Fatalf("failed to create tl: %v", err)
		}

		tlUUID, _ := uuid.Parse(tl.ID)
		sessID := uuid.New().String()
		store.sessions[sessID] = db.Session{
			ID:     pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID: pgtype.UUID{Bytes: tlUUID, Valid: true},
		}

		// Deactivate TL
		_, err = svc.SetUserActiveStatus(ctx, adminID.String(), tl.ID, false)
		if err != nil {
			t.Fatalf("unexpected error deactivating tl: %v", err)
		}

		// Check session was purged
		if len(store.sessions) != 0 {
			t.Errorf("expected session to be purged upon deactivation, got %d sessions remaining", len(store.sessions))
		}
	})
}
