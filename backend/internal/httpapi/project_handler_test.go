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
)

type projectMockStore struct {
	mu         sync.Mutex
	users      map[string]db.User
	sessions   map[string]db.Session
	projects   map[string]db.Project
	members    map[string][]string // projectID -> []userID
	columns    map[string]db.BoardColumn
	projectSeq int
}

func newProjectMockStore() *projectMockStore {
	return &projectMockStore{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
		projects: make(map[string]db.Project),
		members:  make(map[string][]string),
		columns:  make(map[string]db.BoardColumn),
	}
}

// User & Session methods
func (m *projectMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	return 0, nil
}
func (m *projectMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
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
func (m *projectMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}
func (m *projectMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}
func (m *projectMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *projectMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *projectMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *projectMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
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
func (m *projectMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
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
func (m *projectMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *projectMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
func (m *projectMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	return nil
}
func (m *projectMockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *projectMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	return nil
}

// Project & Column methods
func (m *projectMockStore) NextProjectCode(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projectSeq++
	return fmt.Sprintf("PRJ-%04d", m.projectSeq), nil
}
func (m *projectMockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	p := db.Project{
		ID:          pgtype.UUID{Bytes: id, Valid: true},
		Code:        arg.Code,
		Name:        arg.Name,
		Description: arg.Description,
		Status:      arg.Status,
		OwnerID:     arg.OwnerID,
		StartDate:   arg.StartDate,
		EndDate:     arg.EndDate,
		CreatedBy:   arg.CreatedBy,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.projects[id.String()] = p
	return p, nil
}
func (m *projectMockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	p, ok := m.projects[uid]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}
func (m *projectMockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	return m.GetProjectByID(ctx, id)
}
func (m *projectMockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []db.Project
	for _, p := range m.projects {
		if !p.DeletedAt.Valid {
			list = append(list, p)
		}
	}
	return list, nil
}
func (m *projectMockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []db.Project
	for _, p := range m.projects {
		if p.OwnerID == ownerID && !p.DeletedAt.Valid {
			list = append(list, p)
		}
	}
	return list, nil
}
func (m *projectMockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uStr := uuid.UUID(userID.Bytes).String()
	var list []db.Project
	for pid, members := range m.members {
		for _, memberID := range members {
			if memberID == uStr {
				if p, ok := m.projects[pid]; ok && !p.DeletedAt.Valid {
					list = append(list, p)
				}
			}
		}
	}
	return list, nil
}
func (m *projectMockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(arg.ID.Bytes).String()
	p, ok := m.projects[uid]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	p.Name = arg.Name
	p.Description = arg.Description
	p.Status = arg.Status
	p.StartDate = arg.StartDate
	p.EndDate = arg.EndDate
	p.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.projects[uid] = p
	return p, nil
}
func (m *projectMockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	p, ok := m.projects[uid]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	p.Status = db.ProjectStatusArchived
	p.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.projects[uid] = p
	return p, nil
}
func (m *projectMockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(arg.ID.Bytes).String()
	p, ok := m.projects[uid]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	p.OwnerID = arg.OwnerID
	p.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.projects[uid] = p
	return p, nil
}
func (m *projectMockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pStr := uuid.UUID(arg.ProjectID.Bytes).String()
	uStr := uuid.UUID(arg.UserID.Bytes).String()
	for _, id := range m.members[pStr] {
		if id == uStr {
			return nil
		}
	}
	m.members[pStr] = append(m.members[pStr], uStr)
	return nil
}
func (m *projectMockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pStr := uuid.UUID(arg.ProjectID.Bytes).String()
	uStr := uuid.UUID(arg.UserID.Bytes).String()
	existing := m.members[pStr]
	filtered := make([]string, 0, len(existing))
	for _, id := range existing {
		if id != uStr {
			filtered = append(filtered, id)
		}
	}
	m.members[pStr] = filtered
	return nil
}
func (m *projectMockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pStr := uuid.UUID(projectID.Bytes).String()
	var list []db.ListProjectMembersRow
	for _, uStr := range m.members[pStr] {
		u, ok := m.users[uStr]
		if ok {
			list = append(list, db.ListProjectMembersRow{
				ID:       u.ID,
				FullName: u.FullName,
				Email:    u.Email,
				Role:     u.Role,
				AddedAt:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			})
		}
	}
	return list, nil
}
func (m *projectMockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pStr := uuid.UUID(arg.ProjectID.Bytes).String()
	uStr := uuid.UUID(arg.UserID.Bytes).String()
	for _, id := range m.members[pStr] {
		if id == uStr {
			return true, nil
		}
	}
	return false, nil
}
func (m *projectMockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pStr := uuid.UUID(projectID.Bytes).String()
	return int64(len(m.members[pStr])), nil
}
func (m *projectMockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) {
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
		CreatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.columns[id.String()] = col
	return col, nil
}
func (m *projectMockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []db.BoardColumn
	for _, c := range m.columns {
		if c.ProjectID == projectID {
			list = append(list, c)
		}
	}
	return list, nil
}
func (m *projectMockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	col, ok := m.columns[uid]
	if !ok {
		return db.BoardColumn{}, pgx.ErrNoRows
	}
	return col, nil
}
func (m *projectMockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(arg.ID.Bytes).String()
	col, ok := m.columns[uid]
	if !ok {
		return db.BoardColumn{}, pgx.ErrNoRows
	}
	col.Name = arg.Name
	col.Position = arg.Position
	col.WipLimit = arg.WipLimit
	m.columns[uid] = col
	return col, nil
}
func (m *projectMockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	delete(m.columns, uid)
	return nil
}
func (m *projectMockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.columns {
		if c.ProjectID == projectID && c.IsDoneColumn {
			return c, nil
		}
	}
	return db.BoardColumn{}, pgx.ErrNoRows
}
func (m *projectMockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var maxPos int32
	for _, c := range m.columns {
		if c.ProjectID == projectID && c.Position > maxPos {
			maxPos = c.Position
		}
	}
	return maxPos, nil
}
func (m *projectMockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) {
	return db.AuditLog{}, nil
}
func (m *projectMockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return nil, nil
}

// ─── HTTP Test Setup ──────────────────────────────────────────────────────────

func setupProjectTestServer(t *testing.T) (http.Handler, *projectMockStore, *auth.Service, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newProjectMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authSvc := auth.NewService(store, secret, logger, nil)
	projSvc := project.NewService(nil, store, nil, logger)

	router := NewRouter(RouterParams{
		Logger:         logger,
		AuthService:    authSvc,
		ProjectService: projSvc,
		SessionSecret:  secret,
		CookieSecure:   false,
	})

	return router, store, authSvc, secret
}

func createTestUserAndSession(t *testing.T, store *projectMockStore, email, name string, role db.UserRole) (db.User, string) {
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

func addSessionCookie(req *http.Request, token string) {
	req.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	})
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestProjectHandler_RBAC_Matrix(t *testing.T) {
	router, store, _, _ := setupProjectTestServer(t)

	pm1, pm1Token := createTestUserAndSession(t, store, "pm1@bank.local", "Project Manager 1", db.UserRoleProjectManager)
	_, pm2Token := createTestUserAndSession(t, store, "pm2@bank.local", "Project Manager 2", db.UserRoleProjectManager)
	tl1, tlToken := createTestUserAndSession(t, store, "tl1@bank.local", "Team Leader 1", db.UserRoleTeamLeader)
	_, adminToken := createTestUserAndSession(t, store, "admin@bank.local", "Admin User", db.UserRoleAdmin)

	t.Run("Tanpa session -> 401 UNAUTHENTICATED", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Team Leader cannot create project -> 403 FORBIDDEN", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "TL Project"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body)
		addSessionCookie(req, tlToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	var createdProjectID string
	t.Run("PM1 creates project -> 201 Created", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "Core Banking", "description": "New system"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body)
		addSessionCookie(req, pm1Token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				ID   string `json:"id"`
				Code string `json:"code"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if res.Data.Name != "Core Banking" {
			t.Errorf("expected name 'Core Banking', got %s", res.Data.Name)
		}
		createdProjectID = res.Data.ID
	})

	t.Run("PM1 gets own project -> 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+createdProjectID, nil)
		addSessionCookie(req, pm1Token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("PM2 gets PM1's project -> 404 NOT_FOUND (prevent enumeration)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+createdProjectID, nil)
		addSessionCookie(req, pm2Token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("PM1 adds TL1 as project member -> 201 Created", func(t *testing.T) {
		tlIDStr := uuid.UUID(tl1.ID.Bytes).String()
		body := bytes.NewBufferString(fmt.Sprintf(`{"user_id": "%s"}`, tlIDStr))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+createdProjectID+"/members", body)
		addSessionCookie(req, pm1Token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("TL1 can now view project -> 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+createdProjectID, nil)
		addSessionCookie(req, tlToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("PM1 cannot transfer owner -> 403 FORBIDDEN (Admin only)", func(t *testing.T) {
		body := bytes.NewBufferString(fmt.Sprintf(`{"new_owner_id": "%s"}`, uuid.New().String()))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+createdProjectID+"/transfer-owner", body)
		addSessionCookie(req, pm1Token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Admin transfers owner to PM2 -> 200 OK", func(t *testing.T) {
		pm2User, _ := store.GetUserByEmail(context.Background(), "pm2@bank.local")
		pm2IDStr := uuid.UUID(pm2User.ID.Bytes).String()
		body := bytes.NewBufferString(fmt.Sprintf(`{"new_owner_id": "%s"}`, pm2IDStr))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+createdProjectID+"/transfer-owner", body)
		addSessionCookie(req, adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Done column deletion is rejected -> 409 CONFLICT", func(t *testing.T) {
		// List columns to find Done column
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+createdProjectID+"/columns", nil)
		addSessionCookie(req, pm2Token) // PM2 is now owner
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var res struct {
			Data []project.ColumnDTO `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)

		var doneColID string
		for _, col := range res.Data {
			if col.IsDoneColumn {
				doneColID = col.ID
			}
		}

		if doneColID == "" {
			t.Fatal("no done column found")
		}

		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/columns/"+doneColID, nil)
		addSessionCookie(delReq, pm2Token)
		delRec := httptest.NewRecorder()
		router.ServeHTTP(delRec, delReq)

		if delRec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict when deleting Done column, got %d. Body: %s", delRec.Code, delRec.Body.String())
		}
	})

	t.Run("PM2 archives project, then update is rejected -> 403 FORBIDDEN", func(t *testing.T) {
		// Archive
		archReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+createdProjectID+"/archive", nil)
		addSessionCookie(archReq, pm2Token)
		archRec := httptest.NewRecorder()
		router.ServeHTTP(archRec, archReq)

		if archRec.Code != http.StatusOK {
			t.Fatalf("expected 200 on archive, got %d", archRec.Code)
		}

		// Update should fail with 403 FORBIDDEN
		updateBody := bytes.NewBufferString(`{"name": "Changed Name"}`)
		updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/"+createdProjectID, updateBody)
		addSessionCookie(updateReq, pm2Token)
		updateRec := httptest.NewRecorder()
		router.ServeHTTP(updateRec, updateReq)

		if updateRec.Code != http.StatusForbidden {
			t.Errorf("expected 403 on updating archived project, got %d", updateRec.Code)
		}
	})
	_ = pm1
}
