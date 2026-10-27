package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/finding"
)

type httpFindingMockStore struct {
	mu         sync.Mutex
	users      map[string]db.User
	sessions   map[string]db.Session
	projects   map[string]db.Project
	members    map[string][]string
	tasks      map[string]db.Task
	findings   map[string]db.AuditFinding
	findingSeq int
}

func newHTTPFindingMockStore() *httpFindingMockStore {
	return &httpFindingMockStore{
		users:      make(map[string]db.User),
		sessions:   make(map[string]db.Session),
		projects:   make(map[string]db.Project),
		members:    make(map[string][]string),
		tasks:      make(map[string]db.Task),
		findings:   make(map[string]db.AuditFinding),
		findingSeq: 1,
	}
}

// ─── db.Querier implementation for httpFindingMockStore ──────────────────────

func (m *httpFindingMockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error { return nil }
func (m *httpFindingMockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error)     { return db.Project{}, nil }
func (m *httpFindingMockStore) AssessRisk(ctx context.Context, arg db.AssessRiskParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpFindingMockStore) CloseRisk(ctx context.Context, id pgtype.UUID) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpFindingMockStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpFindingMockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) { return 0, nil }
func (m *httpFindingMockStore) CountRisksByTaskIDs(ctx context.Context, taskIds []pgtype.UUID) ([]db.CountRisksByTaskIDsRow, error) { return nil, nil }
func (m *httpFindingMockStore) CountRisksFiltered(ctx context.Context, arg db.CountRisksFilteredParams) (int64, error) { return 0, nil }
func (m *httpFindingMockStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpFindingMockStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) { return 0, nil }
func (m *httpFindingMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) { return 0, nil }
func (m *httpFindingMockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpFindingMockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpFindingMockStore) CreateRisk(ctx context.Context, arg db.CreateRiskParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpFindingMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) { return db.Session{}, nil }
func (m *httpFindingMockStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *httpFindingMockStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) { return db.TaskComment{}, nil }
func (m *httpFindingMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) { return db.User{}, nil }
func (m *httpFindingMockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpFindingMockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *httpFindingMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error { return nil }
func (m *httpFindingMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error { return nil }
func (m *httpFindingMockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpFindingMockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpFindingMockStore) GetNextRiskCode(ctx context.Context) (string, error) { return "RSK-0001", nil }
func (m *httpFindingMockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *httpFindingMockStore) GetRiskByID(ctx context.Context, id pgtype.UUID) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpFindingMockStore) GetRiskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetRiskByIDWithDetailsRow, error) { return db.GetRiskByIDWithDetailsRow{}, nil }
func (m *httpFindingMockStore) GetRisksHeatmapCounts(ctx context.Context, arg db.GetRisksHeatmapCountsParams) ([]db.GetRisksHeatmapCountsRow, error) { return nil, nil }
func (m *httpFindingMockStore) GetRisksSummaryStats(ctx context.Context, arg db.GetRisksSummaryStatsParams) (db.GetRisksSummaryStatsRow, error) { return db.GetRisksSummaryStatsRow{}, nil }
func (m *httpFindingMockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) { return nil, nil }
func (m *httpFindingMockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return []db.AuditLog{
		{
			ID:          1,
			OccurredAt:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			Action:      "finding.create",
			EntityType:  "audit_finding",
			EntityID:    "f-123",
			ActorRole:   db.NullUserRole{UserRole: db.UserRoleAudit, Valid: true},
		},
	}, nil
}
func (m *httpFindingMockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) { return nil, nil }
func (m *httpFindingMockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) { return nil, nil }
func (m *httpFindingMockStore) ListRisksFiltered(ctx context.Context, arg db.ListRisksFilteredParams) ([]db.ListRisksFilteredRow, error) { return nil, nil }
func (m *httpFindingMockStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) { return nil, nil }
func (m *httpFindingMockStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) { return nil, nil }
func (m *httpFindingMockStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) { return nil, nil }
func (m *httpFindingMockStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) { return nil, nil }
func (m *httpFindingMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *httpFindingMockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpFindingMockStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpFindingMockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-001", nil }
func (m *httpFindingMockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error { return nil }
func (m *httpFindingMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpFindingMockStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error { return nil }
func (m *httpFindingMockStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error { return nil }
func (m *httpFindingMockStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error { return nil }
func (m *httpFindingMockStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error { return nil }
func (m *httpFindingMockStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) { return db.Task{}, nil }
func (m *httpFindingMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpFindingMockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpFindingMockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpFindingMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error { return nil }
func (m *httpFindingMockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpFindingMockStore) UpdateRiskByRiskManagement(ctx context.Context, arg db.UpdateRiskByRiskManagementParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpFindingMockStore) UpdateRiskMitigationPlan(ctx context.Context, arg db.UpdateRiskMitigationPlanParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *httpFindingMockStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *httpFindingMockStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) { return db.Task{}, nil }

// Auth & Session methods
func (m *httpFindingMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[uuidString(id)]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *httpFindingMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *httpFindingMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	return 0, nil
}

func (m *httpFindingMockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) {
	return db.AuditLog{ID: 1}, nil
}

func (m *httpFindingMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[tokenHash]
	if !ok {
		return db.GetSessionWithUserRow{}, pgx.ErrNoRows
	}
	u, ok := m.users[uuidString(sess.UserID)]
	if !ok {
		return db.GetSessionWithUserRow{}, pgx.ErrNoRows
	}
	return db.GetSessionWithUserRow{
		SessionID:          sess.ID,
		UserID:             sess.UserID,
		TokenHash:          tokenHash,
		SessionCreatedAt:   sess.CreatedAt,
		LastSeenAt:         sess.LastSeenAt,
		ExpiresAt:          sess.ExpiresAt,
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
	}, nil
}

// Project & Finding methods
func (m *httpFindingMockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuidString(id)]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *httpFindingMockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members := m.members[uuidString(arg.ProjectID)]
	uID := uuidString(arg.UserID)
	for _, id := range members {
		if id == uID {
			return true, nil
		}
	}
	return false, nil
}

func (m *httpFindingMockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.Project
	for _, p := range m.projects {
		if uuidString(p.OwnerID) == uuidString(ownerID) {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *httpFindingMockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.Project
	uID := uuidString(userID)
	for projID, members := range m.members {
		for _, mID := range members {
			if mID == uID {
				if p, ok := m.projects[projID]; ok {
					res = append(res, p)
				}
				break
			}
		}
	}
	return res, nil
}

func (m *httpFindingMockStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[uuidString(id)]
	if !ok {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (m *httpFindingMockStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) {
	return db.GetTaskByIDWithDetailsRow{}, nil
}

func (m *httpFindingMockStore) GetNextFindingCode(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	code := fmt.Sprintf("FND-%04d", m.findingSeq)
	m.findingSeq++
	return code, nil
}

func (m *httpFindingMockStore) CreateAuditFinding(ctx context.Context, arg db.CreateAuditFindingParams) (db.AuditFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fID := uuid.New()
	f := db.AuditFinding{
		ID:          pgtype.UUID{Bytes: fID, Valid: true},
		Code:        arg.Code,
		ProjectID:   arg.ProjectID,
		TaskID:      arg.TaskID,
		Title:       arg.Title,
		Description: arg.Description,
		Severity:    arg.Severity,
		Status:      arg.Status,
		RaisedBy:    arg.RaisedBy,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.findings[fID.String()] = f
	return f, nil
}

func (m *httpFindingMockStore) GetAuditFindingByID(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(id)]
	if !ok {
		return db.AuditFinding{}, pgx.ErrNoRows
	}
	return f, nil
}

func (m *httpFindingMockStore) GetAuditFindingByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetAuditFindingByIDWithDetailsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(id)]
	if !ok {
		return db.GetAuditFindingByIDWithDetailsRow{}, pgx.ErrNoRows
	}
	p := m.projects[uuidString(f.ProjectID)]
	uRaised := m.users[uuidString(f.RaisedBy)]
	return db.GetAuditFindingByIDWithDetailsRow{
		ID:             f.ID,
		Code:           f.Code,
		ProjectID:      f.ProjectID,
		TaskID:         f.TaskID,
		Title:          f.Title,
		Description:    f.Description,
		Severity:       f.Severity,
		Status:         f.Status,
		RaisedBy:       f.RaisedBy,
		Response:       f.Response,
		CreatedAt:      f.CreatedAt,
		UpdatedAt:      f.UpdatedAt,
		ProjectName:    p.Name,
		ProjectCode:    p.Code,
		ProjectOwnerID: p.OwnerID,
		RaisedByName:   uRaised.FullName,
	}, nil
}

func (m *httpFindingMockStore) RespondAuditFinding(ctx context.Context, arg db.RespondAuditFindingParams) (db.AuditFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(arg.ID)]
	if !ok {
		return db.AuditFinding{}, pgx.ErrNoRows
	}
	f.Response = arg.Response
	f.RespondedBy = arg.RespondedBy
	f.RespondedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	f.Status = db.FindingStatusResponded
	f.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.findings[uuidString(arg.ID)] = f
	return f, nil
}

func (m *httpFindingMockStore) ReopenAuditFinding(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(id)]
	if !ok {
		return db.AuditFinding{}, pgx.ErrNoRows
	}
	f.Status = db.FindingStatusOpen
	f.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.findings[uuidString(id)] = f
	return f, nil
}

func (m *httpFindingMockStore) CloseAuditFinding(ctx context.Context, arg db.CloseAuditFindingParams) (db.AuditFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(arg.ID)]
	if !ok {
		return db.AuditFinding{}, pgx.ErrNoRows
	}
	f.Status = db.FindingStatusClosed
	f.ClosedBy = arg.ClosedBy
	f.ClosedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	f.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.findings[uuidString(arg.ID)] = f
	return f, nil
}

func (m *httpFindingMockStore) ListAuditFindingsFiltered(ctx context.Context, arg db.ListAuditFindingsFilteredParams) ([]db.ListAuditFindingsFilteredRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.ListAuditFindingsFilteredRow
	for _, f := range m.findings {
		p := m.projects[uuidString(f.ProjectID)]
		u := m.users[uuidString(f.RaisedBy)]
		res = append(res, db.ListAuditFindingsFilteredRow{
			ID:           f.ID,
			Code:         f.Code,
			ProjectID:    f.ProjectID,
			Title:        f.Title,
			Severity:     f.Severity,
			Status:       f.Status,
			ProjectName:  p.Name,
			ProjectCode:  p.Code,
			RaisedByName: u.FullName,
		})
	}
	return res, nil
}

func (m *httpFindingMockStore) CountAuditFindingsFiltered(ctx context.Context, arg db.CountAuditFindingsFilteredParams) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.findings)), nil
}

func (m *httpFindingMockStore) GetAuditFindingsSummaryStats(ctx context.Context, arg db.GetAuditFindingsSummaryStatsParams) (db.GetAuditFindingsSummaryStatsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return db.GetAuditFindingsSummaryStatsRow{
		TotalFindings: int64(len(m.findings)),
	}, nil
}

func (m *httpFindingMockStore) GetUserStats(ctx context.Context) (db.GetUserStatsRow, error) { return db.GetUserStatsRow{}, nil }
func (m *httpFindingMockStore) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) { return nil, nil }

func parseUUIDHelper(s string) pgtype.UUID {
	u, _ := uuid.Parse(s)
	return pgtype.UUID{Bytes: u, Valid: true}
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// ─── HTTP Setup Helper ───────────────────────────────────────────────────────

func setupFindingHTTPServer(t *testing.T) (http.Handler, *httpFindingMockStore, *auth.Service, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newHTTPFindingMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	auditSvc := auditlog.NewService(nil, logger)
	authSvc := auth.NewService(store, secret, logger, nil)
	findingSvc := finding.NewService(nil, store, nil, logger)

	router := NewRouter(RouterParams{
		Logger:         logger,
		AuthService:    authSvc,
		AuditService:   auditSvc,
		FindingService: findingSvc,
		SessionSecret:  secret,
		CookieSecure:   false,
	})

	return router, store, authSvc, secret
}

func createFindingUserSession(t *testing.T, store *httpFindingMockStore, authSvc *auth.Service, role db.UserRole) (*http.Cookie, string) {
	t.Helper()
	uID := uuid.New().String()
	userUUID := parseUUIDHelper(uID)

	store.users[uID] = db.User{
		ID:          userUUID,
		Email:       fmt.Sprintf("%s_%s@bankppo.test", role, uID[:6]),
		FullName:    fmt.Sprintf("User %s", role),
		Role:        role,
		IsActive:    true,
	}

	token := "token-" + uID
	tokenHash := auth.HashSessionToken(token)

	store.sessions[tokenHash] = db.Session{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		UserID:    userUUID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(8 * time.Hour), Valid: true},
		LastSeenAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	cookie := &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	}

	return cookie, uID
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestFindingHTTP_RBAC(t *testing.T) {
	handler, store, authSvc, _ := setupFindingHTTPServer(t)

	auditCookie, auditUID := createFindingUserSession(t, store, authSvc, db.UserRoleAudit)
	pmCookie, pmUID := createFindingUserSession(t, store, authSvc, db.UserRoleProjectManager)
	tlCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleTeamLeader)
	rmCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleRiskManagement)

	projID := uuid.New().String()
	store.projects[projID] = db.Project{
		ID:      parseUUIDHelper(projID),
		Name:    "Core Banking Audit",
		Code:    "CBA-01",
		OwnerID: parseUUIDHelper(pmUID),
		Status:  db.ProjectStatusActive,
	}

	// 1. Anonim cannot access findings -> 401
	t.Run("Anonim -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/findings", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	// 2. Non-audit cannot create finding -> 403
	t.Run("PM cannot create finding -> 403", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"project_id": projID,
			"title":      "Unauthorized finding",
			"severity":   "high",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/findings", bytes.NewReader(body))
		req.AddCookie(pmCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	// 3. Auditor can create finding -> 201
	var findingID string
	t.Run("Auditor creates finding -> 201", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"project_id": projID,
			"title":      "Audit trail gap",
			"severity":   "critical",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/findings", bytes.NewReader(body))
		req.AddCookie(auditCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		var resp ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp.Data.(map[string]any)
		findingID = data["id"].(string)
	})

	// 4. Non-PM cannot respond to finding -> 403
	t.Run("TL and Auditor cannot respond -> 403", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"response": "Fixed"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/findings/"+findingID+"/respond", bytes.NewReader(body))
		req.AddCookie(tlCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}

		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/findings/"+findingID+"/respond", bytes.NewReader(body))
		req2.AddCookie(auditCookie)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec2.Code)
		}
	})

	// 5. Auditor cannot close unresponded finding directly -> 409 Conflict
	t.Run("Auditor cannot close open finding -> 409", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/findings/"+findingID+"/close", nil)
		req.AddCookie(auditCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}
	})

	// 6. PM owner responds -> 200
	t.Run("PM owner responds -> 200", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"response": "Issue addressed in PR-42"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/findings/"+findingID+"/respond", bytes.NewReader(body))
		req.AddCookie(pmCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	// 7. Auditor closes responded finding -> 200
	t.Run("Auditor closes responded finding -> 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/findings/"+findingID+"/close", nil)
		req.AddCookie(auditCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	// 8. CSV Export rate limit & permissions
	t.Run("Non-audit cannot export audit logs -> 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
		req.AddCookie(pmCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for PM exporting audit logs, got %d", rec.Code)
		}
	})

	t.Run("Auditor exports audit logs -> 200 CSV", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
		req.AddCookie(auditCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		contentType := rec.Header().Get("Content-Type")
		if !bytes.Contains([]byte(contentType), []byte("text/csv")) {
			t.Errorf("expected text/csv header, got %s", contentType)
		}
	})

	_ = auditUID
	_ = rmCookie
}
