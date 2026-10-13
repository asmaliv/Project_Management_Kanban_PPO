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
)

type httpMockStore struct {
	mu       sync.Mutex
	users    map[string]db.User
	sessions map[string]db.Session
}

func newHTTPMockStore() *httpMockStore {
	return &httpMockStore{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
	}
}

func (m *httpMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return 1, nil
		}
	}
	return 0, nil
}

func (m *httpMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
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

func (m *httpMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *httpMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	if u, ok := m.users[uid]; ok {
		return u, nil
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *httpMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
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

func (m *httpMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error {
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

func (m *httpMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
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

func (m *httpMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
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

func (m *httpMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
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

func (m *httpMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error {
	return nil
}

func (m *httpMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
}

func (m *httpMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	return nil
}

func (m *httpMockStore) DeleteExpiredSessions(ctx context.Context) error {
	return nil
}

func (m *httpMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
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
func (m *httpMockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) {
	return db.AuditLog{}, nil
}

// ListAuditLogs is a no-op for tests
func (m *httpMockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return []db.AuditLog{}, nil
}

func setupTestServer(t *testing.T) (http.Handler, *auth.Service, *httpMockStore, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newHTTPMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	
	// For tests, pass nil for audit service
	authSvc := auth.NewService(store, secret, logger, nil)

	router := NewRouter(RouterParams{
		Logger:        logger,
		AuthService:   authSvc,
		SessionSecret: secret,
		CookieSecure:  false,
	})

	return router, authSvc, store, secret
}

func TestAuthEndpoints_VerifyChecklist(t *testing.T) {
	router, authSvc, store, secret := setupTestServer(t)
	ctx := context.Background()

	rawPassword := "SecureBankPassword#2026!"
	hash, _ := auth.HashPassword(rawPassword)

	_, err := store.CreateUser(ctx, db.CreateUserParams{
		Email:              "auditor@bank.local",
		FullName:           "Internal Auditor",
		PasswordHash:       hash,
		Role:               db.UserRoleAudit,
		IsActive:           true,
		MustChangePassword: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	t.Run("Tanpa login, endpoint terproteksi -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}

		var body ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Success || body.Error == nil || body.Error.Code != "UNAUTHENTICATED" {
			t.Errorf("expected UNAUTHENTICATED error envelope, got: %+v", body)
		}
	})

	t.Run("Kredensial salah -> pesan generik", func(t *testing.T) {
		payload := []byte(`{"email":"auditor@bank.local","password":"IncorrectPassword#999"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}

		var body ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Error == nil || body.Error.Message != "Email atau password salah." {
			t.Errorf("expected generic error message, got: %+v", body.Error)
		}
	})

	t.Run("Password tidak muncul di respons", func(t *testing.T) {
		payload := []byte(`{"email":"auditor@bank.local","password":"` + rawPassword + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		rawResponse := rec.Body.String()
		if bytes.Contains([]byte(rawResponse), []byte(rawPassword)) ||
			bytes.Contains([]byte(rawResponse), []byte("argon2id")) {
			t.Errorf("password or hash leaked in response: %s", rawResponse)
		}
	})

	t.Run("Request tulis tanpa CSRF token -> ditolak (403)", func(t *testing.T) {
		// First login
		loginRes, err := authSvc.Login(ctx, "auditor@bank.local", rawPassword, "127.0.0.1", "test-agent", "test-request-id")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}

		cookie := &http.Cookie{
			Name:  auth.CookieName,
			Value: loginRes.RawSessionToken,
		}

		payload := []byte(`{"old_password":"` + rawPassword + `","new_password":"BrandNewPassword#2026","confirm_password":"BrandNewPassword#2026"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		// Omit X-CSRF-Token header
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when CSRF token is missing, got %d", rec.Code)
		}

		// Now test with invalid CSRF token
		reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBuffer(payload))
		reqInvalid.Header.Set("Content-Type", "application/json")
		reqInvalid.Header.Set(auth.CSRFHeader, "invalid-bogus-csrf-token")
		reqInvalid.AddCookie(cookie)
		recInvalid := httptest.NewRecorder()
		router.ServeHTTP(recInvalid, reqInvalid)

		if recInvalid.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden on invalid CSRF token, got %d", recInvalid.Code)
		}

		// Now test with VALID CSRF token -> 200 OK
		validCSRF := auth.GenerateCSRFToken(loginRes.RawSessionToken, secret)
		reqValid := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBuffer(payload))
		reqValid.Header.Set("Content-Type", "application/json")
		reqValid.Header.Set(auth.CSRFHeader, validCSRF)
		reqValid.AddCookie(cookie)
		recValid := httptest.NewRecorder()
		router.ServeHTTP(recValid, reqValid)

		if recValid.Code != http.StatusOK {
			t.Errorf("expected 200 with valid CSRF token, got %d, body: %s", recValid.Code, recValid.Body.String())
		}
	})
}
