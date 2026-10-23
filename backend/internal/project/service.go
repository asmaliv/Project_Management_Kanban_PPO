package project

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
	ErrProjectNotFound     = errors.New("proyek tidak ditemukan")
	ErrForbidden           = errors.New("anda tidak memiliki izin untuk melakukan aksi ini")
	ErrProjectArchived     = errors.New("proyek sudah diarsipkan, tidak dapat diubah")
	ErrColumnDoneProtected = errors.New("kolom Done tidak dapat dihapus")
	ErrOneDoneColumnOnly   = errors.New("proyek sudah memiliki kolom Done")
	ErrMemberNotTeamLeader = errors.New("hanya user dengan role team_leader yang dapat ditambahkan sebagai member")
	ErrTransferTargetNotPM = errors.New("target transfer harus memiliki role project_manager")
	ErrColumnNotEmpty      = errors.New("kolom masih berisi task, tidak dapat dihapus")
	ErrInvalidOwner        = errors.New("owner proyek harus memiliki role project_manager")
)

// defaultColumns are created automatically for every new project.
var defaultColumns = []struct {
	name     string
	position int32
	isDone   bool
}{
	{"Backlog", 1, false},
	{"To Do", 2, false},
	{"In Progress", 3, false},
	{"In Review", 4, false},
	{"Done", 5, true},
}

// ProjectDTO is the response shape for a project.
type ProjectDTO struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	OwnerID     string    `json:"owner_id"`
	StartDate   *string   `json:"start_date"`
	EndDate     *string   `json:"end_date"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ColumnDTO is the response shape for a board column.
type ColumnDTO struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Name         string    `json:"name"`
	Position     int32     `json:"position"`
	IsDoneColumn bool      `json:"is_done_column"`
	WipLimit     *int32    `json:"wip_limit"`
	CreatedAt    time.Time `json:"created_at"`
}

// MemberDTO is the response shape for a project member.
type MemberDTO struct {
	ID       string    `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	AddedAt  time.Time `json:"added_at"`
}

// CreateProjectParams holds input for project creation.
type CreateProjectParams struct {
	Name        string
	Description string
	StartDate   *time.Time
	EndDate     *time.Time
}

// UpdateProjectParams holds input for project update.
type UpdateProjectParams struct {
	Name        string
	Description string
	Status      db.ProjectStatus
	StartDate   *time.Time
	EndDate     *time.Time
}

// CreateColumnParams holds input for column creation.
type CreateColumnParams struct {
	Name         string
	Position     int32
	IsDoneColumn bool
	WipLimit     *int32
}

// UpdateColumnParams holds input for column update.
type UpdateColumnParams struct {
	Name     string
	Position int32
	WipLimit *int32
}

// ActorInfo carries the authenticated user's identity for service calls.
type ActorInfo struct {
	ID        string
	Role      db.UserRole
	RequestID string
	IP        string
}

// Service handles project business logic.
type Service struct {
	pool     *pgxpool.Pool
	queries  Store
	auditLog *auditlog.Service
	logger   *slog.Logger
}

// NewService creates a project Service.
func NewService(pool *pgxpool.Pool, queries Store, auditLog *auditlog.Service, logger *slog.Logger) *Service {
	return &Service{
		pool:     pool,
		queries:  queries,
		auditLog: auditLog,
		logger:   logger,
	}
}

// ─── Projects ────────────────────────────────────────────────────────────────

