package finding

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

type mockFindingStore struct {
	mu         sync.Mutex
	users      map[string]db.User
	projects   map[string]db.Project
	members    map[string][]string // projectID -> []userID
	tasks      map[string]db.Task
	findings   map[string]db.AuditFinding
	findingSeq int
}

func newMockFindingStore() *mockFindingStore {
	return &mockFindingStore{
		users:      make(map[string]db.User),
		projects:   make(map[string]db.Project),
		members:    make(map[string][]string),
		tasks:      make(map[string]db.Task),
		findings:   make(map[string]db.AuditFinding),
		findingSeq: 1,
	}
}

// ─── db.Querier implementation for mockFindingStore ──────────────────────────

func (m *mockFindingStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error { return nil }
func (m *mockFindingStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error)     { return db.Project{}, nil }
func (m *mockFindingStore) AssessRisk(ctx context.Context, arg db.AssessRiskParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *mockFindingStore) CloseRisk(ctx context.Context, id pgtype.UUID) (db.Risk, error) { return db.Risk{}, nil }
func (m *mockFindingStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *mockFindingStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) { return 0, nil }
func (m *mockFindingStore) CountRisksByTaskIDs(ctx context.Context, taskIds []pgtype.UUID) ([]db.CountRisksByTaskIDsRow, error) { return nil, nil }
func (m *mockFindingStore) CountRisksFiltered(ctx context.Context, arg db.CountRisksFilteredParams) (int64, error) { return 0, nil }
func (m *mockFindingStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) { return 0, nil }
func (m *mockFindingStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) { return 0, nil }
func (m *mockFindingStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) { return 0, nil }
func (m *mockFindingStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockFindingStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *mockFindingStore) CreateRisk(ctx context.Context, arg db.CreateRiskParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *mockFindingStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) { return db.Session{}, nil }
func (m *mockFindingStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *mockFindingStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) { return db.TaskComment{}, nil }
func (m *mockFindingStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) { return db.User{}, nil }
func (m *mockFindingStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockFindingStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *mockFindingStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error { return nil }
func (m *mockFindingStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error { return nil }
func (m *mockFindingStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockFindingStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockFindingStore) GetNextRiskCode(ctx context.Context) (string, error) { return "RSK-0001", nil }
func (m *mockFindingStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) { return db.Project{}, nil }
func (m *mockFindingStore) GetRiskByID(ctx context.Context, id pgtype.UUID) (db.Risk, error) { return db.Risk{}, nil }
func (m *mockFindingStore) GetRiskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetRiskByIDWithDetailsRow, error) { return db.GetRiskByIDWithDetailsRow{}, nil }
func (m *mockFindingStore) GetRisksHeatmapCounts(ctx context.Context, arg db.GetRisksHeatmapCountsParams) ([]db.GetRisksHeatmapCountsRow, error) { return nil, nil }
func (m *mockFindingStore) GetRisksSummaryStats(ctx context.Context, arg db.GetRisksSummaryStatsParams) (db.GetRisksSummaryStatsRow, error) { return db.GetRisksSummaryStatsRow{}, nil }
func (m *mockFindingStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) { return db.GetSessionWithUserRow{}, nil }
func (m *mockFindingStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) { return db.User{}, nil }
func (m *mockFindingStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) { return db.User{}, nil }
func (m *mockFindingStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) { return 0, nil }
func (m *mockFindingStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) { return db.AuditLog{}, nil }
func (m *mockFindingStore) ListAllProjects(ctx context.Context) ([]db.Project, error) { return nil, nil }
func (m *mockFindingStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) { return nil, nil }
func (m *mockFindingStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) { return nil, nil }
func (m *mockFindingStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) { return nil, nil }
func (m *mockFindingStore) ListRisksFiltered(ctx context.Context, arg db.ListRisksFilteredParams) ([]db.ListRisksFilteredRow, error) { return nil, nil }
func (m *mockFindingStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) { return nil, nil }
func (m *mockFindingStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) { return nil, nil }
func (m *mockFindingStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) { return nil, nil }
func (m *mockFindingStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) { return nil, nil }
func (m *mockFindingStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *mockFindingStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) { return 0, nil }
func (m *mockFindingStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *mockFindingStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-001", nil }
func (m *mockFindingStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error { return nil }
func (m *mockFindingStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockFindingStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error { return nil }
func (m *mockFindingStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error { return nil }
func (m *mockFindingStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error { return nil }
func (m *mockFindingStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error { return nil }
func (m *mockFindingStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) { return db.Task{}, nil }
func (m *mockFindingStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *mockFindingStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) { return db.Project{}, nil }
func (m *mockFindingStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *mockFindingStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error { return nil }
func (m *mockFindingStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *mockFindingStore) UpdateRiskByRiskManagement(ctx context.Context, arg db.UpdateRiskByRiskManagementParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *mockFindingStore) UpdateRiskMitigationPlan(ctx context.Context, arg db.UpdateRiskMitigationPlanParams) (db.Risk, error) { return db.Risk{}, nil }
func (m *mockFindingStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *mockFindingStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) { return db.Task{}, nil }

// Project & Membership queries
func (m *mockFindingStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[uuidString(id)]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *mockFindingStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members, ok := m.members[uuidString(arg.ProjectID)]
	if !ok {
		return false, nil
	}
	uID := uuidString(arg.UserID)
	for _, id := range members {
		if id == uID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFindingStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
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

func (m *mockFindingStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
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

func (m *mockFindingStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[uuidString(id)]
	if !ok {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (m *mockFindingStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) {
	return db.GetTaskByIDWithDetailsRow{}, nil
}

// ─── Audit Findings Store Implementations ────────────────────────────────────

func (m *mockFindingStore) GetNextFindingCode(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	code := "FND-0001"
	m.findingSeq++
	return code, nil
}

func (m *mockFindingStore) CreateAuditFinding(ctx context.Context, arg db.CreateAuditFindingParams) (db.AuditFinding, error) {
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

func (m *mockFindingStore) GetAuditFindingByID(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(id)]
	if !ok {
		return db.AuditFinding{}, pgx.ErrNoRows
	}
	return f, nil
}

func (m *mockFindingStore) GetAuditFindingByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetAuditFindingByIDWithDetailsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[uuidString(id)]
	if !ok {
		return db.GetAuditFindingByIDWithDetailsRow{}, pgx.ErrNoRows
	}
	p := m.projects[uuidString(f.ProjectID)]
	uRaised := m.users[uuidString(f.RaisedBy)]
	var uRespName, uClosedName pgtype.Text
	if f.RespondedBy.Valid {
		if u, ok := m.users[uuidString(f.RespondedBy)]; ok {
			uRespName = pgtype.Text{String: u.FullName, Valid: true}
		}
	}
	if f.ClosedBy.Valid {
		if u, ok := m.users[uuidString(f.ClosedBy)]; ok {
			uClosedName = pgtype.Text{String: u.FullName, Valid: true}
		}
	}
	var taskTitle pgtype.Text
	if f.TaskID.Valid {
		if t, ok := m.tasks[uuidString(f.TaskID)]; ok {
			taskTitle = pgtype.Text{String: t.Title, Valid: true}
		}
	}

	return db.GetAuditFindingByIDWithDetailsRow{
		ID:              f.ID,
		Code:            f.Code,
		ProjectID:       f.ProjectID,
		TaskID:          f.TaskID,
		Title:           f.Title,
		Description:     f.Description,
		Severity:        f.Severity,
		Status:          f.Status,
		RaisedBy:        f.RaisedBy,
		Response:        f.Response,
		RespondedBy:     f.RespondedBy,
		RespondedAt:     f.RespondedAt,
		ClosedBy:        f.ClosedBy,
		ClosedAt:        f.ClosedAt,
		CreatedAt:       f.CreatedAt,
		UpdatedAt:       f.UpdatedAt,
		ProjectName:     p.Name,
		ProjectCode:     p.Code,
		ProjectOwnerID:  p.OwnerID,
		TaskTitle:       taskTitle,
		RaisedByName:    uRaised.FullName,
		RespondedByName: uRespName,
		ClosedByName:    uClosedName,
	}, nil
}

func (m *mockFindingStore) RespondAuditFinding(ctx context.Context, arg db.RespondAuditFindingParams) (db.AuditFinding, error) {
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

func (m *mockFindingStore) ReopenAuditFinding(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) {
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

func (m *mockFindingStore) CloseAuditFinding(ctx context.Context, arg db.CloseAuditFindingParams) (db.AuditFinding, error) {
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

func (m *mockFindingStore) ListAuditFindingsFiltered(ctx context.Context, arg db.ListAuditFindingsFilteredParams) ([]db.ListAuditFindingsFilteredRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.ListAuditFindingsFilteredRow
	for _, f := range m.findings {
		if len(arg.ProjectIds) > 0 {
			match := false
			for _, pid := range arg.ProjectIds {
				if pid == f.ProjectID {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if arg.ProjectID.Valid && f.ProjectID != arg.ProjectID {
			continue
		}
		if arg.Status.Valid && f.Status != arg.Status.FindingStatus {
			continue
		}
		if arg.Severity.Valid && f.Severity != arg.Severity.FindingSeverity {
			continue
		}
		p := m.projects[uuidString(f.ProjectID)]
		uRaised := m.users[uuidString(f.RaisedBy)]
		res = append(res, db.ListAuditFindingsFilteredRow{
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
			RespondedBy:    f.RespondedBy,
			RespondedAt:    f.RespondedAt,
			ClosedBy:       f.ClosedBy,
			ClosedAt:       f.ClosedAt,
			CreatedAt:      f.CreatedAt,
			UpdatedAt:      f.UpdatedAt,
			ProjectName:    p.Name,
			ProjectCode:    p.Code,
			ProjectOwnerID: p.OwnerID,
			RaisedByName:   uRaised.FullName,
		})
	}
	return res, nil
}

func (m *mockFindingStore) CountAuditFindingsFiltered(ctx context.Context, arg db.CountAuditFindingsFilteredParams) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, f := range m.findings {
		if len(arg.ProjectIds) > 0 {
			match := false
			for _, pid := range arg.ProjectIds {
				if pid == f.ProjectID {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if arg.ProjectID.Valid && f.ProjectID != arg.ProjectID {
			continue
		}
		if arg.Status.Valid && f.Status != arg.Status.FindingStatus {
			continue
		}
		if arg.Severity.Valid && f.Severity != arg.Severity.FindingSeverity {
			continue
		}
		count++
	}
	return count, nil
}

func (m *mockFindingStore) GetAuditFindingsSummaryStats(ctx context.Context, arg db.GetAuditFindingsSummaryStatsParams) (db.GetAuditFindingsSummaryStatsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var row db.GetAuditFindingsSummaryStatsRow
	for _, f := range m.findings {
		if len(arg.ProjectIds) > 0 {
			match := false
			for _, pid := range arg.ProjectIds {
				if pid == f.ProjectID {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if arg.ProjectID.Valid && f.ProjectID != arg.ProjectID {
			continue
		}
		row.TotalFindings++
		if f.Status != db.FindingStatusClosed {
			row.OpenFindings++
		}
		switch f.Severity {
		case db.FindingSeverityCritical:
			if f.Status != db.FindingStatusClosed {
				row.CriticalCount++
			}
		case db.FindingSeverityHigh:
			if f.Status != db.FindingStatusClosed {
				row.HighCount++
			}
		case db.FindingSeverityMedium:
			if f.Status != db.FindingStatusClosed {
				row.MediumCount++
			}
		case db.FindingSeverityLow:
			if f.Status != db.FindingStatusClosed {
				row.LowCount++
			}
		}
		if f.Status == db.FindingStatusResponded {
			row.RespondedCount++
		}
	}
	return row, nil
}

// ─── Test Setup Helper ───────────────────────────────────────────────────────

func setupFindingTest(t *testing.T) (*Service, *mockFindingStore, string, string, string, string) {
	t.Helper()
	store := newMockFindingStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(nil, store, nil, logger)

	pmID := uuid.New().String()
	tlID := uuid.New().String()
	auditID := uuid.New().String()
	projID := uuid.New().String()

	pmUUID, _ := parseUUID(pmID)
	tlUUID, _ := parseUUID(tlID)
	auditUUID, _ := parseUUID(auditID)
	projUUID, _ := parseUUID(projID)

	store.users[pmID] = db.User{ID: pmUUID, FullName: "PM User", Role: db.UserRoleProjectManager}
	store.users[tlID] = db.User{ID: tlUUID, FullName: "TL User", Role: db.UserRoleTeamLeader}
	store.users[auditID] = db.User{ID: auditUUID, FullName: "Auditor User", Role: db.UserRoleAudit}

	store.projects[projID] = db.Project{
		ID:      projUUID,
		Code:    "PRJ-001",
		Name:    "Audit Test Project",
		OwnerID: pmUUID,
		Status:  db.ProjectStatusActive,
	}

	store.members[projID] = []string{tlID}

	return svc, store, pmID, tlID, auditID, projID
}

// ─── Unit Tests ─────────────────────────────────────────────────────────────

func TestCreateFinding(t *testing.T) {
	svc, _, pmID, tlID, auditID, projID := setupFindingTest(t)
	ctx := context.Background()

	t.Run("fails if caller is not audit role", func(t *testing.T) {
		_, err := svc.CreateFinding(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, CreateFindingParams{
			ProjectID: projID,
			Title:     "Defect in security",
			Severity:  "high",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}

		_, err = svc.CreateFinding(ctx, ActorInfo{ID: tlID, Role: db.UserRoleTeamLeader}, CreateFindingParams{
			ProjectID: projID,
			Title:     "Defect in security",
			Severity:  "high",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("fails on empty title or invalid severity", func(t *testing.T) {
		_, err := svc.CreateFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, CreateFindingParams{
			ProjectID: projID,
			Title:     "",
			Severity:  "high",
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}

		_, err = svc.CreateFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, CreateFindingParams{
			ProjectID: projID,
			Title:     "Some finding",
			Severity:  "super_critical",
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("creates finding successfully by auditor", func(t *testing.T) {
		finding, err := svc.CreateFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, CreateFindingParams{
			ProjectID:   projID,
			Title:       "Access log gap",
			Description: "Access log is missing for sensitive reports",
			Severity:    "critical",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if finding.Code != "FND-0001" {
			t.Errorf("expected code FND-0001, got %s", finding.Code)
		}
		if finding.Status != "open" {
			t.Errorf("expected status open, got %s", finding.Status)
		}
		if finding.Severity != "critical" {
			t.Errorf("expected severity critical, got %s", finding.Severity)
		}
	})
}

func TestFindingStatusTransitions(t *testing.T) {
	svc, _, pmID, _, auditID, projID := setupFindingTest(t)
	ctx := context.Background()

	// 1. Create finding (open)
	finding, err := svc.CreateFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, CreateFindingParams{
		ProjectID:   projID,
		Title:       "Documentation mismatch",
		Description: "API spec does not match actual payload",
		Severity:    "medium",
	})
	if err != nil {
		t.Fatalf("failed to create finding: %v", err)
	}

	t.Run("cannot close open finding directly", func(t *testing.T) {
		_, err := svc.CloseFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, finding.ID)
		if !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("expected ErrInvalidStatusTransition when closing open finding, got %v", err)
		}
	})

	t.Run("PM responds to open finding", func(t *testing.T) {
		responded, err := svc.RespondFinding(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, finding.ID, RespondFindingParams{
			Response: "API documentation updated in docs/API.md",
		})
		if err != nil {
			t.Fatalf("unexpected error on respond: %v", err)
		}
		if responded.Status != "responded" {
			t.Errorf("expected status responded, got %s", responded.Status)
		}
		if responded.Response != "API documentation updated in docs/API.md" {
			t.Errorf("expected response text saved, got %s", responded.Response)
		}
	})

	t.Run("Auditor closes responded finding", func(t *testing.T) {
		closed, err := svc.CloseFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, finding.ID)
		if err != nil {
			t.Fatalf("unexpected error on close: %v", err)
		}
		if closed.Status != "closed" {
			t.Errorf("expected status closed, got %s", closed.Status)
		}
	})

	t.Run("Auditor can reopen closed finding", func(t *testing.T) {
		reopened, err := svc.ReopenFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, finding.ID)
		if err != nil {
			t.Fatalf("unexpected error on reopen: %v", err)
		}
		if reopened.Status != "open" {
			t.Errorf("expected status open after reopen, got %s", reopened.Status)
		}
	})
}

func TestFindingVisibilityPerRole(t *testing.T) {
	svc, store, pm1ID, tlID, auditID, proj1ID := setupFindingTest(t)
	ctx := context.Background()

	// Create PM2 with separate project
	pm2ID := uuid.New().String()
	proj2ID := uuid.New().String()
	pm2UUID, _ := parseUUID(pm2ID)
	proj2UUID, _ := parseUUID(proj2ID)

	store.users[pm2ID] = db.User{ID: pm2UUID, FullName: "PM 2", Role: db.UserRoleProjectManager}
	store.projects[proj2ID] = db.Project{
		ID:      proj2UUID,
		Code:    "PRJ-002",
		Name:    "PM 2 Project",
		OwnerID: pm2UUID,
		Status:  db.ProjectStatusActive,
	}

	// Create finding on Project 1
	f1, err := svc.CreateFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, CreateFindingParams{
		ProjectID: proj1ID,
		Title:     "Finding on Project 1",
		Severity:  "low",
	})
	if err != nil {
		t.Fatalf("failed to create finding 1: %v", err)
	}

	t.Run("PM1 can view finding on Project 1", func(t *testing.T) {
		got, err := svc.GetFinding(ctx, ActorInfo{ID: pm1ID, Role: db.UserRoleProjectManager}, f1.ID)
		if err != nil {
			t.Fatalf("PM1 should see finding on own project: %v", err)
		}
		if got.ID != f1.ID {
			t.Errorf("expected finding id %s, got %s", f1.ID, got.ID)
		}
	})

	t.Run("PM2 cannot view finding on Project 1 -> 404 ErrFindingNotFound", func(t *testing.T) {
		_, err := svc.GetFinding(ctx, ActorInfo{ID: pm2ID, Role: db.UserRoleProjectManager}, f1.ID)
		if !errors.Is(err, ErrFindingNotFound) {
			t.Fatalf("expected ErrFindingNotFound for PM2, got %v", err)
		}
	})

	t.Run("TL member can view finding on Project 1", func(t *testing.T) {
		got, err := svc.GetFinding(ctx, ActorInfo{ID: tlID, Role: db.UserRoleTeamLeader}, f1.ID)
		if err != nil {
			t.Fatalf("TL member should see finding: %v", err)
		}
		if got.ID != f1.ID {
			t.Errorf("expected finding id %s, got %s", f1.ID, got.ID)
		}
	})

	t.Run("Auditor can view finding", func(t *testing.T) {
		got, err := svc.GetFinding(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, f1.ID)
		if err != nil {
			t.Fatalf("Auditor should see finding: %v", err)
		}
		if got.ID != f1.ID {
			t.Errorf("expected finding id %s, got %s", f1.ID, got.ID)
		}
	})
}
