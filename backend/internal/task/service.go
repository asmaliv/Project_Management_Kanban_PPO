package task

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

// ActorInfo carries authenticated actor metadata.
type ActorInfo struct {
	ID        string
	Role      db.UserRole
	RequestID string
	IP        string
}

// TaskDTO represents a task returned to client.
type TaskDTO struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	ColumnID        string     `json:"column_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Priority        string     `json:"priority"`
	AssigneeID      *string    `json:"assignee_id"`
	AssigneeName    *string    `json:"assignee_name,omitempty"`
	ReporterID      string     `json:"reporter_id"`
	ReporterName    *string    `json:"reporter_name,omitempty"`
	DueDate         *string    `json:"due_date"`
	ProgressPercent int32      `json:"progress_percent"`
	Position        int32      `json:"position"`
	Labels          []string   `json:"labels"`
	Version         int32      `json:"version"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ColumnName      string     `json:"column_name,omitempty"`
	ProjectCode     string     `json:"project_code,omitempty"`
	ProjectName     string     `json:"project_name,omitempty"`
}

// TaskCommentDTO represents a comment returned to client.
type TaskCommentDTO struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	AuthorRole string    `json:"author_role"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateTaskParams holds input to create a task.
type CreateTaskParams struct {
	ProjectID       string
	ColumnID        string
	Title           string
	Description     string
	Priority        db.TaskPriority
	AssigneeID      *string
	DueDate         *time.Time
	ProgressPercent int32
	Labels          []string
}

// UpdateTaskParams holds input to update a task.
type UpdateTaskParams struct {
	Version         int32
	Title           string
	Description     string
	Priority        db.TaskPriority
	AssigneeID      *string
	DueDate         *time.Time
	ProgressPercent int32
	Labels          []string
}

// MoveTaskParams holds input to move a task.
type MoveTaskParams struct {
	ToColumnID string
	Position   int32
	Version    int32
}

// ListTasksFilter holds filters for GET /tasks.
type ListTasksFilter struct {
	ProjectID  *string
	AssigneeID *string
	ColumnID   *string
	IsOverdue  *bool
	Mine       *bool
	Limit      int32
	Offset     int32
}

// Service provides task business logic and enforcement of banking rules.
type Service struct {
	pool     *pgxpool.Pool
	queries  Store
	auditLog *auditlog.Service
	logger   *slog.Logger
}

// NewService instantiates a task Service.
func NewService(pool *pgxpool.Pool, queries Store, auditLog *auditlog.Service, logger *slog.Logger) *Service {
	return &Service{
		pool:     pool,
		queries:  queries,
		auditLog: auditLog,
		logger:   logger,
	}
}

// CreateTask creates a new task in a specified column.
func (s *Service) CreateTask(ctx context.Context, actor ActorInfo, projectID string, params CreateTaskParams) (TaskDTO, error) {
	projUUID := mustParseUUID(projectID)
	p, err := s.queries.GetProjectByID(ctx, projUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrProjectNotFound
		}
		return TaskDTO{}, fmt.Errorf("get project: %w", err)
	}

	if err := s.checkWriteAccess(ctx, actor, p); err != nil {
		return TaskDTO{}, err
	}

	if p.Status == db.ProjectStatusArchived {
		return TaskDTO{}, ErrProjectArchived
	}

	title := strings.TrimSpace(params.Title)
	if title == "" {
		return TaskDTO{}, errors.New("judul task wajib diisi")
	}

	colUUID := mustParseUUID(params.ColumnID)
	col, err := s.queries.GetBoardColumnByID(ctx, colUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrColumnNotFound
		}
		return TaskDTO{}, fmt.Errorf("get column: %w", err)
	}

	if uuid.UUID(col.ProjectID.Bytes).String() != projectID {
		return TaskDTO{}, ErrColumnNotFound
	}

	// Check WIP limit on column
	if col.WipLimit.Valid && col.WipLimit.Int32 > 0 {
		activeCount, err := s.queries.CountActiveTasksInColumn(ctx, col.ID)
		if err != nil {
			return TaskDTO{}, fmt.Errorf("check column wip: %w", err)
		}
		if activeCount >= col.WipLimit.Int32 {
			return TaskDTO{}, ErrWIPLimitExceeded
		}
	}

	var assigneeUUID pgtype.UUID
	if params.AssigneeID != nil && strings.TrimSpace(*params.AssigneeID) != "" {
		assigneeUUID = mustParseUUID(*params.AssigneeID)
		if err := s.validateAssignee(ctx, p, assigneeUUID); err != nil {
			return TaskDTO{}, err
		}
	}

	priority := params.Priority
	if !isValidPriority(priority) {
		priority = db.TaskPriorityMedium
	}

	maxPos, err := s.queries.MaxTaskPositionInColumn(ctx, col.ID)
	if err != nil {
		return TaskDTO{}, fmt.Errorf("get max position: %w", err)
	}
	newPos := maxPos + 1

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return TaskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	actorUUID := mustParseUUID(actor.ID)
	createdTask, err := qtx.CreateTask(ctx, db.CreateTaskParams{
		ProjectID:       p.ID,
		ColumnID:        col.ID,
		Title:           title,
		Description:     strings.TrimSpace(params.Description),
		Priority:        priority,
		AssigneeID:      assigneeUUID,
		ReporterID:      actorUUID,
		DueDate:         toPgDate(params.DueDate),
		ProgressPercent: params.ProgressPercent,
		Position:        newPos,
		Labels:          params.Labels,
	})
	if err != nil {
		return TaskDTO{}, fmt.Errorf("insert task: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "task.create",
			EntityType: "task",
			EntityID:   uuid.UUID(createdTask.ID.Bytes).String(),
			ProjectID:  p.ID,
			After: map[string]any{
				"title":            createdTask.Title,
				"column_id":        uuid.UUID(createdTask.ColumnID.Bytes).String(),
				"priority":         string(createdTask.Priority),
				"position":         createdTask.Position,
				"progress_percent": createdTask.ProgressPercent,
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return TaskDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return TaskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	detail, err := s.queries.GetTaskByIDWithDetails(ctx, createdTask.ID)
	if err != nil {
		return toTaskDTOFromModel(createdTask), nil
	}
	return toTaskDTOFromDetails(detail), nil
}

// GetTask returns a task by ID if the actor has permission.
func (s *Service) GetTask(ctx context.Context, actor ActorInfo, taskID string) (TaskDTO, error) {
	taskUUID := mustParseUUID(taskID)
	t, err := s.queries.GetTaskByIDWithDetails(ctx, taskUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrTaskNotFound
		}
		return TaskDTO{}, fmt.Errorf("get task: %w", err)
	}

	if err := s.checkReadAccessByProjectID(ctx, actor, t.ProjectID, t.ProjectOwnerID); err != nil {
		return TaskDTO{}, err
	}

	return toTaskDTOFromDetails(t), nil
}

// UpdateTask updates task fields with optimistic locking.
func (s *Service) UpdateTask(ctx context.Context, actor ActorInfo, taskID string, params UpdateTaskParams) (TaskDTO, error) {
	taskUUID := mustParseUUID(taskID)
	existing, err := s.queries.GetTaskByIDWithDetails(ctx, taskUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrTaskNotFound
		}
		return TaskDTO{}, fmt.Errorf("get task: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, existing.ProjectID)
	if err != nil {
		return TaskDTO{}, fmt.Errorf("get project: %w", err)
	}

	if p.Status == db.ProjectStatusArchived {
		return TaskDTO{}, ErrProjectArchived
	}

	if err := s.checkWriteAccess(ctx, actor, p); err != nil {
		return TaskDTO{}, err
	}

	title := strings.TrimSpace(params.Title)
	if title == "" {
		return TaskDTO{}, errors.New("judul task wajib diisi")
	}

	var assigneeUUID pgtype.UUID
	if params.AssigneeID != nil && strings.TrimSpace(*params.AssigneeID) != "" {
		assigneeUUID = mustParseUUID(*params.AssigneeID)
		if err := s.validateAssignee(ctx, p, assigneeUUID); err != nil {
			return TaskDTO{}, err
		}
	}

	priority := params.Priority
	if !isValidPriority(priority) {
		priority = existing.Priority
	}

	progress := params.ProgressPercent
	if progress < 0 || progress > 100 {
		return TaskDTO{}, errors.New("progress harus bernilai 0 - 100")
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return TaskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	actorUUID := mustParseUUID(actor.ID)
	updatedTask, err := qtx.UpdateTask(ctx, db.UpdateTaskParams{
		ID:              taskUUID,
		Version:         params.Version,
		Title:           title,
		Description:     strings.TrimSpace(params.Description),
		Priority:        priority,
		AssigneeID:      assigneeUUID,
		DueDate:         toPgDate(params.DueDate),
		ProgressPercent: progress,
		Labels:          params.Labels,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrVersionConflict
		}
		return TaskDTO{}, fmt.Errorf("update task: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "task.update",
			EntityType: "task",
			EntityID:   uuid.UUID(updatedTask.ID.Bytes).String(),
			ProjectID:  p.ID,
			Before: map[string]any{
				"title":            existing.Title,
				"priority":         string(existing.Priority),
				"progress_percent": existing.ProgressPercent,
				"version":          existing.Version,
			},
			After: map[string]any{
				"title":            updatedTask.Title,
				"priority":         string(updatedTask.Priority),
				"progress_percent": updatedTask.ProgressPercent,
				"version":          updatedTask.Version,
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return TaskDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return TaskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	details, err := s.queries.GetTaskByIDWithDetails(ctx, updatedTask.ID)
	if err != nil {
		return toTaskDTOFromModel(updatedTask), nil
	}
	return toTaskDTOFromDetails(details), nil
}

// MoveTask moves a task to a different position and/or column with maker-checker and WIP limit enforcement.
func (s *Service) MoveTask(ctx context.Context, actor ActorInfo, taskID string, params MoveTaskParams) (TaskDTO, error) {
	taskUUID := mustParseUUID(taskID)
	existing, err := s.queries.GetTaskByIDWithDetails(ctx, taskUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrTaskNotFound
		}
		return TaskDTO{}, fmt.Errorf("get task: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, existing.ProjectID)
	if err != nil {
		return TaskDTO{}, fmt.Errorf("get project: %w", err)
	}

	if p.Status == db.ProjectStatusArchived {
		return TaskDTO{}, ErrProjectArchived
	}

	if err := s.checkWriteAccess(ctx, actor, p); err != nil {
		return TaskDTO{}, err
	}

	targetColUUID := mustParseUUID(params.ToColumnID)
	targetCol, err := s.queries.GetBoardColumnByID(ctx, targetColUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrColumnNotFound
		}
		return TaskDTO{}, fmt.Errorf("get target column: %w", err)
	}

	if uuid.UUID(targetCol.ProjectID.Bytes).String() != uuid.UUID(p.ID.Bytes).String() {
		return TaskDTO{}, ErrColumnNotFound
	}

	// Maker-checker rule: Moving to or from Done column requires Project Manager owner
	isMovingFromDone := existing.IsDoneColumn
	isMovingToDone := targetCol.IsDoneColumn
	if isMovingFromDone || isMovingToDone {
		if actor.Role != db.UserRoleProjectManager || uuid.UUID(p.OwnerID.Bytes).String() != actor.ID {
			return TaskDTO{}, ErrMakerCheckerDone
		}
	}

	isSameColumn := existing.ColumnID == targetCol.ID
	newPos := params.Position
	if newPos < 1 {
		newPos = 1
	}

	// WIP limit check (only if moving to a different column)
	if !isSameColumn && targetCol.WipLimit.Valid && targetCol.WipLimit.Int32 > 0 {
		activeCount, err := s.queries.CountActiveTasksInColumn(ctx, targetCol.ID)
		if err != nil {
			return TaskDTO{}, fmt.Errorf("check wip limit: %w", err)
		}
		if activeCount >= targetCol.WipLimit.Int32 {
			return TaskDTO{}, ErrWIPLimitExceeded
		}
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return TaskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	oldPos := existing.Position

	// Reorder positions atomically
	if isSameColumn {
		if newPos > oldPos {
			// Shifting down: items from oldPos+1 to newPos decrease by 1
			if err := qtx.ShiftTasksBetweenPositionsDown(ctx, db.ShiftTasksBetweenPositionsDownParams{
				ColumnID:   existing.ColumnID,
				Position:   oldPos,
				Position_2: newPos,
			}); err != nil {
				return TaskDTO{}, fmt.Errorf("shift positions: %w", err)
			}
		} else if newPos < oldPos {
			// Shifting up: items from newPos to oldPos-1 increase by 1
			if err := qtx.ShiftTasksBetweenPositionsUp(ctx, db.ShiftTasksBetweenPositionsUpParams{
				ColumnID:   existing.ColumnID,
				Position:   newPos,
				Position_2: oldPos,
			}); err != nil {
				return TaskDTO{}, fmt.Errorf("shift positions: %w", err)
			}
		}
	} else {
		// Moving to different column:
		// 1. In old column: shift items after oldPos down by 1
		if err := qtx.ShiftTasksDownInColumn(ctx, db.ShiftTasksDownInColumnParams{
			ColumnID: existing.ColumnID,
			Position: oldPos,
		}); err != nil {
			return TaskDTO{}, fmt.Errorf("shift old column: %w", err)
		}
		// 2. In new column: shift items at or after newPos up by 1
		if err := qtx.ShiftTasksUpInColumn(ctx, db.ShiftTasksUpInColumnParams{
			ColumnID: targetCol.ID,
			Position: newPos,
		}); err != nil {
			return TaskDTO{}, fmt.Errorf("shift new column: %w", err)
		}
	}

	var completedAt pgtype.Timestamptz
	if isMovingToDone {
		completedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	} else if isMovingFromDone {
		completedAt = pgtype.Timestamptz{Valid: false}
	} else {
		completedAt = existing.CompletedAt
	}

	updatedTask, err := qtx.UpdateTaskPositionAndColumn(ctx, db.UpdateTaskPositionAndColumnParams{
		ID:          taskUUID,
		ColumnID:    targetCol.ID,
		Position:    newPos,
		CompletedAt: completedAt,
		Version:     params.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskDTO{}, ErrVersionConflict
		}
		return TaskDTO{}, fmt.Errorf("update task position: %w", err)
	}

	actorUUID := mustParseUUID(actor.ID)
	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "task.move",
			EntityType: "task",
			EntityID:   uuid.UUID(updatedTask.ID.Bytes).String(),
			ProjectID:  p.ID,
			Before: map[string]any{
				"column_id": uuid.UUID(existing.ColumnID.Bytes).String(),
				"position":  existing.Position,
				"version":   existing.Version,
			},
			After: map[string]any{
				"column_id": uuid.UUID(updatedTask.ColumnID.Bytes).String(),
				"position":  updatedTask.Position,
				"version":   updatedTask.Version,
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return TaskDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return TaskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	details, err := s.queries.GetTaskByIDWithDetails(ctx, updatedTask.ID)
	if err != nil {
		return toTaskDTOFromModel(updatedTask), nil
	}
	return toTaskDTOFromDetails(details), nil
}

// DeleteTask soft-deletes a task (only PM owner).
func (s *Service) DeleteTask(ctx context.Context, actor ActorInfo, taskID string) error {
	taskUUID := mustParseUUID(taskID)
	existing, err := s.queries.GetTaskByIDWithDetails(ctx, taskUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("get task: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, existing.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	if p.Status == db.ProjectStatusArchived {
		return ErrProjectArchived
	}

	// Soft delete task: PM (Own) only
	if actor.Role != db.UserRoleProjectManager {
		return ErrForbidden
	}
	if uuid.UUID(p.OwnerID.Bytes).String() != actor.ID {
		return ErrTaskNotFound
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	deleted, err := qtx.SoftDeleteTask(ctx, taskUUID)
	if err != nil {
		return fmt.Errorf("soft delete task: %w", err)
	}

	// Shift items after deleted task down by 1 in same column
	if err := qtx.ShiftTasksDownInColumn(ctx, db.ShiftTasksDownInColumnParams{
		ColumnID: existing.ColumnID,
		Position: existing.Position,
	}); err != nil {
		return fmt.Errorf("shift tasks after delete: %w", err)
	}

	actorUUID := mustParseUUID(actor.ID)
	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "task.delete",
			EntityType: "task",
			EntityID:   uuid.UUID(deleted.ID.Bytes).String(),
			ProjectID:  p.ID,
			Before: map[string]any{
				"title":     existing.Title,
				"column_id": uuid.UUID(existing.ColumnID.Bytes).String(),
				"position":  existing.Position,
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
	}

	return nil
}

// ListTasks returns a paginated and filtered list of tasks visible to actor across projects.
func (s *Service) ListTasks(ctx context.Context, actor ActorInfo, filter ListTasksFilter) ([]TaskDTO, int64, error) {
	allowedProjects, err := s.getAllowedProjectIDs(ctx, actor, filter.ProjectID)
	if err != nil {
		return nil, 0, err
	}

	// If no projects allowed (e.g. TL with 0 projects or non-owner PM), return empty slice
	if allowedProjects != nil && len(allowedProjects) == 0 {
		return []TaskDTO{}, 0, nil
	}

	var assigneeUUID pgtype.UUID
	if filter.Mine != nil && *filter.Mine {
		assigneeUUID = mustParseUUID(actor.ID)
	} else if filter.AssigneeID != nil && *filter.AssigneeID != "" {
		assigneeUUID = mustParseUUID(*filter.AssigneeID)
	}

	var colUUID pgtype.UUID
	if filter.ColumnID != nil && *filter.ColumnID != "" {
		colUUID = mustParseUUID(*filter.ColumnID)
	}

	var isOverdue pgtype.Bool
	if filter.IsOverdue != nil {
		isOverdue = pgtype.Bool{Bool: *filter.IsOverdue, Valid: true}
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.queries.ListTasksByProjectsFiltered(ctx, db.ListTasksByProjectsFilteredParams{
		ProjectIds: allowedProjects,
		AssigneeID: assigneeUUID,
		ColumnID:   colUUID,
		IsOverdue:  isOverdue,
		LimitVal:   limit,
		OffsetVal:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}

	total, err := s.queries.CountTasksByProjectsFiltered(ctx, db.CountTasksByProjectsFilteredParams{
		ProjectIds: allowedProjects,
		AssigneeID: assigneeUUID,
		ColumnID:   colUUID,
		IsOverdue:  isOverdue,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	dtos := make([]TaskDTO, len(rows))
	for i, r := range rows {
		dtos[i] = toTaskDTOFromFilteredRow(r)
	}

	return dtos, total, nil
}

// ListBoardTasks returns all active tasks for a specific project board.
func (s *Service) ListBoardTasks(ctx context.Context, actor ActorInfo, projectID string) ([]TaskDTO, error) {
	projUUID := mustParseUUID(projectID)
	p, err := s.queries.GetProjectByID(ctx, projUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}

	if err := s.checkReadAccessByProjectID(ctx, actor, p.ID, p.OwnerID); err != nil {
		return nil, err
	}

	tasks, err := s.queries.ListTasksByProject(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("list tasks by project: %w", err)
	}

	dtos := make([]TaskDTO, len(tasks))
	for i, t := range tasks {
		dtos[i] = toTaskDTOFromProjectRow(t, p.Code, p.Name)
	}

	return dtos, nil
}

// CreateComment adds a comment to a task.
func (s *Service) CreateComment(ctx context.Context, actor ActorInfo, taskID string, body string) (TaskCommentDTO, error) {
	taskUUID := mustParseUUID(taskID)
	t, err := s.queries.GetTaskByIDWithDetails(ctx, taskUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskCommentDTO{}, ErrTaskNotFound
		}
		return TaskCommentDTO{}, fmt.Errorf("get task: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, t.ProjectID)
	if err != nil {
		return TaskCommentDTO{}, fmt.Errorf("get project: %w", err)
	}

	if p.Status == db.ProjectStatusArchived {
		return TaskCommentDTO{}, ErrProjectArchived
	}

	if err := s.checkWriteAccess(ctx, actor, p); err != nil {
		return TaskCommentDTO{}, err
	}

	cleanBody := strings.TrimSpace(body)
	if cleanBody == "" {
		return TaskCommentDTO{}, ErrCommentBodyEmpty
	}
	if len(cleanBody) > 5000 {
		return TaskCommentDTO{}, ErrCommentBodyTooLong
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return TaskCommentDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	actorUUID := mustParseUUID(actor.ID)
	comment, err := qtx.CreateTaskComment(ctx, db.CreateTaskCommentParams{
		TaskID:   t.ID,
		AuthorID: actorUUID,
		Body:     cleanBody,
	})
	if err != nil {
		return TaskCommentDTO{}, fmt.Errorf("create comment: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "task_comment.create",
			EntityType: "task_comment",
			EntityID:   uuid.UUID(comment.ID.Bytes).String(),
			ProjectID:  p.ID,
			After: map[string]any{
				"task_id": uuid.UUID(t.ID.Bytes).String(),
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return TaskCommentDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return TaskCommentDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	user, err := s.queries.GetUserByID(ctx, actorUUID)
	var authorName string
	var authorRole string
	if err == nil {
		authorName = user.FullName
		authorRole = string(user.Role)
	}

	return TaskCommentDTO{
		ID:         uuid.UUID(comment.ID.Bytes).String(),
		TaskID:     uuid.UUID(comment.TaskID.Bytes).String(),
		AuthorID:   uuid.UUID(comment.AuthorID.Bytes).String(),
		AuthorName: authorName,
		AuthorRole: authorRole,
		Body:       comment.Body,
		CreatedAt:  comment.CreatedAt.Time,
	}, nil
}

// ListComments retrieves all comments for a task.
func (s *Service) ListComments(ctx context.Context, actor ActorInfo, taskID string) ([]TaskCommentDTO, error) {
	taskUUID := mustParseUUID(taskID)
	t, err := s.queries.GetTaskByIDWithDetails(ctx, taskUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}

	if err := s.checkReadAccessByProjectID(ctx, actor, t.ProjectID, t.ProjectOwnerID); err != nil {
		return nil, err
	}

	comments, err := s.queries.ListTaskComments(ctx, t.ID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}

	dtos := make([]TaskCommentDTO, len(comments))
	for i, c := range comments {
		dtos[i] = TaskCommentDTO{
			ID:         uuid.UUID(c.ID.Bytes).String(),
			TaskID:     uuid.UUID(c.TaskID.Bytes).String(),
			AuthorID:   uuid.UUID(c.AuthorID.Bytes).String(),
			AuthorName: c.AuthorName,
			AuthorRole: string(c.AuthorRole),
			Body:       c.Body,
			CreatedAt:  c.CreatedAt.Time,
		}
	}

	return dtos, nil
}

// ─── Authorization Helpers ───────────────────────────────────────────────────

func (s *Service) checkReadAccessByProjectID(ctx context.Context, actor ActorInfo, projectID pgtype.UUID, ownerID pgtype.UUID) error {
	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		return nil
	case db.UserRoleProjectManager:
		if uuid.UUID(ownerID.Bytes).String() != actor.ID {
			return ErrTaskNotFound
		}
		return nil
	case db.UserRoleTeamLeader:
		isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
			ProjectID: projectID,
			UserID:    mustParseUUID(actor.ID),
		})
		if err != nil || !isMember {
			return ErrTaskNotFound
		}
		return nil
	default:
		return ErrForbidden
	}
}

func (s *Service) checkWriteAccess(ctx context.Context, actor ActorInfo, p db.Project) error {
	switch actor.Role {
	case db.UserRoleProjectManager:
		if uuid.UUID(p.OwnerID.Bytes).String() != actor.ID {
			return ErrProjectNotFound
		}
		return nil
	case db.UserRoleTeamLeader:
		isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
			ProjectID: p.ID,
			UserID:    mustParseUUID(actor.ID),
		})
		if err != nil || !isMember {
			return ErrProjectNotFound
		}
		return nil
	default:
		return ErrForbidden
	}
}

func (s *Service) validateAssignee(ctx context.Context, p db.Project, assigneeID pgtype.UUID) error {
	// Owner is always a valid assignee
	if assigneeID == p.OwnerID {
		return nil
	}
	// Or must be a project member
	isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
		ProjectID: p.ID,
		UserID:    assigneeID,
	})
	if err != nil || !isMember {
		return ErrAssigneeNotMember
	}
	return nil
}

func (s *Service) getAllowedProjectIDs(ctx context.Context, actor ActorInfo, specificProjectID *string) ([]pgtype.UUID, error) {
	if specificProjectID != nil && strings.TrimSpace(*specificProjectID) != "" {
		projUUID := mustParseUUID(*specificProjectID)
		p, err := s.queries.GetProjectByID(ctx, projUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return []pgtype.UUID{}, nil
			}
			return nil, err
		}
		if err := s.checkReadAccessByProjectID(ctx, actor, p.ID, p.OwnerID); err != nil {
			return []pgtype.UUID{}, nil
		}
		return []pgtype.UUID{projUUID}, nil
	}

	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		// nil allows all active projects
		return nil, nil
	case db.UserRoleProjectManager:
		projects, err := s.queries.ListProjectsByOwner(ctx, mustParseUUID(actor.ID))
		if err != nil {
			return nil, err
		}
		ids := make([]pgtype.UUID, len(projects))
		for i, prj := range projects {
			ids[i] = prj.ID
		}
		return ids, nil
	case db.UserRoleTeamLeader:
		projects, err := s.queries.ListProjectsByMember(ctx, mustParseUUID(actor.ID))
		if err != nil {
			return nil, err
		}
		ids := make([]pgtype.UUID, len(projects))
		for i, prj := range projects {
			ids[i] = prj.ID
		}
		return ids, nil
	default:
		return []pgtype.UUID{}, nil
	}
}

// ─── DTO Helpers ─────────────────────────────────────────────────────────────

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

func toTaskDTOFromModel(t db.Task) TaskDTO {
	var assigneeID *string
	if t.AssigneeID.Valid {
		s := uuid.UUID(t.AssigneeID.Bytes).String()
		assigneeID = &s
	}

	var dueDateStr *string
	if t.DueDate.Valid {
		s := t.DueDate.Time.Format("2006-01-02")
		dueDateStr = &s
	}

	var completedAt *time.Time
	if t.CompletedAt.Valid {
		completedAt = &t.CompletedAt.Time
	}

	return TaskDTO{
		ID:              uuid.UUID(t.ID.Bytes).String(),
		ProjectID:       uuid.UUID(t.ProjectID.Bytes).String(),
		ColumnID:        uuid.UUID(t.ColumnID.Bytes).String(),
		Title:           t.Title,
		Description:     t.Description,
		Priority:        string(t.Priority),
		AssigneeID:      assigneeID,
		ReporterID:      uuid.UUID(t.ReporterID.Bytes).String(),
		DueDate:         dueDateStr,
		ProgressPercent: t.ProgressPercent,
		Position:        t.Position,
		Labels:          t.Labels,
		Version:         t.Version,
		CompletedAt:     completedAt,
		CreatedAt:       t.CreatedAt.Time,
		UpdatedAt:       t.UpdatedAt.Time,
	}
}

func toTaskDTOFromDetails(d db.GetTaskByIDWithDetailsRow) TaskDTO {
	var assigneeID *string
	var assigneeName *string
	if d.AssigneeID.Valid {
		s := uuid.UUID(d.AssigneeID.Bytes).String()
		assigneeID = &s
		if d.AssigneeName.Valid {
			assigneeName = &d.AssigneeName.String
		}
	}

	var dueDateStr *string
	if d.DueDate.Valid {
		s := d.DueDate.Time.Format("2006-01-02")
		dueDateStr = &s
	}

	var completedAt *time.Time
	if d.CompletedAt.Valid {
		completedAt = &d.CompletedAt.Time
	}

	return TaskDTO{
		ID:              uuid.UUID(d.ID.Bytes).String(),
		ProjectID:       uuid.UUID(d.ProjectID.Bytes).String(),
		ColumnID:        uuid.UUID(d.ColumnID.Bytes).String(),
		Title:           d.Title,
		Description:     d.Description,
		Priority:        string(d.Priority),
		AssigneeID:      assigneeID,
		AssigneeName:    assigneeName,
		ReporterID:      uuid.UUID(d.ReporterID.Bytes).String(),
		ReporterName:    &d.ReporterName,
		DueDate:         dueDateStr,
		ProgressPercent: d.ProgressPercent,
		Position:        d.Position,
		Labels:          d.Labels,
		Version:         d.Version,
		CompletedAt:     completedAt,
		CreatedAt:       d.CreatedAt.Time,
		UpdatedAt:       d.UpdatedAt.Time,
		ColumnName:      d.ColumnName,
		ProjectCode:     d.ProjectCode,
		ProjectName:     d.ProjectName,
	}
}

func toTaskDTOFromProjectRow(d db.ListTasksByProjectRow, projectCode, projectName string) TaskDTO {
	var assigneeID *string
	var assigneeName *string
	if d.AssigneeID.Valid {
		s := uuid.UUID(d.AssigneeID.Bytes).String()
		assigneeID = &s
		if d.AssigneeName.Valid {
			assigneeName = &d.AssigneeName.String
		}
	}

	var dueDateStr *string
	if d.DueDate.Valid {
		s := d.DueDate.Time.Format("2006-01-02")
		dueDateStr = &s
	}

	var completedAt *time.Time
	if d.CompletedAt.Valid {
		completedAt = &d.CompletedAt.Time
	}

	return TaskDTO{
		ID:              uuid.UUID(d.ID.Bytes).String(),
		ProjectID:       uuid.UUID(d.ProjectID.Bytes).String(),
		ColumnID:        uuid.UUID(d.ColumnID.Bytes).String(),
		Title:           d.Title,
		Description:     d.Description,
		Priority:        string(d.Priority),
		AssigneeID:      assigneeID,
		AssigneeName:    assigneeName,
		ReporterID:      uuid.UUID(d.ReporterID.Bytes).String(),
		ReporterName:    &d.ReporterName,
		DueDate:         dueDateStr,
		ProgressPercent: d.ProgressPercent,
		Position:        d.Position,
		Labels:          d.Labels,
		Version:         d.Version,
		CompletedAt:     completedAt,
		CreatedAt:       d.CreatedAt.Time,
		UpdatedAt:       d.UpdatedAt.Time,
		ProjectCode:     projectCode,
		ProjectName:     projectName,
	}
}

func toTaskDTOFromFilteredRow(d db.ListTasksByProjectsFilteredRow) TaskDTO {
	var assigneeID *string
	var assigneeName *string
	if d.AssigneeID.Valid {
		s := uuid.UUID(d.AssigneeID.Bytes).String()
		assigneeID = &s
		if d.AssigneeName.Valid {
			assigneeName = &d.AssigneeName.String
		}
	}

	var dueDateStr *string
	if d.DueDate.Valid {
		s := d.DueDate.Time.Format("2006-01-02")
		dueDateStr = &s
	}

	var completedAt *time.Time
	if d.CompletedAt.Valid {
		completedAt = &d.CompletedAt.Time
	}

	return TaskDTO{
		ID:              uuid.UUID(d.ID.Bytes).String(),
		ProjectID:       uuid.UUID(d.ProjectID.Bytes).String(),
		ColumnID:        uuid.UUID(d.ColumnID.Bytes).String(),
		Title:           d.Title,
		Description:     d.Description,
		Priority:        string(d.Priority),
		AssigneeID:      assigneeID,
		AssigneeName:    assigneeName,
		ReporterID:      uuid.UUID(d.ReporterID.Bytes).String(),
		ReporterName:    &d.ReporterName,
		DueDate:         dueDateStr,
		ProgressPercent: d.ProgressPercent,
		Position:        d.Position,
		Labels:          d.Labels,
		Version:         d.Version,
		CompletedAt:     completedAt,
		CreatedAt:       d.CreatedAt.Time,
		UpdatedAt:       d.UpdatedAt.Time,
		ColumnName:      d.ColumnName,
		ProjectCode:     d.ProjectCode,
		ProjectName:     d.ProjectName,
	}
}

func isValidPriority(p db.TaskPriority) bool {
	switch p {
	case db.TaskPriorityLow, db.TaskPriorityMedium, db.TaskPriorityHigh, db.TaskPriorityCritical:
		return true
	default:
		return false
	}
}