// CreateProject creates a new project with default board columns. Only PM role allowed.
func (s *Service) CreateProject(ctx context.Context, actor ActorInfo, params CreateProjectParams) (ProjectDTO, error) {
	if actor.Role != db.UserRoleProjectManager {
		return ProjectDTO{}, ErrForbidden
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return ProjectDTO{}, errors.New("nama proyek wajib diisi")
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return ProjectDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	code, err := qtx.NextProjectCode(ctx)
	if err != nil {
		return ProjectDTO{}, fmt.Errorf("generate project code: %w", err)
	}

	actorUUID := mustParseUUID(actor.ID)

	p, err := qtx.CreateProject(ctx, db.CreateProjectParams{
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(params.Description),
		Status:      db.ProjectStatusActive,
		OwnerID:     actorUUID,
		StartDate:   toPgDate(params.StartDate),
		EndDate:     toPgDate(params.EndDate),
		CreatedBy:   actorUUID,
	})
	if err != nil {
		return ProjectDTO{}, fmt.Errorf("create project: %w", err)
	}

	// Create default board columns
	for _, col := range defaultColumns {
		var wip pgtype.Int4
		_, err := qtx.CreateBoardColumn(ctx, db.CreateBoardColumnParams{
			ProjectID:    p.ID,
			Name:         col.name,
			Position:     col.position,
			IsDoneColumn: col.isDone,
			WipLimit:     wip,
		})
		if err != nil {
			return ProjectDTO{}, fmt.Errorf("create default column %q: %w", col.name, err)
		}
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.create",
			EntityType: "project",
			EntityID:   uuid.UUID(p.ID.Bytes).String(),
			ProjectID:  p.ID,
			After:      map[string]any{"code": p.Code, "name": p.Name, "status": string(p.Status)},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return ProjectDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return ProjectDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	return toProjectDTO(p), nil
}

// ListProjects returns projects visible to the actor based on their role.
func (s *Service) ListProjects(ctx context.Context, actor ActorInfo) ([]ProjectDTO, error) {
	actorUUID := mustParseUUID(actor.ID)

	var rows []db.Project
	var err error

	switch actor.Role {
	case db.UserRoleProjectManager:
		rows, err = s.queries.ListProjectsByOwner(ctx, actorUUID)
	case db.UserRoleTeamLeader:
		rows, err = s.queries.ListProjectsByMember(ctx, actorUUID)
	default: // admin, audit, risk_management see all
		rows, err = s.queries.ListAllProjects(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	result := make([]ProjectDTO, 0, len(rows))
	for _, p := range rows {
		result = append(result, toProjectDTO(p))
	}
	return result, nil
}

// GetProject returns a single project if the actor has access.
func (s *Service) GetProject(ctx context.Context, actor ActorInfo, projectID string) (ProjectDTO, error) {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectDTO{}, ErrProjectNotFound
		}
		return ProjectDTO{}, fmt.Errorf("get project: %w", err)
	}

	if err := s.checkReadAccess(ctx, actor, p); err != nil {
		return ProjectDTO{}, err
	}
	return toProjectDTO(p), nil
}

// UpdateProject updates project metadata. Only the PM owner can update.
func (s *Service) UpdateProject(ctx context.Context, actor ActorInfo, projectID string, params UpdateProjectParams) (ProjectDTO, error) {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectDTO{}, ErrProjectNotFound
		}
		return ProjectDTO{}, fmt.Errorf("get project: %w", err)
	}

	if err := s.checkOwnerAccess(actor, p); err != nil {
		return ProjectDTO{}, err
	}
	if p.Status == db.ProjectStatusArchived {
		return ProjectDTO{}, ErrProjectArchived
	}

	before := map[string]any{"name": p.Name, "description": p.Description, "status": string(p.Status)}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return ProjectDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	updated, err := qtx.UpdateProject(ctx, db.UpdateProjectParams{
		ID:          p.ID,
		Name:        strings.TrimSpace(params.Name),
		Description: strings.TrimSpace(params.Description),
		Status:      params.Status,
		StartDate:   toPgDate(params.StartDate),
		EndDate:     toPgDate(params.EndDate),
	})
	if err != nil {
		return ProjectDTO{}, fmt.Errorf("update project: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.update",
			EntityType: "project",
			EntityID:   projectID,
			ProjectID:  p.ID,
			Before:     before,
			After:      map[string]any{"name": updated.Name, "description": updated.Description, "status": string(updated.Status)},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return ProjectDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return ProjectDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}
	return toProjectDTO(updated), nil
}

// ArchiveProject archives a project. Only the PM owner can archive.
func (s *Service) ArchiveProject(ctx context.Context, actor ActorInfo, projectID string) (ProjectDTO, error) {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectDTO{}, ErrProjectNotFound
		}
		return ProjectDTO{}, fmt.Errorf("get project: %w", err)
	}

	if err := s.checkOwnerAccess(actor, p); err != nil {
		return ProjectDTO{}, err
	}
	if p.Status == db.ProjectStatusArchived {
		return ProjectDTO{}, ErrProjectArchived
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return ProjectDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	archived, err := qtx.ArchiveProject(ctx, p.ID)
	if err != nil {
		return ProjectDTO{}, fmt.Errorf("archive project: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.archive",
			EntityType: "project",
			EntityID:   projectID,
			ProjectID:  p.ID,
			Before:     map[string]any{"status": string(p.Status)},
			After:      map[string]any{"status": "archived"},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return ProjectDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return ProjectDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}
	return toProjectDTO(archived), nil
}

// TransferOwner transfers project ownership. Only Admin can do this; target must be PM.
func (s *Service) TransferOwner(ctx context.Context, actor ActorInfo, projectID, newOwnerID string) (ProjectDTO, error) {
	if actor.Role != db.UserRoleAdmin {
		return ProjectDTO{}, ErrForbidden
	}

	newOwner, err := s.queries.GetUserByID(ctx, mustParseUUID(newOwnerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectDTO{}, errors.New("user target transfer tidak ditemukan")
		}
		return ProjectDTO{}, fmt.Errorf("get new owner user: %w", err)
	}
	if newOwner.Role != db.UserRoleProjectManager {
		return ProjectDTO{}, ErrTransferTargetNotPM
	}

	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectDTO{}, ErrProjectNotFound
		}
		return ProjectDTO{}, fmt.Errorf("get project: %w", err)
	}

	oldOwnerID := uuid.UUID(p.OwnerID.Bytes).String()

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return ProjectDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	updated, err := qtx.TransferProjectOwner(ctx, db.TransferProjectOwnerParams{
		ID:      p.ID,
		OwnerID: mustParseUUID(newOwnerID),
	})
	if err != nil {
		return ProjectDTO{}, fmt.Errorf("transfer owner: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.transfer_owner",
			EntityType: "project",
			EntityID:   projectID,
			ProjectID:  p.ID,
			Before:     map[string]any{"owner_id": oldOwnerID},
			After:      map[string]any{"owner_id": newOwnerID},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return ProjectDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return ProjectDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}
	return toProjectDTO(updated), nil
}

