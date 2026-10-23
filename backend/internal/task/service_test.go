package task

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

	"ppo-kanban/internal/db"
)

type mockStore struct {
	mu           sync.Mutex
	projects     map[string]db.Project
	members      map[string][]string // projectID -> []userID
	columns      map[string]db.BoardColumn
	tasks        map[string]db.Task
	taskComments map[string]db.TaskComment
	users        map[string]db.User
}

func newMockStore() *mockStore {
	return &mockStore{
		projects:     make(map[string]db.Project),
		members:      make(map[string][]string),
		columns:      make(map[string]db.BoardColumn),
		tasks:        make(map[string]db.Task),
		taskComments: make(map[string]db.TaskComment),
		users:        make(map[string]db.User),
	}
}

// ─── db.Querier implementation for mockStore ─────────────────────────────────

func (m *mockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(arg.ProjectID.Bytes).String()
	uID := uuid.UUID(arg.UserID.Bytes).String()
	m.members[pID] = append(m.members[pID], uID)
	return nil
}

func (m *mockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	p.Status = db.ProjectStatusArchived
	m.projects[uuid.UUID(id.Bytes).String()] = p
	return p, nil
}

func (m *mockStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) {
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

func (m *mockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.members[uuid.UUID(projectID.Bytes).String()])), nil
}

func (m *mockStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tID := uuid.UUID(taskID.Bytes).String()
	var count int32
	for _, c := range m.taskComments {
		if uuid.UUID(c.TaskID.Bytes).String() == tID {
			count++
		}
	}
	return count, nil
}

func (m *mockStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.tasks)), nil
}

