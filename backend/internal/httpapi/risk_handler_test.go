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
	"ppo-kanban/internal/risk"
)

type httpRiskMockStore struct {
	mu           sync.Mutex
	users        map[string]db.User
	sessions     map[string]db.Session
	projects     map[string]db.Project
	members      map[string][]string // projectID -> []userID
	columns      map[string]db.BoardColumn
	tasks        map[string]db.Task
	taskComments map[string]db.TaskComment
	risks        map[string]db.Risk
	riskSeq      int
}

func newHTTPRiskMockStore() *httpRiskMockStore {
	return &httpRiskMockStore{
		users:        make(map[string]db.User),
		sessions:     make(map[string]db.Session),
		projects:     make(map[string]db.Project),
		members:      make(map[string][]string),
		columns:      make(map[string]db.BoardColumn),
		tasks:        make(map[string]db.Task),
		taskComments: make(map[string]db.TaskComment),
		risks:        make(map[string]db.Risk),
		riskSeq:      1,
	}
}

// ─── db.Querier implementation ───────────────────────────────────────────────

func (m *httpRiskMockStore) AddProjectMember(ctx context.Context, arg db.AddProjectMemberParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pID := uuid.UUID(arg.ProjectID.Bytes).String()
	uID := uuid.UUID(arg.UserID.Bytes).String()
	m.members[pID] = append(m.members[pID], uID)
	return nil
}

func (m *httpRiskMockStore) ArchiveProject(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.projects[uuid.UUID(id.Bytes).String()]
	p.Status = db.ProjectStatusArchived
	m.projects[uuid.UUID(id.Bytes).String()] = p
	return p, nil
}

func (m *httpRiskMockStore) CountActiveTasksInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpRiskMockStore) CountProjectMembers(ctx context.Context, projectID pgtype.UUID) (int64, error) { return 0, nil }
func (m *httpRiskMockStore) CountTaskComments(ctx context.Context, taskID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpRiskMockStore) CountTasksByProjectsFiltered(ctx context.Context, arg db.CountTasksByProjectsFilteredParams) (int64, error) { return 0, nil }
func (m *httpRiskMockStore) CountUsersByEmail(ctx context.Context, email string) (int64, error) { return 0, nil }
func (m *httpRiskMockStore) CreateBoardColumn(ctx context.Context, arg db.CreateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }

func (m *httpRiskMockStore) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
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

func (m *httpRiskMockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
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
	}
	m.sessions[arg.TokenHash] = s
	return s, nil
}

func (m *httpRiskMockStore) CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *httpRiskMockStore) CreateTaskComment(ctx context.Context, arg db.CreateTaskCommentParams) (db.TaskComment, error) { return db.TaskComment{}, nil }

func (m *httpRiskMockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
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
	}
	m.users[id.String()] = u
	return u, nil
}

func (m *httpRiskMockStore) DeleteBoardColumn(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpRiskMockStore) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *httpRiskMockStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error { return nil }
func (m *httpRiskMockStore) DeleteSessionsByUserID(ctx context.Context, userID pgtype.UUID) error { return nil }
func (m *httpRiskMockStore) GetBoardColumnByID(ctx context.Context, id pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpRiskMockStore) GetDoneColumn(ctx context.Context, projectID pgtype.UUID) (db.BoardColumn, error) { return db.BoardColumn{}, nil }

func (m *httpRiskMockStore) GetProjectByID(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	p, ok := m.projects[idStr]
	if !ok || p.DeletedAt.Valid {
		return db.Project{}, pgx.ErrNoRows
	}
	return p, nil
}

func (m *httpRiskMockStore) GetProjectByIDIncludeDeleted(ctx context.Context, id pgtype.UUID) (db.Project, error) {
	return m.GetProjectByID(ctx, id)
}

func (m *httpRiskMockStore) GetSessionWithUser(ctx context.Context, tokenHash string) (db.GetSessionWithUserRow, error) {
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
		ExpiresAt:          s.ExpiresAt,
		Email:              u.Email,
		FullName:           u.FullName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
	}, nil
}

