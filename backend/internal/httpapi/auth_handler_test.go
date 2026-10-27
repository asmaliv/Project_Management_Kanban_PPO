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

// Phase 4 Querier stubs for auth tests
func (m *httpMockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error { return nil }
func (m *httpMockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *httpMockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) { return 0, nil }
func (m *httpMockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpMockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpMockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpMockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpMockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpMockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *httpMockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *httpMockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) { return false, nil }
func (m *httpMockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) { return nil, nil }
func (m *httpMockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) { return nil, nil }
func (m *httpMockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) { return nil, nil }
func (m *httpMockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) { return nil, nil }
func (m *httpMockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) { return nil, nil }
func (m *httpMockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpMockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-0001", nil }
func (m *httpMockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error { return nil }
func (m *httpMockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpMockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpMockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) { return db.Project{}, nil }

func (m *httpMockStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpMockStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpMockStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) { return 0, nil }
func (m *httpMockStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *httpMockStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) { return db.TaskComment{}, nil }
func (m *httpMockStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) { return db.Task{}, pgx.ErrNoRows }
func (m *httpMockStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) { return db.GetTaskByIDWithDetailsRow{}, pgx.ErrNoRows }
func (m *httpMockStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) { return nil, nil }
func (m *httpMockStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) { return nil, nil }
func (m *httpMockStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) { return nil, nil }
func (m *httpMockStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) { return nil, nil }
func (m *httpMockStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpMockStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error { return nil }
func (m *httpMockStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error { return nil }
func (m *httpMockStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error { return nil }
func (m *httpMockStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error { return nil }
func (m *httpMockStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) { return db.Task{}, nil }
func (m *httpMockStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *httpMockStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) { return db.Task{}, nil }

// Phase 6 Querier stubs
func (m *httpMockStore) AssessRisk(ctx context.Context, arg db.AssessRiskParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpMockStore) CloseRisk(ctx context.Context, id pgtype.UUID) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpMockStore) CountRisksByTaskIDs(ctx context.Context, taskIds []pgtype.UUID) ([]db.CountRisksByTaskIDsRow, error) { return nil, nil }
func (m *httpMockStore) CountRisksFiltered(ctx context.Context, arg db.CountRisksFilteredParams) (int64, error) { return 0, nil }
func (m *httpMockStore) CreateRisk(ctx context.Context, arg db.CreateRiskParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpMockStore) GetNextRiskCode(ctx context.Context) (string, error) { return "RSK-0001", nil }
func (m *httpMockStore) GetRiskByID(ctx context.Context, id pgtype.UUID) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpMockStore) GetRiskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetRiskByIDWithDetailsRow, error) { return db.GetRiskByIDWithDetailsRow{}, nil }
func (m *httpMockStore) GetRisksHeatmapCounts(ctx context.Context, arg db.GetRisksHeatmapCountsParams) ([]db.GetRisksHeatmapCountsRow, error) { return nil, nil }
func (m *httpMockStore) GetRisksSummaryStats(ctx context.Context, arg db.GetRisksSummaryStatsParams) (db.GetRisksSummaryStatsRow, error) { return db.GetRisksSummaryStatsRow{}, nil }
func (m *httpMockStore) ListRisksFiltered(ctx context.Context, arg db.ListRisksFilteredParams) ([]db.ListRisksFilteredRow, error) { return nil, nil }
func (m *httpMockStore) UpdateRiskByRiskManagement(ctx context.Context, arg db.UpdateRiskByRiskManagementParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpMockStore) UpdateRiskMitigationPlan(ctx context.Context, arg db.UpdateRiskMitigationPlanParams) (db.Risk, error) { return db.Risk{}, nil }

// Phase 7 Querier stubs
func (m *httpMockStore) CloseAuditFinding(ctx context.Context, arg db.CloseAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpMockStore) CountAuditFindingsFiltered(ctx context.Context, arg db.CountAuditFindingsFilteredParams) (int64, error) { return 0, nil }
func (m *httpMockStore) CreateAuditFinding(ctx context.Context, arg db.CreateAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpMockStore) GetAuditFindingByID(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpMockStore) GetAuditFindingByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetAuditFindingByIDWithDetailsRow, error) { return db.GetAuditFindingByIDWithDetailsRow{}, nil }
func (m *httpMockStore) GetAuditFindingsSummaryStats(ctx context.Context, arg db.GetAuditFindingsSummaryStatsParams) (db.GetAuditFindingsSummaryStatsRow, error) { return db.GetAuditFindingsSummaryStatsRow{}, nil }
func (m *httpMockStore) GetNextFindingCode(ctx context.Context) (string, error) { return "FND-0001", nil }
func (m *httpMockStore) ListAuditFindingsFiltered(ctx context.Context, arg db.ListAuditFindingsFilteredParams) ([]db.ListAuditFindingsFilteredRow, error) { return nil, nil }
func (m *httpMockStore) ReopenAuditFinding(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpMockStore) RespondAuditFinding(ctx context.Context, arg db.RespondAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpMockStore) GetUserStats(ctx context.Context) (db.GetUserStatsRow, error) { return db.GetUserStatsRow{}, nil }
func (m *httpMockStore) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) { return nil, nil }

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
