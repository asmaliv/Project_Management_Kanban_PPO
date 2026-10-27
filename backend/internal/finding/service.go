package finding

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/db"
)

var (
	ErrFindingNotFound          = errors.New("temuan audit tidak ditemukan")
	ErrForbidden                = errors.New("anda tidak memiliki izin untuk melakukan aksi ini")
	ErrProjectNotFound          = errors.New("proyek tidak ditemukan")
	ErrTaskNotFound             = errors.New("task tidak ditemukan")
	ErrProjectArchived          = errors.New("proyek sudah diarsipkan dan bersifat read-only")
	ErrInvalidInput             = errors.New("input data temuan audit tidak valid")
	ErrInvalidStatusTransition  = errors.New("transisi status temuan tidak valid")
	ErrOnlyAuditCanCreate       = errors.New("hanya auditor yang dapat membuat temuan audit")
	ErrOnlyAuditCanClose        = errors.New("hanya auditor yang dapat menutup temuan audit")
	ErrOnlyAuditCanReopen       = errors.New("hanya auditor yang dapat membuka kembali temuan audit")
	ErrOnlyPMCanRespond         = errors.New("hanya Project Manager pemilik proyek yang dapat merespons temuan audit")
	ErrMustBeRespondedBeforeClose = errors.New("temuan audit hanya bisa ditutup setelah berstatus responded")
)

// ActorInfo carries authenticated actor metadata.
type ActorInfo struct {
	ID        string
	Role      db.UserRole
	RequestID string
	IP        string
}

