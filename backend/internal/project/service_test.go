package project

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/db"
)

type mockStore struct {
	mu           sync.Mutex
	projects     map[string]db.Project
	members      map[string][]string // projectID -> []userID
	columns      map[string]db.BoardColumn
	users        map[string]db.User
	projectSeq   int
}

func newMockStore() *mockStore {
	return &mockStore{
		projects: make(map[string]db.Project),
		members:  make(map[string][]string),
		columns:  make(map[string]db.BoardColumn),
		users:    make(map[string]db.User),
	}
}

func (m *mockStore) NextProjectCode(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projectSeq++
	return fmt.Sprintf("PRJ-%04d", m.projectSeq), nil
}

func (m *mockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
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

func (m *mockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	p, ok := m.projects[uid]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *mockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	p, ok := m.projects[uid]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *mockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) {
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

func (m *mockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
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

func (m *mockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
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

func (m *mockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) {
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

func (m *mockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) {
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

func (m *mockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) {
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

func (m *mockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error {
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

func (m *mockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error {
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

func (m *mockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) {
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

func (m *mockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
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

func (m *mockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pStr := uuid.UUID(projectID.Bytes).String()
	return int64(len(m.members[pStr])), nil
}

func (m *mockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) {
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

func (m *mockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) {
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

func (m *mockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	col, ok := m.columns[uid]
	if !ok {
		return db.BoardColumn{}, pgx.ErrNoRows
	}
	return col, nil
}

func (m *mockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) {
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

func (m *mockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid := uuid.UUID(id.Bytes).String()
	delete(m.columns, uid)
	return nil
}

func (m *mockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.columns {
		if c.ProjectID == projectID && c.IsDoneColumn {
			return c, nil
		}
	}
	return db.BoardColumn{}, pgx.ErrNoRows
}

func (m *mockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) {
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

// Stubs for remaining Querier methods
func (m *mockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	return 0, nil
}
func (m *mockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
	return db.Session{}, nil
}
func (m *mockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	return db.User{}, nil
}
func (m *mockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *mockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
func (m *mockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error {
	return nil
}
func (m *mockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	return db.GetSessionWithUserRow{}, pgx.ErrNoRows
}
func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}
func (m *mockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}
func (m *mockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) {
	return 0, nil
}
func (m *mockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) {
	return db.AuditLog{}, nil
}
func (m *mockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return nil, nil
}
func (m *mockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *mockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	return nil
}
func (m *mockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	return nil
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestCreateProject(t *testing.T) {
	store := newMockStore()
	svc := NewService(nil, store, nil, nil)
	ctx := context.Background()

	pmID := uuid.New().String()

	t.Run("fails if actor is not project manager", func(t *testing.T) {
		actor := ActorInfo{ID: pmID, Role: db.UserRoleTeamLeader}
		_, err := svc.CreateProject(ctx, actor, CreateProjectParams{Name: "Project Alpha"})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("fails if project name is empty", func(t *testing.T) {
		actor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}
		_, err := svc.CreateProject(ctx, actor, CreateProjectParams{Name: "   "})
		if err == nil {
			t.Error("expected error for empty name, got nil")
		}
	})

	t.Run("creates project and default columns successfully", func(t *testing.T) {
		actor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}
		dto, err := svc.CreateProject(ctx, actor, CreateProjectParams{
			Name:        "Banking App",
			Description: "Core modern banking project",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Code != "PRJ-0001" {
			t.Errorf("expected code PRJ-0001, got %s", dto.Code)
		}
		if dto.Name != "Banking App" {
			t.Errorf("expected name 'Banking App', got %s", dto.Name)
		}
		if dto.Status != "active" {
			t.Errorf("expected status 'active', got %s", dto.Status)
		}

		// Verify 5 default columns exist
		cols, err := svc.ListColumns(ctx, actor, dto.ID)
		if err != nil {
			t.Fatalf("failed to list columns: %v", err)
		}
		if len(cols) != 5 {
			t.Fatalf("expected 5 default columns, got %d", len(cols))
		}

		var hasDone bool
		for _, col := range cols {
			if col.IsDoneColumn {
				hasDone = true
			}
		}
		if !hasDone {
			t.Error("expected default columns to contain a Done column")
		}
	})
}

func TestProjectRBAC_Visibility(t *testing.T) {
	store := newMockStore()
	svc := NewService(nil, store, nil, nil)
	ctx := context.Background()

	pm1ID := uuid.New().String()
	pm2ID := uuid.New().String()
	tlMemberID := uuid.New().String()
	tlNonMemberID := uuid.New().String()
	adminID := uuid.New().String()

	// PM 1 creates project
	p1, err := svc.CreateProject(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager}, CreateProjectParams{
		Name: "PM1 Project",
	})
	if err != nil {
		t.Fatalf("create project 1 failed: %v", err)
	}

	// Add tlMember as member of p1
	tlUser := db.User{
		ID:       mustParseUUID(tlMemberID),
		FullName: "Team Leader 1",
		Email:    "tl1@bank.internal",
		Role:     db.UserRoleTeamLeader,
	}
	store.users[tlMemberID] = tlUser
	if err := svc.AddMember(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager}, p1.ID, tlMemberID); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	// PM 2 creates project
	p2, err := svc.CreateProject(ctx, ActorInfo{ID: pm2ID, Role: db.UserRoleProjectManager}, CreateProjectParams{
		Name: "PM2 Project",
	})
	if err != nil {
		t.Fatalf("create project 2 failed: %v", err)
	}

	t.Run("PM1 cannot see PM2 project -> 404 ErrProjectNotFound", func(t *testing.T) {
		_, err := svc.GetProject(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager}, p2.ID)
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})

	t.Run("TL member can see P1 project", func(t *testing.T) {
		res, err := svc.GetProject(ctx, ActorInfo{ID: tlMemberID, Role: db.UserRoleTeamLeader}, p1.ID)
		if err != nil {
			t.Fatalf("expected TL member to see project, got %v", err)
		}
		if res.ID != p1.ID {
			t.Errorf("expected ID %s, got %s", p1.ID, res.ID)
		}
	})

	t.Run("TL non-member cannot see P1 project -> 404 ErrProjectNotFound", func(t *testing.T) {
		_, err := svc.GetProject(ctx, ActorInfo{ID: tlNonMemberID, Role: db.UserRoleTeamLeader}, p1.ID)
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})

	t.Run("Admin can see any project", func(t *testing.T) {
		res, err := svc.GetProject(ctx, ActorInfo{ID: adminID, Role: db.UserRoleAdmin}, p1.ID)
		if err != nil {
			t.Fatalf("expected Admin to see project, got %v", err)
		}
		if res.ID != p1.ID {
			t.Errorf("expected ID %s, got %s", p1.ID, res.ID)
		}
	})

	t.Run("PM list only returns owned projects", func(t *testing.T) {
		list, err := svc.ListProjects(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager})
		if err != nil {
			t.Fatalf("list projects failed: %v", err)
		}
		if len(list) != 1 || list[0].ID != p1.ID {
			t.Errorf("expected exactly P1 in list, got %v", list)
		}
	})
}

func TestArchivedProjectReadOnly(t *testing.T) {
	store := newMockStore()
	svc := NewService(nil, store, nil, nil)
	ctx := context.Background()

	pmID := uuid.New().String()
	p, err := svc.CreateProject(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, CreateProjectParams{
		Name: "To Archive",
	})
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	// Archive the project
	_, err = svc.ArchiveProject(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, p.ID)
	if err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	t.Run("update archived project returns ErrProjectArchived", func(t *testing.T) {
		_, err := svc.UpdateProject(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, p.ID, UpdateProjectParams{
			Name:   "New Name",
			Status: db.ProjectStatusActive,
		})
		if !errors.Is(err, ErrProjectArchived) {
			t.Errorf("expected ErrProjectArchived, got %v", err)
		}
	})

	t.Run("add member to archived project returns ErrProjectArchived", func(t *testing.T) {
		tlID := uuid.New().String()
		store.users[tlID] = db.User{
			ID:   mustParseUUID(tlID),
			Role: db.UserRoleTeamLeader,
		}
		err := svc.AddMember(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, p.ID, tlID)
		if !errors.Is(err, ErrProjectArchived) {
			t.Errorf("expected ErrProjectArchived, got %v", err)
		}
	})
}

func TestTransferOwnerRules(t *testing.T) {
	store := newMockStore()
	svc := NewService(nil, store, nil, nil)
	ctx := context.Background()

	pm1ID := uuid.New().String()
	pm2ID := uuid.New().String()
	tlID := uuid.New().String()
	adminID := uuid.New().String()

	p, err := svc.CreateProject(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager}, CreateProjectParams{
		Name: "Transfer Test",
	})
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	t.Run("PM cannot transfer owner -> ErrForbidden", func(t *testing.T) {
		pm2User := db.User{ID: mustParseUUID(pm2ID), Role: db.UserRoleProjectManager}
		store.users[pm2ID] = pm2User
		_, err := svc.TransferOwner(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager}, p.ID, pm2ID)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("Admin cannot transfer to non-PM -> ErrTransferTargetNotPM", func(t *testing.T) {
		tlUser := db.User{ID: mustParseUUID(tlID), Role: db.UserRoleTeamLeader}
		store.users[tlID] = tlUser
		_, err := svc.TransferOwner(ctx, ActorInfo{ID: adminID, Role: db.UserRoleAdmin}, p.ID, tlID)
		if !errors.Is(err, ErrTransferTargetNotPM) {
			t.Errorf("expected ErrTransferTargetNotPM, got %v", err)
		}
	})

	t.Run("Admin can transfer to another PM", func(t *testing.T) {
		pm2User := db.User{ID: mustParseUUID(pm2ID), Role: db.UserRoleProjectManager}
		store.users[pm2ID] = pm2User
		updated, err := svc.TransferOwner(ctx, ActorInfo{ID: adminID, Role: db.UserRoleAdmin}, p.ID, pm2ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.OwnerID != pm2ID {
			t.Errorf("expected new owner %s, got %s", pm2ID, updated.OwnerID)
		}
	})
}

func TestColumnProtection(t *testing.T) {
	store := newMockStore()
	svc := NewService(nil, store, nil, nil)
	ctx := context.Background()

	pmID := uuid.New().String()
	p, err := svc.CreateProject(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, CreateProjectParams{
		Name: "Column Test",
	})
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	cols, err := svc.ListColumns(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, p.ID)
	if err != nil {
		t.Fatalf("list columns failed: %v", err)
	}

	var doneCol ColumnDTO
	var todoCol ColumnDTO
	for _, c := range cols {
		if c.IsDoneColumn {
			doneCol = c
		} else if c.Name == "To Do" {
			todoCol = c
		}
	}

	t.Run("Done column cannot be deleted -> ErrColumnDoneProtected", func(t *testing.T) {
		err := svc.DeleteColumn(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, doneCol.ID)
		if !errors.Is(err, ErrColumnDoneProtected) {
			t.Errorf("expected ErrColumnDoneProtected, got %v", err)
		}
	})

	t.Run("Non-Done column can be deleted", func(t *testing.T) {
		err := svc.DeleteColumn(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, todoCol.ID)
		if err != nil {
			t.Fatalf("expected non-done column to be deleted, got %v", err)
		}
	})

	t.Run("Cannot add second Done column -> ErrOneDoneColumnOnly", func(t *testing.T) {
		_, err := svc.CreateColumn(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, p.ID, CreateColumnParams{
			Name:         "Done 2",
			Position:     6,
			IsDoneColumn: true,
		})
		if !errors.Is(err, ErrOneDoneColumnOnly) {
			t.Errorf("expected ErrOneDoneColumnOnly, got %v", err)
		}
	})
}