// ─── Members ─────────────────────────────────────────────────────────────────

// ListMembers returns project members. Access: PM owner, or any role that can read.
func (s *Service) ListMembers(ctx context.Context, actor ActorInfo, projectID string) ([]MemberDTO, error) {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	if err := s.checkReadAccess(ctx, actor, p); err != nil {
		return nil, err
	}

	rows, err := s.queries.ListProjectMembers(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	result := make([]MemberDTO, 0, len(rows))
	for _, m := range rows {
		result = append(result, MemberDTO{
			ID:       uuid.UUID(m.ID.Bytes).String(),
			FullName: m.FullName,
			Email:    m.Email,
			Role:     string(m.Role),
			AddedAt:  m.AddedAt.Time,
		})
	}
	return result, nil
}

// AddMember adds a team_leader user to the project. Only PM owner.
func (s *Service) AddMember(ctx context.Context, actor ActorInfo, projectID, memberUserID string) error {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("get project: %w", err)
	}
	if err := s.checkOwnerAccess(actor, p); err != nil {
		return err
	}
	if p.Status == db.ProjectStatusArchived {
		return ErrProjectArchived
	}

	member, err := s.queries.GetUserByID(ctx, mustParseUUID(memberUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("user tidak ditemukan")
		}
		return fmt.Errorf("get member user: %w", err)
	}
	if member.Role != db.UserRoleTeamLeader {
		return ErrMemberNotTeamLeader
	}

	memberID := uuid.UUID(member.ID.Bytes).String()

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	if err := qtx.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: p.ID,
		UserID:    member.ID,
		AddedBy:   mustParseUUID(actor.ID),
	}); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.member.add",
			EntityType: "project_member",
			EntityID:   memberID,
			ProjectID:  p.ID,
			After:      map[string]any{"project_id": projectID, "user_id": memberID},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
}

