package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/rbac"
	"ppo-kanban/internal/users"
)

type userHttpMockStore struct {
	mu       sync.Mutex
	users    map[string]db.User
	sessions map[string]db.Session
}

func newUserHttpMockStore() *userHttpMockStore {
	return &userHttpMockStore{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
	}
}

func (m *userHttpMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
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

func (m *userHttpMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return 1, nil
		}
	}
	return 0, nil
}

func (m *userHttpMockStore) CountUsers(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.users)), nil
}

func (m *userHttpMockStore) ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error) {
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
			CreatedAt:          u.CreatedAt,
			UpdatedAt:          u.UpdatedAt,
		})
	}
	return rows, nil
}

func (m *userHttpMockStore) ListUsersSummary(ctx context.Context) ([]db.ListUsersSummaryRow, error) {
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

func (m *userHttpMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *userHttpMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *userHttpMockStore) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.UpdateUserRow, error) {
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
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}, nil
}

func (m *userHttpMockStore) SetUserActiveStatus(ctx context.Context, arg db.SetUserActiveStatusParams) (db.SetUserActiveStatusRow, error) {
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
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}, nil
}

func (m *userHttpMockStore) ResetUserPassword(ctx context.Context, arg db.ResetUserPasswordParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(arg.ID.Bytes).String()
	u, ok := m.users[uid]
	if !ok {
		return pgx.ErrNoRows
	}
	u.PasswordHash = arg.PasswordHash
	u.MustChangePassword = true
	m.users[uid] = u
	return nil
}

func (m *userHttpMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := db.Session{
		ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
		UserID:     arg.UserID,
		TokenHash:  arg.TokenHash,
		CreatedAt:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		LastSeenAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ExpiresAt:  arg.ExpiresAt,
	}
	m.sessions[arg.TokenHash] = s
	return s, nil
}

func (m *userHttpMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
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
		UserID:             s.UserID,
		TokenHash:          s.TokenHash,
		SessionCreatedAt:   s.CreatedAt,
		LastSeenAt:         s.LastSeenAt,
		ExpiresAt:          s.ExpiresAt,
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
	}, nil
}

