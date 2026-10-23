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

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/project"
	"ppo-kanban/internal/task"
)

type taskMockStore struct {
	mu           sync.Mutex
	users        map[string]db.User
	sessions     map[string]db.Session
	projects     map[string]db.Project
	members      map[string][]string // projectID -> []userID
	columns      map[string]db.BoardColumn
	tasks        map[string]db.Task
	taskComments map[string]db.TaskComment
}

func newTaskMockStore() *taskMockStore {
	return &taskMockStore{
		users:        make(map[string]db.User),
		sessions:     make(map[string]db.Session),
		projects:     make(map[string]db.Project),
		members:      make(map[string][]string),
		columns:      make(map[string]db.BoardColumn),
		tasks:        make(map[string]db.Task),
		taskComments: make(map[string]db.TaskComment),
	}
}

// ─── db.Querier implementation ───────────────────────────────────────────────

func (m *taskMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	return 0, nil
}
func (m *taskMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
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
func (m *taskMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}
func (m *taskMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}
func (m *taskMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *taskMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *taskMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *taskMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
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
func (m *taskMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[tokenHash]
	if !ok {
		return db.GetSessionWithUserRow{}, pgx.ErrNoRows
	}
	u, ok := m.users[uuid.UUID(s.UserID.Bytes).String()]
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
func (m *taskMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *taskMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
func (m *taskMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	return nil
}
func (m *taskMockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *taskMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	return nil
}
func (m *taskMockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-0001", nil }
func (m *taskMockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	p := db.Project{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Code:      arg.Code,
		Name:      arg.Name,
		Status:    arg.Status,
		OwnerID:   arg.OwnerID,
		CreatedBy: arg.CreatedBy,
	}
	m.projects[id.String()] = p
	return p, nil
}
func (m *taskMockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuid.UUID(id.Bytes).String()]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}
func (m *taskMockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}
func (m *taskMockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
	return nil, nil
}
func (m *taskMockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
	return nil, nil
}
func (m *taskMockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) { return nil, nil }
func (m *taskMockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) {
	return db.Project{}, nil
}
func (m *taskMockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	return db.Project{}, nil
}
func (m *taskMockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) {
	return db.Project{}, nil
}
func (m *taskMockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(arg.ProjectID.Bytes).String()
	uID := uuid.UUID(arg.UserID.Bytes).String()
	members := m.members[pID]
	for _, id := range members {
		if id == uID {
			return true, nil
		}
	}
	return false, nil
}
func (m *taskMockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(arg.ProjectID.Bytes).String()
	uID := uuid.UUID(arg.UserID.Bytes).String()
	m.members[pID] = append(m.members[pID], uID)
	return nil
}
func (m *taskMockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error {
	return nil
}
func (m *taskMockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) {
	return nil, nil
}
func (m *taskMockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) {
	return 0, nil
}
func (m *taskMockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	col := db.BoardColumn{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		ProjectID:    arg.ProjectID,
		Name:         arg.Name,
		Position:     arg.Position,
		IsDoneColumn: arg.IsDoneColumn,
		WipLimit:     arg.WipLimit,
	}
	m.columns[id.String()] = col
	return col, nil
}
func (m *taskMockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(projectID.Bytes).String()
	var cols []db.BoardColumn
	for _, c := range m.columns {
		if uuid.UUID(c.ProjectID.Bytes).String() == pID {
			cols = append(cols, c)
		}
	}
	return cols, nil
}
func (m *taskMockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	col, ok := m.columns[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.BoardColumn{}, pgx.ErrNoRows
	}
	return col, nil
}
func (m *taskMockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) {
	return db.BoardColumn{}, nil
}
func (m *taskMockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *taskMockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(projectID.Bytes).String()
	for _, col := range m.columns {
		if uuid.UUID(col.ProjectID.Bytes).String() == pID && col.IsDoneColumn {
			return col, nil
		}
	}
	return db.BoardColumn{}, pgx.ErrNoRows
}
func (m *taskMockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *taskMockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) {
	return db.AuditLog{}, nil
}
func (m *taskMockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return nil, nil
}

// Phase 5 Task queries
func (m *taskMockStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	colStr := uuid.UUID(columnID.Bytes).String()
	var count int32
	for _, t := range m.tasks {
		if uuid.UUID(t.ColumnID.Bytes).String() == colStr && !t.DeletedAt.Valid {
			count++
		}
	}
	return count, nil
}
func (m *taskMockStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *taskMockStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.tasks)), nil
}
func (m *taskMockStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	t := db.Task{
		ID:              pgtype.UUID{Bytes: id, Valid: true},
		ProjectID:       arg.ProjectID,
		ColumnID:        arg.ColumnID,
		Title:           arg.Title,
		Description:     arg.Description,
		Priority:        arg.Priority,
		AssigneeID:      arg.AssigneeID,
		ReporterID:      arg.ReporterID,
		DueDate:         arg.DueDate,
		ProgressPercent: arg.ProgressPercent,
		Position:        arg.Position,
		Labels:          arg.Labels,
		Version:         1,
		CreatedAt:       pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.tasks[id.String()] = t
	return t, nil
}
func (m *taskMockStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	c := db.TaskComment{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		TaskID:    arg.TaskID,
		AuthorID:  arg.AuthorID,
		Body:      arg.Body,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.taskComments[id.String()] = c
	return c, nil
}
func (m *taskMockStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[uuid.UUID(id.Bytes).String()]
	if !ok || t.DeletedAt.Valid {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}
func (m *taskMockStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[uuid.UUID(id.Bytes).String()]
	if !ok || t.DeletedAt.Valid {
		return db.GetTaskByIDWithDetailsRow{}, pgx.ErrNoRows
	}

	p := m.projects[uuid.UUID(t.ProjectID.Bytes).String()]
	c := m.columns[uuid.UUID(t.ColumnID.Bytes).String()]

	var assigneeName pgtype.Text
	if t.AssigneeID.Valid {
		u := m.users[uuid.UUID(t.AssigneeID.Bytes).String()]
		assigneeName = pgtype.Text{String: u.FullName, Valid: true}
	}
	reporter := m.users[uuid.UUID(t.ReporterID.Bytes).String()]

	return db.GetTaskByIDWithDetailsRow{
		ID:              t.ID,
		ProjectID:       t.ProjectID,
		ColumnID:        t.ColumnID,
		Title:           t.Title,
		Description:     t.Description,
		Priority:        t.Priority,
		AssigneeID:      t.AssigneeID,
		ReporterID:      t.ReporterID,
		DueDate:         t.DueDate,
		ProgressPercent: t.ProgressPercent,
		Position:        t.Position,
		Labels:          t.Labels,
		Version:         t.Version,
		CompletedAt:     t.CompletedAt,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		DeletedAt:       t.DeletedAt,
		AssigneeName:    assigneeName,
		ReporterName:    reporter.FullName,
		ColumnName:      c.Name,
		IsDoneColumn:    c.IsDoneColumn,
		WipLimit:        c.WipLimit,
		ProjectCode:     p.Code,
		ProjectName:     p.Name,
		ProjectOwnerID:  p.OwnerID,
	}, nil
}
func (m *taskMockStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tID := uuid.UUID(taskID.Bytes).String()
	var rows []db.ListTaskCommentsRow
	for _, c := range m.taskComments {
		if uuid.UUID(c.TaskID.Bytes).String() == tID {
			u := m.users[uuid.UUID(c.AuthorID.Bytes).String()]
			rows = append(rows, db.ListTaskCommentsRow{
				ID:         c.ID,
				TaskID:     c.TaskID,
				AuthorID:   c.AuthorID,
				Body:       c.Body,
				CreatedAt:  c.CreatedAt,
				AuthorName: u.FullName,
				AuthorRole: u.Role,
			})
		}
	}
	return rows, nil
}
func (m *taskMockStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) {
	return nil, nil
}
func (m *taskMockStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(projectID.Bytes).String()
	var rows []db.ListTasksByProjectRow
	for _, t := range m.tasks {
		if uuid.UUID(t.ProjectID.Bytes).String() == pID && !t.DeletedAt.Valid {
			rows = append(rows, db.ListTasksByProjectRow{
				ID:              t.ID,
				ProjectID:       t.ProjectID,
				ColumnID:        t.ColumnID,
				Title:           t.Title,
				Description:     t.Description,
				Priority:        t.Priority,
				AssigneeID:      t.AssigneeID,
				ReporterID:      t.ReporterID,
				DueDate:         t.DueDate,
				ProgressPercent: t.ProgressPercent,
				Position:        t.Position,
				Labels:          t.Labels,
				Version:         t.Version,
				CompletedAt:     t.CompletedAt,
				CreatedAt:       t.CreatedAt,
				UpdatedAt:       t.UpdatedAt,
			})
		}
	}
	return rows, nil
}
func (m *taskMockStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rows []db.ListTasksByProjectsFilteredRow
	for _, t := range m.tasks {
		if !t.DeletedAt.Valid {
			p := m.projects[uuid.UUID(t.ProjectID.Bytes).String()]
			c := m.columns[uuid.UUID(t.ColumnID.Bytes).String()]
			u := m.users[uuid.UUID(t.ReporterID.Bytes).String()]
			rows = append(rows, db.ListTasksByProjectsFilteredRow{
				ID:              t.ID,
				ProjectID:       t.ProjectID,
				ColumnID:        t.ColumnID,
				Title:           t.Title,
				Description:     t.Description,
				Priority:        t.Priority,
				AssigneeID:      t.AssigneeID,
				ReporterID:      t.ReporterID,
				DueDate:         t.DueDate,
				ProgressPercent: t.ProgressPercent,
				Position:        t.Position,
				Labels:          t.Labels,
				Version:         t.Version,
				CompletedAt:     t.CompletedAt,
				CreatedAt:       t.CreatedAt,
				UpdatedAt:       t.UpdatedAt,
				ColumnName:      c.Name,
				ProjectCode:     p.Code,
				ProjectName:     p.Name,
				ReporterName:    u.FullName,
			})
		}
	}
	return rows, nil
}
func (m *taskMockStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	colStr := uuid.UUID(columnID.Bytes).String()
	var maxPos int32
	for _, t := range m.tasks {
		if uuid.UUID(t.ColumnID.Bytes).String() == colStr && !t.DeletedAt.Valid && t.Position > maxPos {
			maxPos = t.Position
		}
	}
	return maxPos, nil
}
func (m *taskMockStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error {
	return nil
}
func (m *taskMockStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error {
	return nil
}
func (m *taskMockStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error {
	return nil
}
func (m *taskMockStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error {
	return nil
}
func (m *taskMockStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	t, ok := m.tasks[idStr]
	if !ok || t.DeletedAt.Valid {
		return db.Task{}, pgx.ErrNoRows
	}
	t.DeletedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.tasks[idStr] = t
	return t, nil
}
func (m *taskMockStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(arg.ID.Bytes).String()
	t, ok := m.tasks[idStr]
	if !ok || t.DeletedAt.Valid || t.Version != arg.Version {
		return db.Task{}, pgx.ErrNoRows
	}
	t.Title = arg.Title
	t.Description = arg.Description
	t.Priority = arg.Priority
	t.AssigneeID = arg.AssigneeID
	t.DueDate = arg.DueDate
	t.ProgressPercent = arg.ProgressPercent
	t.Labels = arg.Labels
	t.Version++
	m.tasks[idStr] = t
	return t, nil
}
func (m *taskMockStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(arg.ID.Bytes).String()
	t, ok := m.tasks[idStr]
	if !ok || t.DeletedAt.Valid || t.Version != arg.Version {
		return db.Task{}, pgx.ErrNoRows
	}
	t.ColumnID = arg.ColumnID
	t.Position = arg.Position
	t.CompletedAt = arg.CompletedAt
	t.Version++
	m.tasks[idStr] = t
	return t, nil
}

// ─── HTTP Test Setup Helper ──────────────────────────────────────────────────

func setupTaskHTTPServer(t *testing.T) (http.Handler, *taskMockStore, *auth.Service, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newTaskMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authSvc := auth.NewService(store, secret, logger, nil)
	projSvc := project.NewService(nil, store, nil, logger)
	taskSvc := task.NewService(nil, store, nil, logger)

	router := NewRouter(RouterParams{
		Logger:         logger,
		AuthService:    authSvc,
		ProjectService: projSvc,
		TaskService:    taskSvc,
		SessionSecret:  secret,
		CookieSecure:   false,
	})

	return router, store, authSvc, secret
}

func createTaskTestUser(t *testing.T, store *taskMockStore, email, name string, role db.UserRole) (db.User, string) {
	t.Helper()
	ctx := context.Background()
	hash, _ := auth.HashPassword("TestSecretPassword#123")
	user, err := store.CreateUser(ctx, db.CreateUserParams{
		Email:              email,
		FullName:           name,
		PasswordHash:       hash,
		Role:               role,
		IsActive:           true,
		MustChangePassword: false,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	token := uuid.New().String()
	tokenHash := auth.HashSessionToken(token)
	_, err = store.CreateSession(ctx, db.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	return user, token
}

// ─── HTTP Handler Tests ──────────────────────────────────────────────────────

func TestHTTP_CreateTask_RBAC(t *testing.T) {
	handler, store, _, _ := setupTaskHTTPServer(t)

	pm, pmToken := createTaskTestUser(t, store, "pm@bank.com", "Project Manager", db.UserRoleProjectManager)
	tl, tlToken := createTaskTestUser(t, store, "tl@bank.com", "Team Leader", db.UserRoleTeamLeader)
	_, auditToken := createTaskTestUser(t, store, "audit@bank.com", "Audit Officer", db.UserRoleAudit)

	ctx := context.Background()
	p, _ := store.CreateProject(ctx, db.CreateProjectParams{
		Code:      "PRJ-0001",
		Name:      "Core Banking Migration",
		Status:    db.ProjectStatusActive,
		OwnerID:   pm.ID,
		CreatedBy: pm.ID,
	})
	col, _ := store.CreateBoardColumn(ctx, db.CreateBoardColumnParams{
		ProjectID:    p.ID,
		Name:         "Backlog",
		Position:     1,
		IsDoneColumn: false,
	})
	_ = store.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: p.ID,
		UserID:    tl.ID,
	})

	url := fmt.Sprintf("/api/v1/projects/%s/tasks", uuid.UUID(p.ID.Bytes).String())
	reqBody := []byte(fmt.Sprintf(`{"column_id":"%s","title":"Database Indexing"}`, uuid.UUID(col.ID.Bytes).String()))

	// 1. Anonymous -> 401
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anon, got %d", rec.Code)
	}

	// 2. PM Owner -> 201 Created
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	addSessionCookie(req, pmToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for PM owner, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. TL Member -> 201 Created
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	addSessionCookie(req, tlToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for TL member, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Audit Role -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	addSessionCookie(req, auditToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Audit role, got %d", rec.Code)
	}
}

func TestHTTP_MoveTask_MakerChecker(t *testing.T) {
	handler, store, _, _ := setupTaskHTTPServer(t)

	pm, pmToken := createTaskTestUser(t, store, "pm@bank.com", "Project Manager", db.UserRoleProjectManager)
	tl, tlToken := createTaskTestUser(t, store, "tl@bank.com", "Team Leader", db.UserRoleTeamLeader)

	ctx := context.Background()
	p, _ := store.CreateProject(ctx, db.CreateProjectParams{
		Code:      "PRJ-0001",
		Name:      "Payment Gateway",
		Status:    db.ProjectStatusActive,
		OwnerID:   pm.ID,
		CreatedBy: pm.ID,
	})
	colTodo, _ := store.CreateBoardColumn(ctx, db.CreateBoardColumnParams{
		ProjectID:    p.ID,
		Name:         "To Do",
		Position:     1,
		IsDoneColumn: false,
	})
	colDone, _ := store.CreateBoardColumn(ctx, db.CreateBoardColumnParams{
		ProjectID:    p.ID,
		Name:         "Done",
		Position:     2,
		IsDoneColumn: true,
	})
	_ = store.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: p.ID,
		UserID:    tl.ID,
	})

	createdTask, _ := store.CreateTask(ctx, db.CreateTaskParams{
		ProjectID:   p.ID,
		ColumnID:    colTodo.ID,
		Title:       "QRIS integration",
		ReporterID:  pm.ID,
		Position:    1,
		Description: "",
		Priority:    db.TaskPriorityHigh,
	})

	url := fmt.Sprintf("/api/v1/tasks/%s/move", uuid.UUID(createdTask.ID.Bytes).String())
	doneColID := uuid.UUID(colDone.ID.Bytes).String()
	moveBody := []byte(fmt.Sprintf(`{"to_column_id":"%s","position":1,"version":1}`, doneColID))

	// 1. Team Leader attempts to move task to Done -> 403 Forbidden
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(moveBody))
	addSessionCookie(req, tlToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for TL moving to Done, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Project Manager moves task to Done -> 200 OK
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(moveBody))
	addSessionCookie(req, pmToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for PM moving to Done, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Stale version (attempting with version: 1 after it changed to 2) -> 409 Conflict
	staleBody := []byte(fmt.Sprintf(`{"to_column_id":"%s","position":1,"version":1}`, doneColID))
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(staleBody))
	addSessionCookie(req, pmToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for stale version, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTTP_TaskComments(t *testing.T) {
	handler, store, _, _ := setupTaskHTTPServer(t)

	pm, pmToken := createTaskTestUser(t, store, "pm@bank.com", "Project Manager", db.UserRoleProjectManager)
	tl, tlToken := createTaskTestUser(t, store, "tl@bank.com", "Team Leader", db.UserRoleTeamLeader)

	ctx := context.Background()
	p, _ := store.CreateProject(ctx, db.CreateProjectParams{
		Code:      "PRJ-0001",
		Name:      "Compliance Audit System",
		Status:    db.ProjectStatusActive,
		OwnerID:   pm.ID,
		CreatedBy: pm.ID,
	})
	col, _ := store.CreateBoardColumn(ctx, db.CreateBoardColumnParams{
		ProjectID:    p.ID,
		Name:         "Review",
		Position:     1,
		IsDoneColumn: false,
	})
	_ = store.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: p.ID,
		UserID:    tl.ID,
	})

	taskObj, _ := store.CreateTask(ctx, db.CreateTaskParams{
		ProjectID:   p.ID,
		ColumnID:    col.ID,
		Title:       "Data retention policy",
		ReporterID:  pm.ID,
		Position:    1,
		Description: "",
		Priority:    db.TaskPriorityMedium,
	})

	url := fmt.Sprintf("/api/v1/tasks/%s/comments", uuid.UUID(taskObj.ID.Bytes).String())

	// 1. TL adds comment -> 201 Created
	commentBody := []byte(`{"body":"Verified against regulatory checklist."}`)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(commentBody))
	addSessionCookie(req, tlToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for create comment, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. PM reads comments -> 200 OK with 1 comment
	req = httptest.NewRequest(http.MethodGet, url, nil)
	addSessionCookie(req, pmToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for list comments, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool             `json:"success"`
		Data    []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 comment, got %d", len(resp.Data))
	}
}