// RemoveMember removes a member from the project. Only PM owner.
func (s *Service) RemoveMember(ctx context.Context, actor ActorInfo, projectID, userID string) error {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("get project: %w", err)
	}
	if err := s.checkOwnerAccess(actor, p); err != nil {
		return err
	}
	if p.Status == db.ProjectStatusArchived {
		return ErrProjectArchived
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	if err := qtx.RemoveProjectMember(ctx, db.RemoveProjectMemberParams{
		ProjectID: p.ID,
		UserID:    mustParseUUID(userID),
	}); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.member.remove",
			EntityType: "project_member",
			EntityID:   userID,
			ProjectID:  p.ID,
			Before:     map[string]any{"project_id": projectID, "user_id": userID},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
}

// ─── Columns ─────────────────────────────────────────────────────────────────

// ListColumns returns all columns for a project.
func (s *Service) ListColumns(ctx context.Context, actor ActorInfo, projectID string) ([]ColumnDTO, error) {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	if err := s.checkReadAccess(ctx, actor, p); err != nil {
		return nil, err
	}

	cols, err := s.queries.ListBoardColumns(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}

	result := make([]ColumnDTO, 0, len(cols))
	for _, c := range cols {
		result = append(result, toColumnDTO(c))
	}
	return result, nil
}

// CreateColumn adds a new column to the board. Only PM owner.
func (s *Service) CreateColumn(ctx context.Context, actor ActorInfo, projectID string, params CreateColumnParams) (ColumnDTO, error) {
	p, err := s.queries.GetProjectByID(ctx, mustParseUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ColumnDTO{}, ErrProjectNotFound
		}
		return ColumnDTO{}, fmt.Errorf("get project: %w", err)
	}
	if err := s.checkOwnerAccess(actor, p); err != nil {
		return ColumnDTO{}, err
	}
	if p.Status == db.ProjectStatusArchived {
		return ColumnDTO{}, ErrProjectArchived
	}

	// Can only have one Done column
	if params.IsDoneColumn {
		if _, err := s.queries.GetDoneColumn(ctx, p.ID); err == nil {
			return ColumnDTO{}, ErrOneDoneColumnOnly
		}
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return ColumnDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	col, err := qtx.CreateBoardColumn(ctx, db.CreateBoardColumnParams{
		ProjectID:    p.ID,
		Name:         strings.TrimSpace(params.Name),
		Position:     params.Position,
		IsDoneColumn: params.IsDoneColumn,
		WipLimit:     toPgInt4(params.WipLimit),
	})
	if err != nil {
		return ColumnDTO{}, fmt.Errorf("create column: %w", err)
	}

	colID := uuid.UUID(col.ID.Bytes).String()
	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.column.create",
			EntityType: "board_column",
			EntityID:   colID,
			ProjectID:  p.ID,
			After:      map[string]any{"name": col.Name, "position": col.Position, "is_done_column": col.IsDoneColumn},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return ColumnDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return ColumnDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}
	return toColumnDTO(col), nil
}