func (m *userHttpMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *userHttpMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
}
func (m *userHttpMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}
func (m *userHttpMockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *userHttpMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *userHttpMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error {
	return nil
}
func (m *userHttpMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *userHttpMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	return nil
}

func setupUserTestServer(t *testing.T) (http.Handler, *auth.Service, *users.Service, *userHttpMockStore, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newUserHttpMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authSvc := auth.NewService(store, secret, logger)
	userSvc := users.NewService(store, logger)

	router := NewRouter(RouterParams{
		Logger:        logger,
		AuthService:   authSvc,
		UserService:   userSvc,
		SessionSecret: secret,
		CookieSecure:  false,
	})

	return router, authSvc, userSvc, store, secret
}

// helper to create a user and log them in to get valid session cookie & csrf token
func loginAs(t *testing.T, authSvc *auth.Service, store *userHttpMockStore, email, role string) (string, string, string) {
	t.Helper()
	pass := "Str0ngPass#2026!"
	hash, err := auth.HashPassword(pass)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	uid := uuid.New()
	store.mu.Lock()
	store.users[uid.String()] = db.User{
		ID:                 pgtype.UUID{Bytes: uid, Valid: true},
		Email:              email,
		FullName:           "Test " + role,
		PasswordHash:       hash,
		Role:               db.UserRole(role),
		IsActive:           true,
		MustChangePassword: false,
	}
	store.mu.Unlock()

	res, err := authSvc.Login(context.Background(), email, pass, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to login as %s: %v", role, err)
	}

	return uid.String(), res.RawSessionToken, res.CSRFToken
}

func TestRBACMatrix_UserManagement(t *testing.T) {
	router, authSvc, _, store, _ := setupUserTestServer(t)

	// Create users for all 5 roles
	adminID, adminToken, adminCSRF := loginAs(t, authSvc, store, "admin@bank.co.id", rbac.RoleAdmin)
	_, pmToken, pmCSRF := loginAs(t, authSvc, store, "pm@bank.co.id", rbac.RoleProjectManager)
	_, tlToken, tlCSRF := loginAs(t, authSvc, store, "tl@bank.co.id", rbac.RoleTeamLeader)
	_, auditToken, auditCSRF := loginAs(t, authSvc, store, "audit@bank.co.id", rbac.RoleAudit)
	_, riskToken, riskCSRF := loginAs(t, authSvc, store, "risk@bank.co.id", rbac.RoleRiskManagement)

	roles := []struct {
		role  string
		token string
		csrf  string
	}{
		{"admin", adminToken, adminCSRF},
		{"project_manager", pmToken, pmCSRF},
		{"team_leader", tlToken, tlCSRF},
		{"audit", auditToken, auditCSRF},
		{"risk_management", riskToken, riskCSRF},
	}

	t.Run("GET /api/v1/users is allowed for all authenticated roles, but rejected for anonymous (401)", func(t *testing.T) {
		// Anonymous
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for anonymous, got %d", rec.Code)
		}

		// Authenticated roles
		for _, r := range roles {
			t.Run("Role: "+r.role, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
				req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: r.token})
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("expected 200 for %s, got %d", r.role, rec.Code)
				}
			})
		}
	})

	t.Run("Admin-only write endpoints strictly enforce RBAC (Admin=allowed, others=403, anon=401)", func(t *testing.T) {
		newUserPayload := users.CreateUserRequest{
			Email:             "newmember@bank.co.id",
			FullName:          "New Member",
			Role:              rbac.RoleTeamLeader,
			TemporaryPassword: "TempStr0ngPass#2026",
		}
		bodyBytes, _ := json.Marshal(newUserPayload)

		// Anonymous -> 401
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for anonymous on POST /users, got %d", rec.Code)
		}

		// Non-admin roles -> 403 Forbidden
		nonAdmins := roles[1:]
		for _, r := range nonAdmins {
			t.Run("Non-admin role: "+r.role, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-CSRF-Token", r.csrf)
				req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: r.token})
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if rec.Code != http.StatusForbidden {
					t.Errorf("expected 403 for non-admin %s on POST /users, got %d", r.role, rec.Code)
				}
			})
		}

		// Admin with CSRF -> 201 Created
		reqAdmin := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(bodyBytes))
		reqAdmin.Header.Set("Content-Type", "application/json")
		reqAdmin.Header.Set("X-CSRF-Token", adminCSRF)
		reqAdmin.AddCookie(&http.Cookie{Name: auth.CookieName, Value: adminToken})
		recAdmin := httptest.NewRecorder()
		router.ServeHTTP(recAdmin, reqAdmin)
		if recAdmin.Code != http.StatusCreated {
			t.Fatalf("expected 201 for admin on POST /users, got %d (%s)", recAdmin.Code, recAdmin.Body.String())
		}
	})

	t.Run("Admin cannot deactivate self (422)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+adminID+"/deactivate", nil)
		req.Header.Set("X-CSRF-Token", adminCSRF)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: adminToken})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity when admin deactivates self, got %d", rec.Code)
		}
	})

	t.Run("Admin cannot downgrade own role (422)", func(t *testing.T) {
		body, _ := json.Marshal(users.UpdateUserRequest{
			FullName: "Admin Attempting Downgrade",
			Role:     rbac.RoleTeamLeader,
		})
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+adminID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", adminCSRF)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: adminToken})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity when admin downgrades self, got %d", rec.Code)
		}
	})

	t.Run("Write requests without CSRF token are rejected (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+adminID+"/activate", nil)
		// Missing X-CSRF-Token header
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: adminToken})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for request missing CSRF token, got %d", rec.Code)
		}
	})
}