func (m *mockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) {
	return 0, nil
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

func (m *mockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
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

func (m *mockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
	return db.Session{}, nil
}

func (m *mockStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) {
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

func (m *mockStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) {
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

func (m *mockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	return db.User{}, nil
}

func (m *mockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.columns, uuid.UUID(id.Bytes).String())
	return nil
}

func (m *mockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *mockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
func (m *mockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error { return nil }

func (m *mockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	col, ok := m.columns[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.BoardColumn{}, pgx.ErrNoRows
	}
	return col, nil
}

func (m *mockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) {
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

func (m *mockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuid.UUID(id.Bytes).String()]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *mockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuid.UUID(id.Bytes).String()]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *mockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	return db.GetSessionWithUserRow{}, pgx.ErrNoRows
}

func (m *mockStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[uuid.UUID(id.Bytes).String()]
	if !ok || t.DeletedAt.Valid {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (m *mockStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) {
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

func (m *mockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
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

func (m *mockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) {
	return nil, nil
}
func (m *mockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return nil, nil
}

func (m *mockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) {
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

func (m *mockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) {
	return nil, nil
}

func (m *mockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
	return nil, nil
}

func (m *mockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
	return nil, nil
}

func (m *mockStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) {
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

func (m *mockStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) {
	return nil, nil
}

func (m *mockStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) {
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

func (m *mockStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) {
	return nil, nil
}

func (m *mockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *mockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) {
	return 0, nil
}

func (m *mockStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) {
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

func (m *mockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-0001", nil }
func (m *mockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error {
	return nil
}
func (m *mockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error {
	return nil
}

func (m *mockStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	colID := uuid.UUID(arg.ColumnID.Bytes).String()
	for k, t := range m.tasks {
		if uuid.UUID(t.ColumnID.Bytes).String() == colID && !t.DeletedAt.Valid && t.Position > arg.Position && t.Position <= arg.Position_2 {
			t.Position--
			m.tasks[k] = t
		}
	}
	return nil
}

func (m *mockStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	colID := uuid.UUID(arg.ColumnID.Bytes).String()
	for k, t := range m.tasks {
		if uuid.UUID(t.ColumnID.Bytes).String() == colID && !t.DeletedAt.Valid && t.Position >= arg.Position && t.Position < arg.Position_2 {
			t.Position++
			m.tasks[k] = t
		}
	}
	return nil
}

func (m *mockStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	colID := uuid.UUID(arg.ColumnID.Bytes).String()
	for k, t := range m.tasks {
		if uuid.UUID(t.ColumnID.Bytes).String() == colID && !t.DeletedAt.Valid && t.Position > arg.Position {
			t.Position--
			m.tasks[k] = t
		}
	}
	return nil
}

func (m *mockStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	colID := uuid.UUID(arg.ColumnID.Bytes).String()
	for k, t := range m.tasks {
		if uuid.UUID(t.ColumnID.Bytes).String() == colID && !t.DeletedAt.Valid && t.Position >= arg.Position {
			t.Position++
			m.tasks[k] = t
		}
	}
	return nil
}

func (m *mockStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) {
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

func (m *mockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) {
	return db.Project{}, nil
}
func (m *mockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) {
	return db.BoardColumn{}, nil
}
func (m *mockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error {
	return nil
}
func (m *mockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) {
	return db.Project{}, nil
}

func (m *mockStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) {
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

func (m *mockStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) {
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

// ─── Test Setup Helper ───────────────────────────────────────────────────────

func setupTaskTest(t *testing.T) (*Service, *mockStore, string, string, string, string, string) {
	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(nil, store, nil, logger)

	pmID := uuid.New().String()
	tlID := uuid.New().String()
	otherTLID := uuid.New().String()

	store.users[pmID] = db.User{
		ID:       pgtype.UUID{Bytes: uuid.MustParse(pmID), Valid: true},
		FullName: "PM User",
		Role:     db.UserRoleProjectManager,
	}
	store.users[tlID] = db.User{
		ID:       pgtype.UUID{Bytes: uuid.MustParse(tlID), Valid: true},
		FullName: "TL Member",
		Role:     db.UserRoleTeamLeader,
	}
	store.users[otherTLID] = db.User{
		ID:       pgtype.UUID{Bytes: uuid.MustParse(otherTLID), Valid: true},
		FullName: "Other TL",
		Role:     db.UserRoleTeamLeader,
	}

	projID := uuid.New().String()
	store.projects[projID] = db.Project{
		ID:      pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Code:    "PRJ-0001",
		Name:    "Project Alpha",
		Status:  db.ProjectStatusActive,
		OwnerID: pgtype.UUID{Bytes: uuid.MustParse(pmID), Valid: true},
	}
	store.members[projID] = []string{tlID}

	colTodoID := uuid.New().String()
	store.columns[colTodoID] = db.BoardColumn{
		ID:           pgtype.UUID{Bytes: uuid.MustParse(colTodoID), Valid: true},
		ProjectID:    pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Name:         "To Do",
		Position:     1,
		IsDoneColumn: false,
	}

	colDoneID := uuid.New().String()
	store.columns[colDoneID] = db.BoardColumn{
		ID:           pgtype.UUID{Bytes: uuid.MustParse(colDoneID), Valid: true},
		ProjectID:    pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Name:         "Done",
		Position:     2,
		IsDoneColumn: true,
	}

	return svc, store, projID, colTodoID, colDoneID, pmID, tlID
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestCreateTask(t *testing.T) {
	ctx := context.Background()
	svc, _, projID, colTodoID, _, pmID, tlID := setupTaskTest(t)

	// 1. PM owner creates task -> Success
	pmActor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}
	task1, err := svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Setup Architecture",
		Priority: db.TaskPriorityHigh,
	})
	if err != nil {
		t.Fatalf("expected create task success, got: %v", err)
	}
	if task1.Title != "Setup Architecture" || task1.Position != 1 || task1.Version != 1 {
		t.Errorf("unexpected task fields: %+v", task1)
	}

	// 2. TL member creates task -> Success
	tlActor := ActorInfo{ID: tlID, Role: db.UserRoleTeamLeader}
	task2, err := svc.CreateTask(ctx, tlActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Write Unit Tests",
		Priority: db.TaskPriorityMedium,
	})
	if err != nil {
		t.Fatalf("expected TL create task success, got: %v", err)
	}
	if task2.Position != 2 {
		t.Errorf("expected position 2, got %d", task2.Position)
	}

	// 3. Unauthorized role (e.g. Audit) creates task -> Forbidden
	auditActor := ActorInfo{ID: uuid.New().String(), Role: db.UserRoleAudit}
	_, err = svc.CreateTask(ctx, auditActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Audit Task",
	})
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestMakerCheckerDoneColumn(t *testing.T) {
	ctx := context.Background()
	svc, _, projID, colTodoID, colDoneID, pmID, tlID := setupTaskTest(t)

	pmActor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}
	tlActor := ActorInfo{ID: tlID, Role: db.UserRoleTeamLeader}

	// Create task in To Do
	task, err := svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Implement KYC validation",
	})
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	// 1. TL attempts to move task into Done -> Must fail with ErrMakerCheckerDone
	_, err = svc.MoveTask(ctx, tlActor, task.ID, MoveTaskParams{
		ToColumnID: colDoneID,
		Position:   1,
		Version:    task.Version,
	})
	if err != ErrMakerCheckerDone {
		t.Errorf("expected ErrMakerCheckerDone for TL moving to Done, got: %v", err)
	}

	// 2. PM owner moves task to Done -> Success
	movedTask, err := svc.MoveTask(ctx, pmActor, task.ID, MoveTaskParams{
		ToColumnID: colDoneID,
		Position:   1,
		Version:    task.Version,
	})
	if err != nil {
		t.Fatalf("expected PM move to Done success, got: %v", err)
	}
	if movedTask.ColumnID != colDoneID || movedTask.Version != 2 || movedTask.CompletedAt == nil {
		t.Errorf("unexpected moved task state: %+v", movedTask)
	}

	// 3. TL attempts to move task OUT of Done -> Must fail with ErrMakerCheckerDone
	_, err = svc.MoveTask(ctx, tlActor, task.ID, MoveTaskParams{
		ToColumnID: colTodoID,
		Position:   1,
		Version:    movedTask.Version,
	})
	if err != ErrMakerCheckerDone {
		t.Errorf("expected ErrMakerCheckerDone for TL moving out of Done, got: %v", err)
	}
}

func TestOptimisticLocking(t *testing.T) {
	ctx := context.Background()
	svc, _, projID, colTodoID, _, pmID, _ := setupTaskTest(t)
	pmActor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}

	task, err := svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Initial Task",
	})
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	// First update with matching version 1 -> succeeds, version becomes 2
	updated, err := svc.UpdateTask(ctx, pmActor, task.ID, UpdateTaskParams{
		Version: 1,
		Title:   "Updated Task 1",
	})
	if err != nil {
		t.Fatalf("update 1 failed: %v", err)
	}
	if updated.Version != 2 {
		t.Errorf("expected version 2, got %d", updated.Version)
	}

	// Concurrent update with stale version 1 -> Must fail with ErrVersionConflict
	_, err = svc.UpdateTask(ctx, pmActor, task.ID, UpdateTaskParams{
		Version: 1,
		Title:   "Concurrent Stale Update",
	})
	if err != ErrVersionConflict {
		t.Errorf("expected ErrVersionConflict, got: %v", err)
	}
}

func TestWIPLimitEnforcement(t *testing.T) {
	ctx := context.Background()
	svc, store, projID, colTodoID, _, pmID, _ := setupTaskTest(t)
	pmActor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}

	// Create column with WIP limit = 1
	colWipID := uuid.New().String()
	store.columns[colWipID] = db.BoardColumn{
		ID:        pgtype.UUID{Bytes: uuid.MustParse(colWipID), Valid: true},
		ProjectID: pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Name:      "In Progress",
		Position:  2,
		WipLimit:  pgtype.Int4{Int32: 1, Valid: true},
	}

	// Fill the WIP limit: create 1 task
	t1, err := svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colWipID,
		Title:    "Active WIP Task",
	})
	if err != nil {
		t.Fatalf("create t1 failed: %v", err)
	}

	// 1. Try to create second task in WIP column -> ErrWIPLimitExceeded
	_, err = svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colWipID,
		Title:    "Exceeding Task",
	})
	if err != ErrWIPLimitExceeded {
		t.Errorf("expected ErrWIPLimitExceeded on create, got: %v", err)
	}

	// 2. Try to move task from To Do into WIP column -> ErrWIPLimitExceeded
	t2, err := svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Task in Todo",
	})
	if err != nil {
		t.Fatalf("create t2 failed: %v", err)
	}

	_, err = svc.MoveTask(ctx, pmActor, t2.ID, MoveTaskParams{
		ToColumnID: colWipID,
		Position:   2,
		Version:    t2.Version,
	})
	if err != ErrWIPLimitExceeded {
		t.Errorf("expected ErrWIPLimitExceeded on move, got: %v", err)
	}

	// 3. Reordering WITHIN the same WIP column does NOT exceed WIP limit
	_, err = svc.MoveTask(ctx, pmActor, t1.ID, MoveTaskParams{
		ToColumnID: colWipID,
		Position:   1,
		Version:    t1.Version,
	})
	if err != nil {
		t.Errorf("expected reorder within same column to succeed, got: %v", err)
	}
}

func TestTaskComments(t *testing.T) {
	ctx := context.Background()
	svc, _, projID, colTodoID, _, pmID, tlID := setupTaskTest(t)
	pmActor := ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}
	tlActor := ActorInfo{ID: tlID, Role: db.UserRoleTeamLeader}

	task, err := svc.CreateTask(ctx, pmActor, projID, CreateTaskParams{
		ColumnID: colTodoID,
		Title:    "Task with Comments",
	})
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	// Empty comment -> ErrCommentBodyEmpty
	_, err = svc.CreateComment(ctx, tlActor, task.ID, "   ")
	if err != ErrCommentBodyEmpty {
		t.Errorf("expected ErrCommentBodyEmpty, got: %v", err)
	}

	// Valid comment by TL
	c1, err := svc.CreateComment(ctx, tlActor, task.ID, "Started looking into this.")
	if err != nil {
		t.Fatalf("create comment failed: %v", err)
	}
	if c1.Body != "Started looking into this." {
		t.Errorf("unexpected comment body: %s", c1.Body)
	}

	// List comments
	comments, err := svc.ListComments(ctx, pmActor, task.ID)
	if err != nil {
		t.Fatalf("list comments failed: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(comments))
	}
}