// UpdateColumn updates a board column's name/position/wip_limit. Only PM owner.
func (s *Service) UpdateColumn(ctx context.Context, actor ActorInfo, columnID string, params UpdateColumnParams) (ColumnDTO, error) {
	col, err := s.queries.GetBoardColumnByID(ctx, mustParseUUID(columnID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ColumnDTO{}, errors.New("kolom tidak ditemukan")
		}
		return ColumnDTO{}, fmt.Errorf("get column: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, col.ProjectID)
	if err != nil {
		return ColumnDTO{}, fmt.Errorf("get project: %w", err)
	}
	if err := s.checkOwnerAccess(actor, p); err != nil {
		return ColumnDTO{}, err
	}
	if p.Status == db.ProjectStatusArchived {
		return ColumnDTO{}, ErrProjectArchived
	}

	before := map[string]any{"name": col.Name, "position": col.Position}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return ColumnDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	updated, err := qtx.UpdateBoardColumn(ctx, db.UpdateBoardColumnParams{
		ID:       col.ID,
		Name:     strings.TrimSpace(params.Name),
		Position: params.Position,
		WipLimit: toPgInt4(params.WipLimit),
	})
	if err != nil {
		return ColumnDTO{}, fmt.Errorf("update column: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.column.update",
			EntityType: "board_column",
			EntityID:   columnID,
			ProjectID:  p.ID,
			Before:     before,
			After:      map[string]any{"name": updated.Name, "position": updated.Position},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return ColumnDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return ColumnDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}
	return toColumnDTO(updated), nil
}

// DeleteColumn deletes a board column. Done column cannot be deleted.
func (s *Service) DeleteColumn(ctx context.Context, actor ActorInfo, columnID string) error {
	col, err := s.queries.GetBoardColumnByID(ctx, mustParseUUID(columnID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("kolom tidak ditemukan")
		}
		return fmt.Errorf("get column: %w", err)
	}

	if col.IsDoneColumn {
		return ErrColumnDoneProtected
	}

	p, err := s.queries.GetProjectByID(ctx, col.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}
	if err := s.checkOwnerAccess(actor, p); err != nil {
		return err
	}
	if p.Status == db.ProjectStatusArchived {
		return ErrProjectArchived
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		var err error
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	if err := qtx.DeleteBoardColumn(ctx, col.ID); err != nil {
		return fmt.Errorf("delete column: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    mustParseUUID(actor.ID),
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "project.column.delete",
			EntityType: "board_column",
			EntityID:   columnID,
			ProjectID:  p.ID,
			Before:     map[string]any{"name": col.Name, "position": col.Position},
			IP:         actor.IP,
			RequestID:  actor.RequestID,
		}); err != nil {
			return fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
}

// ─── Access Checks ────────────────────────────────────────────────────────────

// checkReadAccess enforces read visibility per RBAC matrix (SECURITY.md).
// Returns ErrProjectNotFound (not 403) when PM/TL can't see the project.
func (s *Service) checkReadAccess(ctx context.Context, actor ActorInfo, p db.Project) error {
	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		return nil // can see all
	case db.UserRoleProjectManager:
		if uuid.UUID(p.OwnerID.Bytes).String() != actor.ID {
			return ErrProjectNotFound // 404, not 403, to prevent enumeration
		}
	case db.UserRoleTeamLeader:
		isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
			ProjectID: p.ID,
			UserID:    mustParseUUID(actor.ID),
		})
		if err != nil || !isMember {
			return ErrProjectNotFound
		}
	default:
		return ErrForbidden
	}
	return nil
}

// checkOwnerAccess enforces write access: only PM owner.
func (s *Service) checkOwnerAccess(actor ActorInfo, p db.Project) error {
	if actor.Role != db.UserRoleProjectManager {
		return ErrForbidden
	}
	if uuid.UUID(p.OwnerID.Bytes).String() != actor.ID {
		return ErrProjectNotFound // 404 for cross-PM access
	}
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func mustParseUUID(s string) pgtype.UUID {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func toPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func toPgInt4(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}

func toProjectDTO(p db.Project) ProjectDTO {
	dto := ProjectDTO{
		ID:          uuid.UUID(p.ID.Bytes).String(),
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		OwnerID:     uuid.UUID(p.OwnerID.Bytes).String(),
		CreatedBy:   uuid.UUID(p.CreatedBy.Bytes).String(),
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
	if p.StartDate.Valid {
		s := p.StartDate.Time.Format("2006-01-02")
		dto.StartDate = &s
	}
	if p.EndDate.Valid {
		e := p.EndDate.Time.Format("2006-01-02")
		dto.EndDate = &e
	}
	return dto
}

func toColumnDTO(c db.BoardColumn) ColumnDTO {
	dto := ColumnDTO{
		ID:           uuid.UUID(c.ID.Bytes).String(),
		ProjectID:    uuid.UUID(c.ProjectID.Bytes).String(),
		Name:         c.Name,
		Position:     c.Position,
		IsDoneColumn: c.IsDoneColumn,
		CreatedAt:    c.CreatedAt.Time,
	}
	if c.WipLimit.Valid {
		dto.WipLimit = &c.WipLimit.Int32
	}
	return dto
}