// FindingDTO represents an audit finding returned to client.
type FindingDTO struct {
	ID              string     `json:"id"`
	Code            string     `json:"code"`
	ProjectID       string     `json:"project_id"`
	ProjectName     string     `json:"project_name,omitempty"`
	ProjectCode     string     `json:"project_code,omitempty"`
	TaskID          *string    `json:"task_id"`
	TaskTitle       *string    `json:"task_title,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Severity        string     `json:"severity"`
	Status          string     `json:"status"`
	RaisedBy        string     `json:"raised_by"`
	RaisedByName    string     `json:"raised_by_name,omitempty"`
	Response        string     `json:"response"`
	RespondedBy     *string    `json:"responded_by"`
	RespondedByName *string    `json:"responded_by_name,omitempty"`
	RespondedAt     *time.Time `json:"responded_at"`
	ClosedBy        *string    `json:"closed_by"`
	ClosedByName    *string    `json:"closed_by_name,omitempty"`
	ClosedAt        *time.Time `json:"closed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateFindingParams holds input for creating a new audit finding.
type CreateFindingParams struct {
	ProjectID   string  `json:"project_id"`
	TaskID      *string `json:"task_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
}

// RespondFindingParams holds input for responding to an audit finding.
type RespondFindingParams struct {
	Response string `json:"response"`
}

// ListFindingsParams holds filter criteria for listing findings.
type ListFindingsParams struct {
	ProjectID *string `json:"project_id"`
	TaskID    *string `json:"task_id"`
	Severity  *string `json:"severity"`
	Status    *string `json:"status"`
	Search    *string `json:"search"`
	Limit     int32   `json:"limit"`
	Offset    int32   `json:"offset"`
}

// FindingSummaryDTO represents statistical aggregations of audit findings.
type FindingSummaryDTO struct {
	TotalFindings int64 `json:"total_findings"`
	OpenFindings  int64 `json:"open_findings"`
	CriticalCount int64 `json:"critical_count"`
	HighCount     int64 `json:"high_count"`
	MediumCount   int64 `json:"medium_count"`
	LowCount      int64 `json:"low_count"`
	RespondedCount int64 `json:"responded_count"`
}

// Service provides audit finding business logic with strict banking-grade RBAC and audit logging.
type Service struct {
	pool     *pgxpool.Pool
	queries  Store
	auditLog *auditlog.Service
	logger   *slog.Logger
}

// NewService instantiates a finding Service.
func NewService(pool *pgxpool.Pool, queries Store, auditLog *auditlog.Service, logger *slog.Logger) *Service {
	return &Service{
		pool:     pool,
		queries:  queries,
		auditLog: auditLog,
		logger:   logger,
	}
}

// CreateFinding raises a new finding on a project/task.
// Permission: Only audit role can create findings.
func (s *Service) CreateFinding(ctx context.Context, actor ActorInfo, params CreateFindingParams) (FindingDTO, error) {
	if actor.Role != db.UserRoleAudit {
		return FindingDTO{}, fmt.Errorf("%w: %s", ErrForbidden, ErrOnlyAuditCanCreate)
	}

	title := strings.TrimSpace(params.Title)
	if title == "" {
		return FindingDTO{}, fmt.Errorf("%w: judul temuan audit wajib diisi", ErrInvalidInput)
	}
	if len(title) > 255 {
		return FindingDTO{}, fmt.Errorf("%w: judul temuan audit maksimal 255 karakter", ErrInvalidInput)
	}

	severity := db.FindingSeverity(strings.ToLower(strings.TrimSpace(params.Severity)))
	switch severity {
	case db.FindingSeverityLow, db.FindingSeverityMedium, db.FindingSeverityHigh, db.FindingSeverityCritical:
	default:
		return FindingDTO{}, fmt.Errorf("%w: severity harus bernilai low, medium, high, atau critical", ErrInvalidInput)
	}

	projUUID, err := parseUUID(params.ProjectID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("%w: format project_id tidak valid", ErrInvalidInput)
	}

	project, err := s.queries.GetProjectByID(ctx, projUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingDTO{}, ErrProjectNotFound
		}
		return FindingDTO{}, fmt.Errorf("get project: %w", err)
	}

	if project.Status == db.ProjectStatusArchived {
		return FindingDTO{}, ErrProjectArchived
	}

	var taskUUID pgtype.UUID
	if params.TaskID != nil && strings.TrimSpace(*params.TaskID) != "" {
		tUUID, err := parseUUID(*params.TaskID)
		if err != nil {
			return FindingDTO{}, fmt.Errorf("%w: format task_id tidak valid", ErrInvalidInput)
		}
		t, err := s.queries.GetTaskByID(ctx, tUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return FindingDTO{}, ErrTaskNotFound
			}
			return FindingDTO{}, fmt.Errorf("get task: %w", err)
		}
		if t.ProjectID != projUUID {
			return FindingDTO{}, fmt.Errorf("%w: task tidak berada pada proyek yang dipilih", ErrInvalidInput)
		}
		taskUUID = tUUID
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	code, err := s.queries.GetNextFindingCode(ctx)
	if err != nil {
		// Fallback code format
		code = fmt.Sprintf("FND-%04d", time.Now().Unix()%10000)
	}

	finding, err := s.queries.CreateAuditFinding(ctx, db.CreateAuditFindingParams{
		Code:        code,
		ProjectID:   projUUID,
		TaskID:      taskUUID,
		Title:       title,
		Description: strings.TrimSpace(params.Description),
		Severity:    severity,
		Status:      db.FindingStatusOpen,
		RaisedBy:    actorUUID,
	})
	if err != nil {
		return FindingDTO{}, fmt.Errorf("create audit finding: %w", err)
	}

	// Transactional Audit log
	if s.auditLog != nil {
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "finding.create",
			EntityType: "audit_finding",
			EntityID:   uuidString(finding.ID),
			ProjectID:  projUUID,
			After: map[string]any{
				"code":        finding.Code,
				"title":       finding.Title,
				"severity":    string(finding.Severity),
				"status":      string(finding.Status),
				"project_id":  uuidString(finding.ProjectID),
				"task_id":     uuidString(finding.TaskID),
				"raised_by":   uuidString(finding.RaisedBy),
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		})
	}

	return s.GetFinding(ctx, actor, uuidString(finding.ID))
}

// RespondFinding allows PM owner to respond to an open audit finding.
func (s *Service) RespondFinding(ctx context.Context, actor ActorInfo, findingID string, params RespondFindingParams) (FindingDTO, error) {
	if actor.Role != db.UserRoleProjectManager {
		return FindingDTO{}, fmt.Errorf("%w: %s", ErrForbidden, ErrOnlyPMCanRespond)
	}

	fUUID, err := parseUUID(findingID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("%w: format id temuan tidak valid", ErrInvalidInput)
	}

	finding, err := s.queries.GetAuditFindingByIDWithDetails(ctx, fUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingDTO{}, ErrFindingNotFound
		}
		return FindingDTO{}, fmt.Errorf("get finding: %w", err)
	}

	// Check project ownership
	if uuidString(finding.ProjectOwnerID) != actor.ID {
		return FindingDTO{}, fmt.Errorf("%w: %s", ErrForbidden, ErrOnlyPMCanRespond)
	}

	// Status must be open
	if finding.Status != db.FindingStatusOpen {
		return FindingDTO{}, fmt.Errorf("%w: hanya temuan berstatus open yang dapat direspons", ErrInvalidStatusTransition)
	}

	resp := strings.TrimSpace(params.Response)
	if resp == "" {
		return FindingDTO{}, fmt.Errorf("%w: isi tanggapan/respons tidak boleh kosong", ErrInvalidInput)
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	updated, err := s.queries.RespondAuditFinding(ctx, db.RespondAuditFindingParams{
		ID:           fUUID,
		Response:     resp,
		RespondedBy:  actorUUID,
	})
	if err != nil {
		return FindingDTO{}, fmt.Errorf("respond finding: %w", err)
	}

	if s.auditLog != nil {
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "finding.respond",
			EntityType: "audit_finding",
			EntityID:   uuidString(updated.ID),
			ProjectID:  updated.ProjectID,
			Before: map[string]any{
				"status":   string(finding.Status),
				"response": finding.Response,
			},
			After: map[string]any{
				"status":       string(updated.Status),
				"response":     updated.Response,
				"responded_by": uuidString(updated.RespondedBy),
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		})
	}

	return s.GetFinding(ctx, actor, findingID)
}

// ReopenFinding allows auditor to reopen a responded or closed finding.
func (s *Service) ReopenFinding(ctx context.Context, actor ActorInfo, findingID string) (FindingDTO, error) {
	if actor.Role != db.UserRoleAudit {
		return FindingDTO{}, fmt.Errorf("%w: %s", ErrForbidden, ErrOnlyAuditCanReopen)
	}

	fUUID, err := parseUUID(findingID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("%w: format id temuan tidak valid", ErrInvalidInput)
	}

	finding, err := s.queries.GetAuditFindingByID(ctx, fUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingDTO{}, ErrFindingNotFound
		}
		return FindingDTO{}, fmt.Errorf("get finding: %w", err)
	}

	if finding.Status == db.FindingStatusOpen {
		return FindingDTO{}, fmt.Errorf("%w: temuan sudah berstatus open", ErrInvalidStatusTransition)
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	updated, err := s.queries.ReopenAuditFinding(ctx, fUUID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("reopen finding: %w", err)
	}

	if s.auditLog != nil {
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "finding.reopen",
			EntityType: "audit_finding",
			EntityID:   uuidString(updated.ID),
			ProjectID:  updated.ProjectID,
			Before: map[string]any{
				"status": string(finding.Status),
			},
			After: map[string]any{
				"status": string(updated.Status),
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		})
	}

	return s.GetFinding(ctx, actor, findingID)
}

// CloseFinding allows auditor to close a finding that has been responded to.
// Rule: Direct transition from open to closed is rejected!
func (s *Service) CloseFinding(ctx context.Context, actor ActorInfo, findingID string) (FindingDTO, error) {
	if actor.Role != db.UserRoleAudit {
		return FindingDTO{}, fmt.Errorf("%w: %s", ErrForbidden, ErrOnlyAuditCanClose)
	}

	fUUID, err := parseUUID(findingID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("%w: format id temuan tidak valid", ErrInvalidInput)
	}

	finding, err := s.queries.GetAuditFindingByID(ctx, fUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingDTO{}, ErrFindingNotFound
		}
		return FindingDTO{}, fmt.Errorf("get finding: %w", err)
	}

	if finding.Status == db.FindingStatusClosed {
		return FindingDTO{}, fmt.Errorf("%w: temuan sudah berstatus closed", ErrInvalidStatusTransition)
	}

	if finding.Status != db.FindingStatusResponded {
		return FindingDTO{}, fmt.Errorf("%w: %s", ErrInvalidStatusTransition, ErrMustBeRespondedBeforeClose)
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	updated, err := s.queries.CloseAuditFinding(ctx, db.CloseAuditFindingParams{
		ID:       fUUID,
		ClosedBy: actorUUID,
	})
	if err != nil {
		return FindingDTO{}, fmt.Errorf("close finding: %w", err)
	}

	if s.auditLog != nil {
		_ = s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "finding.close",
			EntityType: "audit_finding",
			EntityID:   uuidString(updated.ID),
			ProjectID:  updated.ProjectID,
			Before: map[string]any{
				"status": string(finding.Status),
			},
			After: map[string]any{
				"status":    string(updated.Status),
				"closed_by": uuidString(updated.ClosedBy),
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		})
	}

	return s.GetFinding(ctx, actor, findingID)
}

// GetFinding retrieves finding detail respecting role visibility constraints.
func (s *Service) GetFinding(ctx context.Context, actor ActorInfo, findingID string) (FindingDTO, error) {
	fUUID, err := parseUUID(findingID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("%w: format id temuan tidak valid", ErrInvalidInput)
	}

	f, err := s.queries.GetAuditFindingByIDWithDetails(ctx, fUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingDTO{}, ErrFindingNotFound
		}
		return FindingDTO{}, fmt.Errorf("get finding details: %w", err)
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return FindingDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	// Verify visibility according to role
	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		// Can see any project's findings
	case db.UserRoleProjectManager:
		if f.ProjectOwnerID != actorUUID {
			return FindingDTO{}, ErrFindingNotFound
		}
	case db.UserRoleTeamLeader:
		isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
			ProjectID: f.ProjectID,
			UserID:    actorUUID,
		})
		if err != nil {
			return FindingDTO{}, fmt.Errorf("check membership: %w", err)
		}
		if !isMember {
			return FindingDTO{}, ErrFindingNotFound
		}
	default:
		return FindingDTO{}, ErrForbidden
	}

	return toFindingDTO(f), nil
}

// ListFindings lists findings filtered and scoped according to role permissions.
func (s *Service) ListFindings(ctx context.Context, actor ActorInfo, params ListFindingsParams) ([]FindingDTO, int64, error) {
	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid actor id: %w", err)
	}

	var scopedProjectIDs []pgtype.UUID
	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		// Can see across all projects, scopedProjectIDs remains nil
	case db.UserRoleProjectManager:
		owned, err := s.queries.ListProjectsByOwner(ctx, actorUUID)
		if err != nil {
			return nil, 0, fmt.Errorf("list owned projects: %w", err)
		}
		if len(owned) == 0 {
			return []FindingDTO{}, 0, nil
		}
		for _, p := range owned {
			scopedProjectIDs = append(scopedProjectIDs, p.ID)
		}
	case db.UserRoleTeamLeader:
		memberProjects, err := s.queries.ListProjectsByMember(ctx, actorUUID)
		if err != nil {
			return nil, 0, fmt.Errorf("list member projects: %w", err)
		}
		if len(memberProjects) == 0 {
			return []FindingDTO{}, 0, nil
		}
		for _, p := range memberProjects {
			scopedProjectIDs = append(scopedProjectIDs, p.ID)
		}
	default:
		return nil, 0, ErrForbidden
	}

	var projUUID pgtype.UUID
	if params.ProjectID != nil && strings.TrimSpace(*params.ProjectID) != "" {
		pUUID, err := parseUUID(*params.ProjectID)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: invalid project_id filter", ErrInvalidInput)
		}
		// If scoped, ensure requested project is within caller's scope
		if len(scopedProjectIDs) > 0 {
			found := false
			for _, id := range scopedProjectIDs {
				if id == pUUID {
					found = true
					break
				}
			}
			if !found {
				return []FindingDTO{}, 0, nil
			}
		}
		projUUID = pUUID
	}

	var taskUUID pgtype.UUID
	if params.TaskID != nil && strings.TrimSpace(*params.TaskID) != "" {
		tUUID, err := parseUUID(*params.TaskID)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: invalid task_id filter", ErrInvalidInput)
		}
		taskUUID = tUUID
	}

	var severity db.NullFindingSeverity
	if params.Severity != nil && strings.TrimSpace(*params.Severity) != "" {
		sev := db.FindingSeverity(strings.ToLower(strings.TrimSpace(*params.Severity)))
		severity = db.NullFindingSeverity{FindingSeverity: sev, Valid: true}
	}

	var status db.NullFindingStatus
	if params.Status != nil && strings.TrimSpace(*params.Status) != "" {
		st := db.FindingStatus(strings.ToLower(strings.TrimSpace(*params.Status)))
		status = db.NullFindingStatus{FindingStatus: st, Valid: true}
	}

	var search pgtype.Text
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		search = pgtype.Text{String: strings.TrimSpace(*params.Search), Valid: true}
	}

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.queries.ListAuditFindingsFiltered(ctx, db.ListAuditFindingsFilteredParams{
		ProjectIds: scopedProjectIDs,
		ProjectID:  projUUID,
		TaskID:     taskUUID,
		Severity:   severity,
		Status:     status,
		Search:     search,
		LimitVal:   limit,
		OffsetVal:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit findings: %w", err)
	}

	total, err := s.queries.CountAuditFindingsFiltered(ctx, db.CountAuditFindingsFilteredParams{
		ProjectIds: scopedProjectIDs,
		ProjectID:  projUUID,
		TaskID:     taskUUID,
		Severity:   severity,
		Status:     status,
		Search:     search,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count audit findings: %w", err)
	}

	res := make([]FindingDTO, len(rows))
	for i, r := range rows {
		res[i] = toFindingDTO(db.GetAuditFindingByIDWithDetailsRow(r))
	}

	return res, total, nil
}

// GetFindingSummary returns findings breakdown counts scoped to the caller.
func (s *Service) GetFindingSummary(ctx context.Context, actor ActorInfo, projectID *string) (FindingSummaryDTO, error) {
	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return FindingSummaryDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	var scopedProjectIDs []pgtype.UUID
	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
	case db.UserRoleProjectManager:
		owned, err := s.queries.ListProjectsByOwner(ctx, actorUUID)
		if err != nil {
			return FindingSummaryDTO{}, fmt.Errorf("list owned projects: %w", err)
		}
		if len(owned) == 0 {
			return FindingSummaryDTO{}, nil
		}
		for _, p := range owned {
			scopedProjectIDs = append(scopedProjectIDs, p.ID)
		}
	case db.UserRoleTeamLeader:
		memberProjects, err := s.queries.ListProjectsByMember(ctx, actorUUID)
		if err != nil {
			return FindingSummaryDTO{}, fmt.Errorf("list member projects: %w", err)
		}
		if len(memberProjects) == 0 {
			return FindingSummaryDTO{}, nil
		}
		for _, p := range memberProjects {
			scopedProjectIDs = append(scopedProjectIDs, p.ID)
		}
	default:
		return FindingSummaryDTO{}, ErrForbidden
	}

	var projUUID pgtype.UUID
	if projectID != nil && strings.TrimSpace(*projectID) != "" {
		pUUID, err := parseUUID(*projectID)
		if err != nil {
			return FindingSummaryDTO{}, fmt.Errorf("%w: invalid project_id filter", ErrInvalidInput)
		}
		projUUID = pUUID
	}

	stats, err := s.queries.GetAuditFindingsSummaryStats(ctx, db.GetAuditFindingsSummaryStatsParams{
		ProjectIds: scopedProjectIDs,
		ProjectID:  projUUID,
	})
	if err != nil {
		return FindingSummaryDTO{}, fmt.Errorf("get audit findings summary: %w", err)
	}

	return FindingSummaryDTO{
		TotalFindings:  stats.TotalFindings,
		OpenFindings:   stats.OpenFindings,
		CriticalCount:  stats.CriticalCount,
		HighCount:      stats.HighCount,
		MediumCount:    stats.MediumCount,
		LowCount:       stats.LowCount,
		RespondedCount: stats.RespondedCount,
	}, nil
}

// ─── Internal Helpers ────────────────────────────────────────────────────────

func toFindingDTO(f db.GetAuditFindingByIDWithDetailsRow) FindingDTO {
	dto := FindingDTO{
		ID:           uuidString(f.ID),
		Code:         f.Code,
		ProjectID:    uuidString(f.ProjectID),
		ProjectName:  f.ProjectName,
		ProjectCode:  f.ProjectCode,
		Title:        f.Title,
		Description:  f.Description,
		Severity:     string(f.Severity),
		Status:       string(f.Status),
		RaisedBy:     uuidString(f.RaisedBy),
		RaisedByName: f.RaisedByName,
		Response:     f.Response,
		CreatedAt:    f.CreatedAt.Time,
		UpdatedAt:    f.UpdatedAt.Time,
	}

	if f.TaskID.Valid {
		tid := uuidString(f.TaskID)
		dto.TaskID = &tid
	}
	if f.TaskTitle.Valid {
		dto.TaskTitle = &f.TaskTitle.String
	}
	if f.RespondedBy.Valid {
		rb := uuidString(f.RespondedBy)
		dto.RespondedBy = &rb
	}
	if f.RespondedByName.Valid {
		dto.RespondedByName = &f.RespondedByName.String
	}
	if f.RespondedAt.Valid {
		dto.RespondedAt = &f.RespondedAt.Time
	}
	if f.ClosedBy.Valid {
		cb := uuidString(f.ClosedBy)
		dto.ClosedBy = &cb
	}
	if f.ClosedByName.Valid {
		dto.ClosedByName = &f.ClosedByName.String
	}
	if f.ClosedAt.Valid {
		dto.ClosedAt = &f.ClosedAt.Time
	}

	return dto
}

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