func (m *httpRiskMockStore) GetTaskByID(ctx context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	t, ok := m.tasks[idStr]
	if !ok || t.DeletedAt.Valid {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (m *httpRiskMockStore) GetTaskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetTaskByIDWithDetailsRow, error) {
	return db.GetTaskByIDWithDetailsRow{}, pgx.ErrNoRows
}

func (m *httpRiskMockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}

func (m *httpRiskMockStore) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	u, ok := m.users[idStr]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *httpRiskMockStore) IncrementFailedLogin(ctx context.Context, id pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpRiskMockStore) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.AuditLog, error) { return db.AuditLog{}, nil }

func (m *httpRiskMockStore) IsProjectMember(ctx context.Context, arg db.IsProjectMemberParams) (bool, error) {
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

func (m *httpRiskMockStore) ListAllProjects(ctx context.Context) ([]db.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]db.Project, 0, len(m.projects))
	for _, p := range m.projects {
		if !p.DeletedAt.Valid {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *httpRiskMockStore) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) { return nil, nil }
func (m *httpRiskMockStore) ListBoardColumns(ctx context.Context, projectID pgtype.UUID) ([]db.BoardColumn, error) { return nil, nil }
func (m *httpRiskMockStore) ListProjectMembers(ctx context.Context, projectID pgtype.UUID) ([]db.ListProjectMembersRow, error) { return nil, nil }

func (m *httpRiskMockStore) ListProjectsByMember(ctx context.Context, userID pgtype.UUID) ([]db.Project, error) {
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

func (m *httpRiskMockStore) ListProjectsByOwner(ctx context.Context, ownerID pgtype.UUID) ([]db.Project, error) {
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

func (m *httpRiskMockStore) ListTaskComments(ctx context.Context, taskID pgtype.UUID) ([]db.ListTaskCommentsRow, error) { return nil, nil }
func (m *httpRiskMockStore) ListTasksByColumn(ctx context.Context, columnID pgtype.UUID) ([]db.ListTasksByColumnRow, error) { return nil, nil }
func (m *httpRiskMockStore) ListTasksByProject(ctx context.Context, projectID pgtype.UUID) ([]db.ListTasksByProjectRow, error) { return nil, nil }
func (m *httpRiskMockStore) ListTasksByProjectsFiltered(ctx context.Context, arg db.ListTasksByProjectsFilteredParams) ([]db.ListTasksByProjectsFilteredRow, error) { return nil, nil }
func (m *httpRiskMockStore) LockAccount(ctx context.Context, arg db.LockAccountParams) error { return nil }
func (m *httpRiskMockStore) MaxColumnPosition(ctx context.Context, projectID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpRiskMockStore) MaxTaskPositionInColumn(ctx context.Context, columnID pgtype.UUID) (int32, error) { return 0, nil }
func (m *httpRiskMockStore) NextProjectCode(ctx context.Context) (string, error) { return "PRJ-0001", nil }
func (m *httpRiskMockStore) RemoveProjectMember(ctx context.Context, arg db.RemoveProjectMemberParams) error { return nil }
func (m *httpRiskMockStore) ResetFailedLoginAndTouchLastLogin(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpRiskMockStore) ShiftTasksBetweenPositionsDown(ctx context.Context, arg db.ShiftTasksBetweenPositionsDownParams) error { return nil }
func (m *httpRiskMockStore) ShiftTasksBetweenPositionsUp(ctx context.Context, arg db.ShiftTasksBetweenPositionsUpParams) error { return nil }
func (m *httpRiskMockStore) ShiftTasksDownInColumn(ctx context.Context, arg db.ShiftTasksDownInColumnParams) error { return nil }
func (m *httpRiskMockStore) ShiftTasksUpInColumn(ctx context.Context, arg db.ShiftTasksUpInColumnParams) error { return nil }
func (m *httpRiskMockStore) SoftDeleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error) { return db.Task{}, nil }
func (m *httpRiskMockStore) TouchSessionLastSeen(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *httpRiskMockStore) TransferProjectOwner(ctx context.Context, arg db.TransferProjectOwnerParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpRiskMockStore) UpdateBoardColumn(ctx context.Context, arg db.UpdateBoardColumnParams) (db.BoardColumn, error) { return db.BoardColumn{}, nil }
func (m *httpRiskMockStore) UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error { return nil }
func (m *httpRiskMockStore) UpdateProject(ctx context.Context, arg db.UpdateProjectParams) (db.Project, error) { return db.Project{}, nil }
func (m *httpRiskMockStore) UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error) { return db.Task{}, nil }
func (m *httpRiskMockStore) UpdateTaskPositionAndColumn(ctx context.Context, arg db.UpdateTaskPositionAndColumnParams) (db.Task, error) { return db.Task{}, nil }

