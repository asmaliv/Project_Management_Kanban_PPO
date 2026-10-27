package risk

import (
	"context"
	"errors"
	"fmt"
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

type riskMockStore struct {
	mu           sync.Mutex
	users        map[string]db.User
	projects     map[string]db.Project
	members      map[string][]string // projectID -> []userID
	tasks        map[string]db.Task
	risks        map[string]db.Risk
	riskSeq      int
}

func newRiskMockStore() *riskMockStore {
	return &riskMockStore{
		users:    make(map[string]db.User),
		projects: make(map[string]db.Project),
		members:  make(map[string][]string),
		tasks:    make(map[string]db.Task),
		risks:    make(map[string]db.Risk),
		riskSeq:  1,
	}
}

// ─── db.Querier implementation for riskMockStore ─────────────────────────────

func (m *riskMockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error { return nil }
func (m *riskMockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error)     { return db.Project{}, nil }
func (m *riskMockStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *riskMockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) { return 0, nil }
func (m *riskMockStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) { return 0, nil }
func (m *riskMockStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) { return 0, nil }
func (m *riskMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) { return 0, nil }
func (m *riskMockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *riskMockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *riskMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) { return db.Session{}, nil }
func (m *riskMockStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *riskMockStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) { return db.TaskComment{}, nil }
func (m *riskMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) { return db.User{}, nil }
func (m *riskMockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *riskMockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *riskMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error { return nil }
func (m *riskMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error { return nil }
func (m *riskMockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *riskMockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }

func (m *riskMockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	p, ok := m.projects[idStr]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *riskMockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	return m.GetProjectByID(ctx, id)
}

func (m *riskMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
	return db.GetSessionWithUserRow{}, pgx.ErrNoRows
}

func (m *riskMockStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	t, ok := m.tasks[idStr]
	if !ok || t.DeletedAt.Valid {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (m *riskMockStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) {
	return db.GetTaskByIDWithDetailsRow{}, pgx.ErrNoRows
}

func (m *riskMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}

func (m *riskMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	u, ok := m.users[idStr]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *riskMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) { return 0, nil }
func (m *riskMockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) { return db.AuditLog{}, nil }

func (m *riskMockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(arg.ProjectID.Bytes).String()
	uID := uuid.UUID(arg.UserID.Bytes).String()
	for _, mUID := range m.members[pID] {
		if mUID == uID {
			return true, nil
		}
	}
	return false, nil
}

func (m *riskMockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := make([]db.Project, 0, len(m.projects))
	for _, p := range m.projects {
		if !p.DeletedAt.Valid {
			list = append(list, p)
		}
	}
	return list, nil
}

func (m *riskMockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) { return nil, nil }
func (m *riskMockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) { return nil, nil }
func (m *riskMockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) { return nil, nil }

func (m *riskMockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uID := uuid.UUID(userID.Bytes).String()
	var res []db.Project
	for pID, mems := range m.members {
		for _, mUID := range mems {
			if mUID == uID {
				if p, ok := m.projects[pID]; ok && !p.DeletedAt.Valid {
					res = append(res, p)
				}
			}
		}
	}
	return res, nil
}

func (m *riskMockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.Project
	for _, p := range m.projects {
		if p.OwnerID == ownerID && !p.DeletedAt.Valid {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *riskMockStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) { return nil, nil }
func (m *riskMockStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) { return nil, nil }
func (m *riskMockStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) { return nil, nil }
func (m *riskMockStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) { return nil, nil }
func (m *riskMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *riskMockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) { return 0, nil }
func (m *riskMockStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *riskMockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-0001", nil }
func (m *riskMockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error { return nil }
func (m *riskMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *riskMockStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error { return nil }
func (m *riskMockStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error { return nil }
func (m *riskMockStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error { return nil }
func (m *riskMockStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error { return nil }
func (m *riskMockStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) { return db.Task{}, nil }
func (m *riskMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *riskMockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) { return db.Project{}, nil }
func (m *riskMockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *riskMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error { return nil }
func (m *riskMockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *riskMockStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *riskMockStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) { return db.Task{}, nil }

// Risk-specific methods
func (m *riskMockStore) GetNextRiskCode(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	code := "RSK-0001"
	m.riskSeq++
	return code, nil
}

func (m *riskMockStore) CreateRisk(ctx context.Context, arg db.CreateRiskParams) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	var score pgtype.Int4
	if arg.Likelihood.Valid && arg.Impact.Valid {
		score = pgtype.Int4{Int32: arg.Likelihood.Int32 * arg.Impact.Int32, Valid: true}
	}
	r := db.Risk{
		ID:             pgtype.UUID{Bytes: id, Valid: true},
		Code:           arg.Code,
		ProjectID:      arg.ProjectID,
		TaskID:         arg.TaskID,
		Title:          arg.Title,
		Description:    arg.Description,
		Category:       arg.Category,
		Likelihood:     arg.Likelihood,
		Impact:         arg.Impact,
		Score:          score,
		Level:          arg.Level,
		Status:         arg.Status,
		MitigationPlan: arg.MitigationPlan,
		OwnerID:        arg.OwnerID,
		RaisedBy:       arg.RaisedBy,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.risks[id.String()] = r
	return r, nil
}

func (m *riskMockStore) GetRiskByID(ctx context.Context, id pgtype.UUID) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.Risk{}, pgx.ErrNoRows
	}
	return r, nil
}

func (m *riskMockStore) GetRiskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetRiskByIDWithDetailsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.GetRiskByIDWithDetailsRow{}, pgx.ErrNoRows
	}
	pIDStr := uuid.UUID(r.ProjectID.Bytes).String()
	p, ok := m.projects[pIDStr]
	if !ok || p.DeletedAt.Valid {
		return db.GetRiskByIDWithDetailsRow{}, pgx.ErrNoRows
	}

	var taskTitle pgtype.Text
	if r.TaskID.Valid {
		tIDStr := uuid.UUID(r.TaskID.Bytes).String()
		if t, ok := m.tasks[tIDStr]; ok {
			taskTitle = pgtype.Text{String: t.Title, Valid: true}
		}
	}

	var raisedByName string
	raisedStr := uuid.UUID(r.RaisedBy.Bytes).String()
	if u, ok := m.users[raisedStr]; ok {
		raisedByName = u.FullName
	}

	var ownerName pgtype.Text
	if r.OwnerID.Valid {
		oStr := uuid.UUID(r.OwnerID.Bytes).String()
		if u, ok := m.users[oStr]; ok {
			ownerName = pgtype.Text{String: u.FullName, Valid: true}
		}
	}

	return db.GetRiskByIDWithDetailsRow{
		ID:             r.ID,
		Code:           r.Code,
		ProjectID:      r.ProjectID,
		TaskID:         r.TaskID,
		Title:          r.Title,
		Description:    r.Description,
		Category:       r.Category,
		Likelihood:     r.Likelihood,
		Impact:         r.Impact,
		Score:          r.Score,
		Level:          r.Level,
		Status:         r.Status,
		MitigationPlan: r.MitigationPlan,
		OwnerID:        r.OwnerID,
		RaisedBy:       r.RaisedBy,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
		ClosedAt:       r.ClosedAt,
		ProjectName:    p.Name,
		ProjectCode:    p.Code,
		ProjectOwnerID: p.OwnerID,
		TaskTitle:      taskTitle,
		OwnerName:      ownerName,
		RaisedByName:   raisedByName,
	}, nil
}

func (m *riskMockStore) AssessRisk(ctx context.Context, arg db.AssessRiskParams) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(arg.ID.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.Risk{}, pgx.ErrNoRows
	}
	r.Likelihood = arg.Likelihood
	r.Impact = arg.Impact
	r.Score = pgtype.Int4{Int32: arg.Likelihood.Int32 * arg.Impact.Int32, Valid: true}
	r.Level = arg.Level
	if arg.MitigationPlan.Valid {
		r.MitigationPlan = arg.MitigationPlan.String
	}
	if len(r.MitigationPlan) > 0 {
		r.Status = db.RiskStatusMitigating
	} else {
		r.Status = db.RiskStatusAssessed
	}
	r.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.risks[idStr] = r
	return r, nil
}

func (m *riskMockStore) CloseRisk(ctx context.Context, id pgtype.UUID) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.Risk{}, pgx.ErrNoRows
	}
	r.Status = db.RiskStatusClosed
	r.ClosedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	r.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.risks[idStr] = r
	return r, nil
}

func (m *riskMockStore) UpdateRiskByRiskManagement(ctx context.Context, arg db.UpdateRiskByRiskManagementParams) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(arg.ID.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.Risk{}, pgx.ErrNoRows
	}
	r.Title = arg.Title
	r.Description = arg.Description
	r.Category = arg.Category
	r.TaskID = arg.TaskID
	r.OwnerID = arg.OwnerID
	r.Status = arg.Status
	r.MitigationPlan = arg.MitigationPlan
	r.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.risks[idStr] = r
	return r, nil
}

func (m *riskMockStore) UpdateRiskMitigationPlan(ctx context.Context, arg db.UpdateRiskMitigationPlanParams) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(arg.ID.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.Risk{}, pgx.ErrNoRows
	}
	r.MitigationPlan = arg.MitigationPlan
	if r.Status == db.RiskStatusIdentified {
		r.Status = db.RiskStatusMitigating
	}
	r.UpdatedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.risks[idStr] = r
	return r, nil
}

func (m *riskMockStore) ListRisksFiltered(ctx context.Context, arg db.ListRisksFilteredParams) ([]db.ListRisksFilteredRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.ListRisksFilteredRow
	for _, r := range m.risks {
		pIDStr := uuid.UUID(r.ProjectID.Bytes).String()
		p, ok := m.projects[pIDStr]
		if !ok || p.DeletedAt.Valid {
			continue
		}
		res = append(res, db.ListRisksFilteredRow{
			ID:             r.ID,
			Code:           r.Code,
			ProjectID:      r.ProjectID,
			TaskID:         r.TaskID,
			Title:          r.Title,
			Description:    r.Description,
			Category:       r.Category,
			Likelihood:     r.Likelihood,
			Impact:         r.Impact,
			Score:          r.Score,
			Level:          r.Level,
			Status:         r.Status,
			MitigationPlan: r.MitigationPlan,
			OwnerID:        r.OwnerID,
			RaisedBy:       r.RaisedBy,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
			ClosedAt:       r.ClosedAt,
			ProjectName:    p.Name,
			ProjectCode:    p.Code,
			ProjectOwnerID: p.OwnerID,
		})
	}
	return res, nil
}

func (m *riskMockStore) CountRisksFiltered(ctx context.Context, arg db.CountRisksFilteredParams) (int64, error) {
	rows, err := m.ListRisksFiltered(ctx, db.ListRisksFilteredParams{
		ProjectIds: arg.ProjectIds,
		ProjectID:  arg.ProjectID,
		TaskID:     arg.TaskID,
		Level:      arg.Level,
		Status:     arg.Status,
		Category:   arg.Category,
		Search:     arg.Search,
	})
	return int64(len(rows)), err
}

func (m *riskMockStore) GetRisksHeatmapCounts(ctx context.Context, arg db.GetRisksHeatmapCountsParams) ([]db.GetRisksHeatmapCountsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := make(map[string]int64)
	for _, r := range m.risks {
		if r.Likelihood.Valid && r.Impact.Valid {
			if !arg.IncludeClosed.Bool && r.Status == db.RiskStatusClosed {
				continue
			}
			key := fmt.Sprintf("%d_%d", r.Likelihood.Int32, r.Impact.Int32)
			counts[key]++
		}
	}
	var res []db.GetRisksHeatmapCountsRow
	for k, cnt := range counts {
		var l, i int32
		fmt.Sscanf(k, "%d_%d", &l, &i)
		res = append(res, db.GetRisksHeatmapCountsRow{
			Likelihood: pgtype.Int4{Int32: l, Valid: true},
			Impact:     pgtype.Int4{Int32: i, Valid: true},
			Count:      cnt,
		})
	}
	return res, nil
}

func (m *riskMockStore) GetRisksSummaryStats(ctx context.Context, arg db.GetRisksSummaryStatsParams) (db.GetRisksSummaryStatsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var row db.GetRisksSummaryStatsRow
	for _, r := range m.risks {
		row.TotalRisks++
		if r.Status != db.RiskStatusClosed {
			row.OpenRisks++
			if r.Level.Valid && (r.Level.RiskLevel == db.RiskLevelHigh || r.Level.RiskLevel == db.RiskLevelCritical) {
				row.HighCriticalRisks++
			}
			if len(r.MitigationPlan) == 0 {
				row.UnmitigatedRisks++
			}
			if r.Level.Valid {
				switch r.Level.RiskLevel {
				case db.RiskLevelLow:
					row.LowLevelCount++
				case db.RiskLevelMedium:
					row.MediumLevelCount++
				case db.RiskLevelHigh:
					row.HighLevelCount++
				case db.RiskLevelCritical:
					row.CriticalLevelCount++
				}
			}
		}
	}
	return row, nil
}

func (m *riskMockStore) CountRisksByTaskIDs(ctx context.Context, taskIds []pgtype.UUID) ([]db.CountRisksByTaskIDsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := make(map[string]struct{ total, highCrit int64 })
	for _, r := range m.risks {
		if r.TaskID.Valid {
			tid := uuid.UUID(r.TaskID.Bytes).String()
			c := counts[tid]
			c.total++
			if r.Level.Valid && (r.Level.RiskLevel == db.RiskLevelHigh || r.Level.RiskLevel == db.RiskLevelCritical) {
				c.highCrit++
			}
			counts[tid] = c
		}
	}
	var res []db.CountRisksByTaskIDsRow
	for _, tUUID := range taskIds {
		tid := uuid.UUID(tUUID.Bytes).String()
		if c, ok := counts[tid]; ok {
			res = append(res, db.CountRisksByTaskIDsRow{
				TaskID:            tUUID,
				Count:             c.total,
				HighCriticalCount: c.highCrit,
			})
		}
	}
	return res, nil
}

// Phase 7 Querier stubs
func (m *riskMockStore) CloseAuditFinding(ctx context.Context, arg db.CloseAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *riskMockStore) CountAuditFindingsFiltered(ctx context.Context, arg db.CountAuditFindingsFilteredParams) (int64, error) { return 0, nil }
func (m *riskMockStore) CreateAuditFinding(ctx context.Context, arg db.CreateAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *riskMockStore) GetAuditFindingByID(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *riskMockStore) GetAuditFindingByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetAuditFindingByIDWithDetailsRow, error) { return db.GetAuditFindingByIDWithDetailsRow{}, nil }
func (m *riskMockStore) GetAuditFindingsSummaryStats(ctx context.Context, arg db.GetAuditFindingsSummaryStatsParams) (db.GetAuditFindingsSummaryStatsRow, error) { return db.GetAuditFindingsSummaryStatsRow{}, nil }
func (m *riskMockStore) GetNextFindingCode(ctx context.Context) (string, error) { return "FND-0001", nil }
func (m *riskMockStore) ListAuditFindingsFiltered(ctx context.Context, arg db.ListAuditFindingsFilteredParams) ([]db.ListAuditFindingsFilteredRow, error) { return nil, nil }
func (m *riskMockStore) ReopenAuditFinding(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *riskMockStore) RespondAuditFinding(ctx context.Context, arg db.RespondAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }

// ─── Tests ───────────────────────────────────────────────────────────────────

func setupRiskTest(t *testing.T) (*Service, *riskMockStore, string, string, string) {
	t.Helper()
	store := newRiskMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(nil, store, nil, logger)

	pmID := uuid.New().String()
	tlID := uuid.New().String()
	rmID := uuid.New().String()

	store.users[pmID] = db.User{
		ID:       pgtype.UUID{Bytes: uuid.MustParse(pmID), Valid: true},
		FullName: "Project Manager 1",
		Role:     db.UserRoleProjectManager,
	}
	store.users[tlID] = db.User{
		ID:       pgtype.UUID{Bytes: uuid.MustParse(tlID), Valid: true},
		FullName: "Team Leader 1",
		Role:     db.UserRoleTeamLeader,
	}
	store.users[rmID] = db.User{
		ID:       pgtype.UUID{Bytes: uuid.MustParse(rmID), Valid: true},
		FullName: "Risk Manager 1",
		Role:     db.UserRoleRiskManagement,
	}

	return svc, store, pmID, tlID, rmID
}

func TestCalculateRiskScoreAndLevel(t *testing.T) {
	tests := []struct {
		name          string
		likelihood    int32
		impact        int32
		expectedScore int32
		expectedLevel db.RiskLevel
		expectErr     bool
	}{
		{"Low min (1x1)", 1, 1, 1, db.RiskLevelLow, false},
		{"Low max (2x2)", 2, 2, 4, db.RiskLevelLow, false},
		{"Medium min (1x5)", 1, 5, 5, db.RiskLevelMedium, false},
		{"Medium max (3x3)", 3, 3, 9, db.RiskLevelMedium, false},
		{"High min (2x5)", 2, 5, 10, db.RiskLevelHigh, false},
		{"High max (3x5)", 3, 5, 15, db.RiskLevelHigh, false},
		{"Critical min (4x4)", 4, 4, 16, db.RiskLevelCritical, false},
		{"Critical max (5x5)", 5, 5, 25, db.RiskLevelCritical, false},
		{"Invalid likelihood 0", 0, 3, 0, "", true},
		{"Invalid likelihood 6", 6, 3, 0, "", true},
		{"Invalid impact 0", 3, 0, 0, "", true},
		{"Invalid impact 6", 3, 6, 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, level, err := CalculateRiskScoreAndLevel(tt.likelihood, tt.impact)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score)
			}
			if level != tt.expectedLevel {
				t.Errorf("expected level %s, got %s", tt.expectedLevel, level)
			}
		})
	}
}

func TestCreateRisk(t *testing.T) {
	svc, store, pmID, tlID, rmID := setupRiskTest(t)
	ctx := context.Background()

	// Setup Project
	projID := uuid.New().String()
	store.projects[projID] = db.Project{
		ID:      pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Code:    "PRJ-0001",
		Name:    "Project Alpha",
		OwnerID: pgtype.UUID{Bytes: uuid.MustParse(pmID), Valid: true},
		Status:  db.ProjectStatusActive,
	}
	store.members[projID] = []string{tlID}

	// 1. PM on own project -> Success
	t.Run("PM can raise risk on own project", func(t *testing.T) {
		r, err := svc.CreateRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, CreateRiskParams{
			ProjectID:   projID,
			Title:       "Server capacity issue",
			Category:    "technical",
			Description: "May exceed limit",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Code != "RSK-0001" {
			t.Errorf("expected code RSK-0001, got %s", r.Code)
		}
		if r.Status != "identified" {
			t.Errorf("expected initial status identified, got %s", r.Status)
		}
	})

	// 2. TL on assigned project -> Success
	t.Run("TL can raise risk on assigned project", func(t *testing.T) {
		r, err := svc.CreateRisk(ctx, ActorInfo{ID: tlID, Role: db.UserRoleTeamLeader}, CreateRiskParams{
			ProjectID: projID,
			Title:     "Vendor delay",
			Category:  "schedule",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Title != "Vendor delay" {
			t.Errorf("expected title 'Vendor delay', got %s", r.Title)
		}
	})

	// 3. Risk Management on any active project -> Success
	t.Run("Risk Management can raise risk on any project", func(t *testing.T) {
		r, err := svc.CreateRisk(ctx, ActorInfo{ID: rmID, Role: db.UserRoleRiskManagement}, CreateRiskParams{
			ProjectID: projID,
			Title:     "Compliance check fail",
			Category:  "compliance",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Category != "compliance" {
			t.Errorf("expected category compliance, got %s", r.Category)
		}
	})

	// 4. Admin and Audit cannot raise risk -> ErrForbidden
	t.Run("Admin and Audit cannot raise risk", func(t *testing.T) {
		adminID := uuid.New().String()
		_, err := svc.CreateRisk(ctx, ActorInfo{ID: adminID, Role: db.UserRoleAdmin}, CreateRiskParams{
			ProjectID: projID,
			Title:     "Admin risk",
			Category:  "operational",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden for admin, got %v", err)
		}

		auditID := uuid.New().String()
		_, err = svc.CreateRisk(ctx, ActorInfo{ID: auditID, Role: db.UserRoleAudit}, CreateRiskParams{
			ProjectID: projID,
			Title:     "Audit risk",
			Category:  "operational",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden for audit, got %v", err)
		}
	})

	// 5. Non-member TL cannot raise risk -> ErrForbidden
	t.Run("Non-member TL cannot raise risk", func(t *testing.T) {
		otherTL := uuid.New().String()
		_, err := svc.CreateRisk(ctx, ActorInfo{ID: otherTL, Role: db.UserRoleTeamLeader}, CreateRiskParams{
			ProjectID: projID,
			Title:     "Intruder TL risk",
			Category:  "security",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden for non-member TL, got %v", err)
		}
	})
}

func TestAssessAndCloseRisk(t *testing.T) {
	svc, store, pmID, _, rmID := setupRiskTest(t)
	ctx := context.Background()

	projID := uuid.New().String()
	store.projects[projID] = db.Project{
		ID:      pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Code:    "PRJ-0001",
		Name:    "Project Alpha",
		OwnerID: pgtype.UUID{Bytes: uuid.MustParse(pmID), Valid: true},
		Status:  db.ProjectStatusActive,
	}

	created, err := svc.CreateRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, CreateRiskParams{
		ProjectID: projID,
		Title:     "Data breach vulnerability",
		Category:  "security",
	})
	if err != nil {
		t.Fatalf("failed to create risk: %v", err)
	}

	// 1. Non-RM assess -> Forbidden
	t.Run("Non-RM cannot assess risk", func(t *testing.T) {
		_, err := svc.AssessRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, created.ID, AssessRiskParams{
			Likelihood: 4,
			Impact:     4,
		})
		if !errors.Is(err, ErrOnlyRiskManagementAssess) {
			t.Errorf("expected ErrOnlyRiskManagementAssess, got %v", err)
		}
	})

	// 2. RM assess with 4x4 = 16 (Critical) without mitigation plan
	t.Run("RM assesses risk to Critical", func(t *testing.T) {
		assessed, err := svc.AssessRisk(ctx, ActorInfo{ID: rmID, Role: db.UserRoleRiskManagement}, created.ID, AssessRiskParams{
			Likelihood: 4,
			Impact:     4,
		})
		if err != nil {
			t.Fatalf("unexpected assess error: %v", err)
		}
		if *assessed.Score != 16 {
			t.Errorf("expected score 16, got %d", *assessed.Score)
		}
		if *assessed.Level != "critical" {
			t.Errorf("expected level critical, got %s", *assessed.Level)
		}
		if assessed.Status != "assessed" {
			t.Errorf("expected status assessed, got %s", assessed.Status)
		}
	})

	// 3. RM attempts to close Critical risk WITHOUT mitigation plan -> Denied
	t.Run("Cannot close Critical risk without mitigation plan", func(t *testing.T) {
		_, err := svc.CloseRisk(ctx, ActorInfo{ID: rmID, Role: db.UserRoleRiskManagement}, created.ID)
		if !errors.Is(err, ErrHighCriticalMitigationRequired) {
			t.Errorf("expected ErrHighCriticalMitigationRequired, got %v", err)
		}
	})

	// 4. PM updates mitigation plan
	t.Run("PM updates mitigation plan", func(t *testing.T) {
		mit := "Implement multi-layer encryption and firewall"
		updated, err := svc.UpdateRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, created.ID, UpdateRiskParams{
			MitigationPlan: &mit,
		})
		if err != nil {
			t.Fatalf("unexpected update error: %v", err)
		}
		if updated.MitigationPlan != mit {
			t.Errorf("expected updated mitigation plan, got %s", updated.MitigationPlan)
		}
	})

	// 5. Now RM closes the Critical risk -> Success
	t.Run("RM closes Critical risk after mitigation plan is filled", func(t *testing.T) {
		closed, err := svc.CloseRisk(ctx, ActorInfo{ID: rmID, Role: db.UserRoleRiskManagement}, created.ID)
		if err != nil {
			t.Fatalf("unexpected close error: %v", err)
		}
		if closed.Status != "closed" {
			t.Errorf("expected status closed, got %s", closed.Status)
		}
		if closed.ClosedAt == nil {
			t.Errorf("expected closed_at to be set")
		}
	})

	// 6. PM cannot close risk -> Forbidden
	t.Run("PM cannot close risk", func(t *testing.T) {
		_, err := svc.CloseRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, created.ID)
		if !errors.Is(err, ErrOnlyRiskManagementClose) {
			t.Errorf("expected ErrOnlyRiskManagementClose, got %v", err)
		}
	})
}

func TestUpdateRiskPermissions(t *testing.T) {
	svc, store, pmID, _, rmID := setupRiskTest(t)
	ctx := context.Background()

	projID := uuid.New().String()
	store.projects[projID] = db.Project{
		ID:      pgtype.UUID{Bytes: uuid.MustParse(projID), Valid: true},
		Code:    "PRJ-0001",
		Name:    "Project Alpha",
		OwnerID: pgtype.UUID{Bytes: uuid.MustParse(pmID), Valid: true},
		Status:  db.ProjectStatusActive,
	}

	created, err := svc.CreateRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, CreateRiskParams{
		ProjectID: projID,
		Title:     "Budget overrun",
		Category:  "financial",
	})
	if err != nil {
		t.Fatalf("failed to create risk: %v", err)
	}

	// PM trying to change title or category -> Denied
	t.Run("PM cannot change risk title or category", func(t *testing.T) {
		newTitle := "New title"
		_, err := svc.UpdateRisk(ctx, ActorInfo{ID: pmID, Role: db.UserRoleProjectManager}, created.ID, UpdateRiskParams{
			Title: &newTitle,
		})
		if !errors.Is(err, ErrOnlyPMUpdateMitigation) {
			t.Errorf("expected ErrOnlyPMUpdateMitigation, got %v", err)
		}
	})

	// RM can change title and category
	t.Run("RM can update risk title and category", func(t *testing.T) {
		newTitle := "Financial liquidity shortfall"
		newCat := "financial"
		updated, err := svc.UpdateRisk(ctx, ActorInfo{ID: rmID, Role: db.UserRoleRiskManagement}, created.ID, UpdateRiskParams{
			Title:    &newTitle,
			Category: &newCat,
		})
		if err != nil {
			t.Fatalf("unexpected update error: %v", err)
		}
		if updated.Title != newTitle {
			t.Errorf("expected title %s, got %s", newTitle, updated.Title)
		}
	})
}