// Risk-specific methods
func (m *httpRiskMockStore) GetNextRiskCode(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	code := fmt.Sprintf("RSK-%04d", m.riskSeq)
	m.riskSeq++
	return code, nil
}

func (m *httpRiskMockStore) CreateRisk(ctx context.Context, arg db.CreateRiskParams) (db.Risk, error) {
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

func (m *httpRiskMockStore) GetRiskByID(ctx context.Context, id pgtype.UUID) (db.Risk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idStr := uuid.UUID(id.Bytes).String()
	r, ok := m.risks[idStr]
	if !ok {
		return db.Risk{}, pgx.ErrNoRows
	}
	return r, nil
}

func (m *httpRiskMockStore) GetRiskByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetRiskByIDWithDetailsRow, error) {
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
	}, nil
}

func (m *httpRiskMockStore) AssessRisk(ctx context.Context, arg db.AssessRiskParams) (db.Risk, error) {
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

func (m *httpRiskMockStore) CloseRisk(ctx context.Context, id pgtype.UUID) (db.Risk, error) {
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

func (m *httpRiskMockStore) UpdateRiskByRiskManagement(ctx context.Context, arg db.UpdateRiskByRiskManagementParams) (db.Risk, error) {
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

func (m *httpRiskMockStore) UpdateRiskMitigationPlan(ctx context.Context, arg db.UpdateRiskMitigationPlanParams) (db.Risk, error) {
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

func (m *httpRiskMockStore) ListRisksFiltered(ctx context.Context, arg db.ListRisksFilteredParams) ([]db.ListRisksFilteredRow, error) {
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

func (m *httpRiskMockStore) CountRisksFiltered(ctx context.Context, arg db.CountRisksFilteredParams) (int64, error) {
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

func (m *httpRiskMockStore) GetRisksHeatmapCounts(ctx context.Context, arg db.GetRisksHeatmapCountsParams) ([]db.GetRisksHeatmapCountsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := make(map[string]int64)
	for _, r := range m.risks {
		if r.Likelihood.Valid && r.Impact.Valid {
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

func (m *httpRiskMockStore) GetRisksSummaryStats(ctx context.Context, arg db.GetRisksSummaryStatsParams) (db.GetRisksSummaryStatsRow, error) {
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
		}
	}
	return row, nil
}

func (m *httpRiskMockStore) CountRisksByTaskIDs(ctx context.Context, taskIds []pgtype.UUID) ([]db.CountRisksByTaskIDsRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []db.CountRisksByTaskIDsRow
	for _, tUUID := range taskIds {
		res = append(res, db.CountRisksByTaskIDsRow{
			TaskID:            tUUID,
			Count:             1,
			HighCriticalCount: 0,
		})
	}
	return res, nil
}

// Phase 7 Querier stubs
func (m *httpRiskMockStore) CloseAuditFinding(ctx context.Context, arg db.CloseAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpRiskMockStore) CountAuditFindingsFiltered(ctx context.Context, arg db.CountAuditFindingsFilteredParams) (int64, error) { return 0, nil }
func (m *httpRiskMockStore) CreateAuditFinding(ctx context.Context, arg db.CreateAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpRiskMockStore) GetAuditFindingByID(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpRiskMockStore) GetAuditFindingByIDWithDetails(ctx context.Context, id pgtype.UUID) (db.GetAuditFindingByIDWithDetailsRow, error) { return db.GetAuditFindingByIDWithDetailsRow{}, nil }
func (m *httpRiskMockStore) GetAuditFindingsSummaryStats(ctx context.Context, arg db.GetAuditFindingsSummaryStatsParams) (db.GetAuditFindingsSummaryStatsRow, error) { return db.GetAuditFindingsSummaryStatsRow{}, nil }
func (m *httpRiskMockStore) GetNextFindingCode(ctx context.Context) (string, error) { return "FND-0001", nil }
func (m *httpRiskMockStore) ListAuditFindingsFiltered(ctx context.Context, arg db.ListAuditFindingsFilteredParams) ([]db.ListAuditFindingsFilteredRow, error) { return nil, nil }
func (m *httpRiskMockStore) ReopenAuditFinding(ctx context.Context, id pgtype.UUID) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpRiskMockStore) RespondAuditFinding(ctx context.Context, arg db.RespondAuditFindingParams) (db.AuditFinding, error) { return db.AuditFinding{}, nil }
func (m *httpRiskMockStore) GetUserStats(ctx context.Context) (db.GetUserStatsRow, error) { return db.GetUserStatsRow{}, nil }
func (m *httpRiskMockStore) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) { return nil, nil }

// ─── HTTP Setup Helper ───────────────────────────────────────────────────────

func setupRiskHTTPServer(t *testing.T) (http.Handler, *httpRiskMockStore, *auth.Service, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newHTTPRiskMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authSvc := auth.NewService(store, secret, logger, nil)
	projSvc := project.NewService(nil, store, nil, logger)
	riskSvc := risk.NewService(nil, store, nil, logger)

	router := NewRouter(RouterParams{
		Logger:         logger,
		AuthService:    authSvc,
		ProjectService: projSvc,
		RiskService:    riskSvc,
		SessionSecret:  secret,
		CookieSecure:   false,
	})

	return router, store, authSvc, secret
}

func createRiskTestUser(t *testing.T, store *httpRiskMockStore, email, name string, role db.UserRole) (db.User, string) {
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

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestHTTP_CreateRisk_RBAC(t *testing.T) {
	handler, store, _, _ := setupRiskHTTPServer(t)

	pm, pmToken := createRiskTestUser(t, store, "pm@bank.com", "Project Manager", db.UserRoleProjectManager)
	tl, tlToken := createRiskTestUser(t, store, "tl@bank.com", "Team Leader", db.UserRoleTeamLeader)
	_, rmToken := createRiskTestUser(t, store, "rm@bank.com", "Risk Manager", db.UserRoleRiskManagement)
	_, auditToken := createRiskTestUser(t, store, "audit@bank.com", "Audit Officer", db.UserRoleAudit)
	_, adminToken := createRiskTestUser(t, store, "admin@bank.com", "System Admin", db.UserRoleAdmin)

	ctx := context.Background()
	p, _ := store.CreateProject(ctx, db.CreateProjectParams{
		Code:      "PRJ-0001",
		Name:      "Loan Origination System",
		Status:    db.ProjectStatusActive,
		OwnerID:   pm.ID,
		CreatedBy: pm.ID,
	})
	store.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: p.ID,
		UserID:    tl.ID,
	})
	projIDStr := uuid.UUID(p.ID.Bytes).String()

	// 1. PM on own project -> 201 Created
	t.Run("PM creates risk on own project -> 201", func(t *testing.T) {
		body := map[string]any{
			"project_id":  projIDStr,
			"title":       "Credit scoring latency risk",
			"category":    "technical",
			"description": "Scoring engine takes over 5s",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/risks", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, pmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 2. TL on assigned project -> 201 Created
	t.Run("TL creates risk on assigned project -> 201", func(t *testing.T) {
		body := map[string]any{
			"project_id": projIDStr,
			"title":      "Resource availability constraint",
			"category":   "operational",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/risks", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, tlToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 3. RM on any project -> 201 Created
	t.Run("RM creates risk on project -> 201", func(t *testing.T) {
		body := map[string]any{
			"project_id": projIDStr,
			"title":      "Regulatory compliance deviation",
			"category":   "compliance",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/risks", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, rmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 4. Admin and Audit -> 403 Forbidden
	t.Run("Admin and Audit cannot raise risk -> 403", func(t *testing.T) {
		body := map[string]any{
			"project_id": projIDStr,
			"title":      "Admin raised risk",
			"category":   "financial",
		}
		jsonBytes, _ := json.Marshal(body)

		// Admin
		req := httptest.NewRequest(http.MethodPost, "/api/v1/risks", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, adminToken)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for admin, got %d", rec.Code)
		}

		// Audit
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/risks", bytes.NewReader(jsonBytes))
		req2.Header.Set("Content-Type", "application/json")
		addSessionCookie(req2, auditToken)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusForbidden {
			t.Errorf("expected 403 for audit, got %d", rec2.Code)
		}
	})

	// 5. Unauthenticated -> 401 Unauthorized
	t.Run("Unauthenticated cannot raise risk -> 401", func(t *testing.T) {
		body := map[string]any{"project_id": projIDStr, "title": "No auth"}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/risks", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}

func TestHTTP_Assess_And_Close_Risk(t *testing.T) {
	handler, store, _, _ := setupRiskHTTPServer(t)

	pm, pmToken := createRiskTestUser(t, store, "pm@bank.com", "Project Manager", db.UserRoleProjectManager)
	_, rmToken := createRiskTestUser(t, store, "rm@bank.com", "Risk Manager", db.UserRoleRiskManagement)

	ctx := context.Background()
	p, _ := store.CreateProject(ctx, db.CreateProjectParams{
		Code:      "PRJ-0001",
		Name:      "Loan System",
		Status:    db.ProjectStatusActive,
		OwnerID:   pm.ID,
		CreatedBy: pm.ID,
	})

	createdRisk, err := store.CreateRisk(ctx, db.CreateRiskParams{
		Code:        "RSK-0001",
		ProjectID:   p.ID,
		Title:       "Critical vulnerability in payment gateway",
		Category:    db.RiskCategorySecurity,
		Status:      db.RiskStatusIdentified,
		RaisedBy:    pm.ID,
	})
	if err != nil {
		t.Fatalf("failed to create mock risk: %v", err)
	}
	riskIDStr := uuid.UUID(createdRisk.ID.Bytes).String()

	// 1. PM cannot assess risk -> 403
	t.Run("PM cannot assess risk -> 403", func(t *testing.T) {
		body := map[string]any{"likelihood": 5, "impact": 5}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/risks/%s/assess", riskIDStr), bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, pmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	// 2. RM assesses risk -> 200 OK (Calculated score 25, level critical)
	t.Run("RM assesses risk -> 200", func(t *testing.T) {
		body := map[string]any{"likelihood": 5, "impact": 5}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/risks/%s/assess", riskIDStr), bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, rmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Score int32  `json:"score"`
				Level string `json:"level"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if res.Data.Score != 25 {
			t.Errorf("expected score 25, got %d", res.Data.Score)
		}
		if res.Data.Level != "critical" {
			t.Errorf("expected level critical, got %s", res.Data.Level)
		}
	})

	// 3. RM closes critical risk without mitigation plan -> 400 Bad Request
	t.Run("RM closes critical risk without mitigation plan -> 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/risks/%s/close", riskIDStr), nil)
		addSessionCookie(req, rmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 4. PM adds mitigation plan via PATCH -> 200 OK
	t.Run("PM updates mitigation plan -> 200", func(t *testing.T) {
		body := map[string]any{"mitigation_plan": "Isolate payment gateway on dedicated subnet"}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/risks/%s", riskIDStr), bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		addSessionCookie(req, pmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 5. Now RM closes risk -> 200 OK
	t.Run("RM closes critical risk after mitigation plan -> 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/risks/%s/close", riskIDStr), nil)
		addSessionCookie(req, rmToken)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHTTP_GetRiskSummary(t *testing.T) {
	handler, store, _, _ := setupRiskHTTPServer(t)
	_, rmToken := createRiskTestUser(t, store, "rm@bank.com", "Risk Manager", db.UserRoleRiskManagement)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/risks/summary", nil)
	addSessionCookie(req, rmToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool `json:"success"`
		Data    risk.RiskSummaryDTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.Data.Heatmap) != 25 {
		t.Errorf("expected 25 heatmap cells in 5x5 grid, got %d", len(res.Data.Heatmap))
	}
}
