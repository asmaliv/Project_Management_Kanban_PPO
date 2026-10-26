package risk

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
	ErrRiskNotFound                    = errors.New("risiko tidak ditemukan")
	ErrForbidden                       = errors.New("anda tidak memiliki izin untuk melakukan aksi ini")
	ErrProjectNotFound                 = errors.New("proyek tidak ditemukan")
	ErrTaskNotFound                    = errors.New("task tidak ditemukan")
	ErrProjectArchived                 = errors.New("proyek sudah diarsipkan, tidak dapat diubah")
	ErrInvalidInput                    = errors.New("input data risiko tidak valid")
	ErrHighCriticalMitigationRequired  = errors.New("Risiko High/Critical tidak dapat ditutup tanpa rencana mitigasi terisi.")
	ErrOnlyRiskManagementAssess        = errors.New("hanya Risk Management yang dapat menilai risiko")
	ErrOnlyRiskManagementClose         = errors.New("hanya Risk Management yang dapat menutup risiko")
	ErrOnlyPMUpdateMitigation          = errors.New("Project Manager hanya dapat memperbarui rencana mitigasi pada proyek miliknya")
)

// ActorInfo carries authenticated actor metadata.
type ActorInfo struct {
	ID        string
	Role      db.UserRole
	RequestID string
	IP        string
}

// RiskDTO represents a risk returned to client.
type RiskDTO struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	ProjectID      string     `json:"project_id"`
	ProjectName    string     `json:"project_name,omitempty"`
	ProjectCode    string     `json:"project_code,omitempty"`
	TaskID         *string    `json:"task_id"`
	TaskTitle      *string    `json:"task_title,omitempty"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	Likelihood     *int32     `json:"likelihood"`
	Impact         *int32     `json:"impact"`
	Score          *int32     `json:"score"`
	Level          *string    `json:"level"`
	Status         string     `json:"status"`
	MitigationPlan string     `json:"mitigation_plan"`
	OwnerID        *string    `json:"owner_id"`
	OwnerName      *string    `json:"owner_name,omitempty"`
	RaisedBy       string     `json:"raised_by"`
	RaisedByName   string     `json:"raised_by_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClosedAt       *time.Time `json:"closed_at"`
}

// CreateRiskParams holds input to raise a new risk.
type CreateRiskParams struct {
	ProjectID   string  `json:"project_id"`
	TaskID      *string `json:"task_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	OwnerID     *string `json:"owner_id"`
}

// AssessRiskParams holds input for assessing likelihood & impact.
type AssessRiskParams struct {
	Likelihood     int32   `json:"likelihood"`
	Impact         int32   `json:"impact"`
	MitigationPlan *string `json:"mitigation_plan"`
}

// UpdateRiskParams holds fields that can be updated.
type UpdateRiskParams struct {
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	Category       *string `json:"category"`
	TaskID         *string `json:"task_id"`
	OwnerID        *string `json:"owner_id"`
	Status         *string `json:"status"`
	MitigationPlan *string `json:"mitigation_plan"`
}

// ListRisksParams holds filter parameters.
type ListRisksParams struct {
	ProjectID *string `json:"project_id"`
	TaskID    *string `json:"task_id"`
	Level     *string `json:"level"`
	Status    *string `json:"status"`
	Category  *string `json:"category"`
	Search    *string `json:"search"`
	Limit     int32   `json:"limit"`
	Offset    int32   `json:"offset"`
}

// HeatmapCellDTO represents a single cell in the 5x5 heatmap.
type HeatmapCellDTO struct {
	Likelihood int32  `json:"likelihood"`
	Impact     int32  `json:"impact"`
	Score      int32  `json:"score"`
	Level      string `json:"level"`
	Count      int64  `json:"count"`
}

// RiskSummaryDTO represents the high-level stats and heatmap data.
type RiskSummaryDTO struct {
	TotalRisks        int64            `json:"total_risks"`
	OpenRisks         int64            `json:"open_risks"`
	HighCriticalRisks int64            `json:"high_critical_risks"`
	UnmitigatedRisks  int64            `json:"unmitigated_risks"`
	ByLevel           map[string]int64 `json:"by_level"`
	Heatmap           []HeatmapCellDTO `json:"heatmap"`
}

// TaskRiskCountDTO represents the count of risks associated with a task.
type TaskRiskCountDTO struct {
	TaskID            string `json:"task_id"`
	TotalRisks        int64  `json:"total_risks"`
	HighCriticalRisks int64  `json:"high_critical_risks"`
}

// Service provides risk register business logic and enforcement of banking rules.
type Service struct {
	pool     *pgxpool.Pool
	queries  Store
	auditLog *auditlog.Service
	logger   *slog.Logger
}

// NewService instantiates a risk Service.
func NewService(pool *pgxpool.Pool, queries Store, auditLog *auditlog.Service, logger *slog.Logger) *Service {
	return &Service{
		pool:     pool,
		queries:  queries,
		auditLog: auditLog,
		logger:   logger,
	}
}

// CalculateRiskScoreAndLevel computes score = likelihood * impact and the corresponding level.
// Ambang batas: 1–4 Low, 5–9 Medium, 10–15 High, 16–25 Critical.
func CalculateRiskScoreAndLevel(likelihood, impact int32) (int32, db.RiskLevel, error) {
	if likelihood < 1 || likelihood > 5 {
		return 0, "", fmt.Errorf("%w: likelihood harus bernilai antara 1 dan 5", ErrInvalidInput)
	}
	if impact < 1 || impact > 5 {
		return 0, "", fmt.Errorf("%w: impact harus bernilai antara 1 dan 5", ErrInvalidInput)
	}
	score := likelihood * impact
	var level db.RiskLevel
	switch {
	case score <= 4:
		level = db.RiskLevelLow
	case score <= 9:
		level = db.RiskLevelMedium
	case score <= 15:
		level = db.RiskLevelHigh
	default:
		level = db.RiskLevelCritical
	}
	return score, level, nil
}

// CreateRisk raises a new risk for a project/task.
// Permission: PM (own project), TL (project member), RM (any project). Admin & Audit denied.
func (s *Service) CreateRisk(ctx context.Context, actor ActorInfo, params CreateRiskParams) (RiskDTO, error) {
	projUUID, err := parseUUID(params.ProjectID)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("%w: format project_id tidak valid", ErrInvalidInput)
	}

	p, err := s.queries.GetProjectByID(ctx, projUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RiskDTO{}, ErrProjectNotFound
		}
		return RiskDTO{}, fmt.Errorf("get project: %w", err)
	}

	if p.Status == db.ProjectStatusArchived {
		return RiskDTO{}, ErrProjectArchived
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("invalid actor id: %w", err)
	}

	switch actor.Role {
	case db.UserRoleProjectManager:
		if p.OwnerID != actorUUID {
			return RiskDTO{}, ErrForbidden
		}
	case db.UserRoleTeamLeader:
		isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
			ProjectID: p.ID,
			UserID:    actorUUID,
		})
		if err != nil {
			return RiskDTO{}, fmt.Errorf("check project member: %w", err)
		}
		if !isMember {
			return RiskDTO{}, ErrForbidden
		}
	case db.UserRoleRiskManagement:
		// Risk Management boleh mengajukan risiko di semua proyek aktif
	default:
		// Admin dan Audit tidak boleh mengajukan risiko
		return RiskDTO{}, ErrForbidden
	}

	title := strings.TrimSpace(params.Title)
	if title == "" {
		return RiskDTO{}, fmt.Errorf("%w: judul risiko wajib diisi", ErrInvalidInput)
	}

	cat, err := parseRiskCategory(params.Category)
	if err != nil {
		return RiskDTO{}, err
	}

	var taskUUID pgtype.UUID
	if params.TaskID != nil && strings.TrimSpace(*params.TaskID) != "" {
		tu, err := parseUUID(*params.TaskID)
		if err != nil {
			return RiskDTO{}, fmt.Errorf("%w: format task_id tidak valid", ErrInvalidInput)
		}
		taskObj, err := s.queries.GetTaskByID(ctx, tu)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RiskDTO{}, ErrTaskNotFound
			}
			return RiskDTO{}, fmt.Errorf("get task: %w", err)
		}
		if taskObj.ProjectID != p.ID {
			return RiskDTO{}, fmt.Errorf("%w: task tidak berada dalam proyek ini", ErrInvalidInput)
		}
		taskUUID = tu
	}

	var ownerUUID pgtype.UUID
	if params.OwnerID != nil && strings.TrimSpace(*params.OwnerID) != "" {
		ou, err := parseUUID(*params.OwnerID)
		if err != nil {
			return RiskDTO{}, fmt.Errorf("%w: format owner_id tidak valid", ErrInvalidInput)
		}
		ownerUUID = ou
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return RiskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	code, err := qtx.GetNextRiskCode(ctx)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("next risk code: %w", err)
	}

	createdRisk, err := qtx.CreateRisk(ctx, db.CreateRiskParams{
		Code:        code,
		ProjectID:   p.ID,
		TaskID:      taskUUID,
		Title:       title,
		Description: strings.TrimSpace(params.Description),
		Category:    cat,
		Likelihood:  pgtype.Int4{Valid: false},
		Impact:      pgtype.Int4{Valid: false},
		Level:       db.NullRiskLevel{Valid: false},
		Status:      db.RiskStatusIdentified,
		OwnerID:     ownerUUID,
		RaisedBy:    actorUUID,
	})
	if err != nil {
		return RiskDTO{}, fmt.Errorf("insert risk: %w", err)
	}

	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "risk.create",
			EntityType: "risk",
			EntityID:   uuid.UUID(createdRisk.ID.Bytes).String(),
			ProjectID:  p.ID,
			After: map[string]any{
				"code":        createdRisk.Code,
				"title":       createdRisk.Title,
				"category":    string(createdRisk.Category),
				"status":      string(createdRisk.Status),
				"project_id":  uuid.UUID(p.ID.Bytes).String(),
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return RiskDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return RiskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	detail, err := s.queries.GetRiskByIDWithDetails(ctx, createdRisk.ID)
	if err != nil {
		return mapRowToDTO(detail), nil
	}

	return mapRowToDTO(detail), nil
}

// GetRiskByID retrieves a risk by ID with detail fields.
// RBAC: Admin, Audit, RM (all), PM (own project), TL (project member).
func (s *Service) GetRiskByID(ctx context.Context, actor ActorInfo, id string) (RiskDTO, error) {
	rUUID, err := parseUUID(id)
	if err != nil {
		return RiskDTO{}, ErrRiskNotFound
	}

	detail, err := s.queries.GetRiskByIDWithDetails(ctx, rUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RiskDTO{}, ErrRiskNotFound
		}
		return RiskDTO{}, fmt.Errorf("get risk: %w", err)
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return RiskDTO{}, ErrForbidden
	}

	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		// Read-all
	case db.UserRoleProjectManager:
		if detail.ProjectOwnerID != actorUUID {
			return RiskDTO{}, ErrRiskNotFound
		}
	case db.UserRoleTeamLeader:
		isMember, err := s.queries.IsProjectMember(ctx, db.IsProjectMemberParams{
			ProjectID: detail.ProjectID,
			UserID:    actorUUID,
		})
		if err != nil || !isMember {
			return RiskDTO{}, ErrRiskNotFound
		}
	default:
		return RiskDTO{}, ErrForbidden
	}

	return mapRowToDTO(detail), nil
}

// AssessRisk updates likelihood and impact, calculating score and level server-side.
// Permission: Hanya Risk Management.
func (s *Service) AssessRisk(ctx context.Context, actor ActorInfo, id string, params AssessRiskParams) (RiskDTO, error) {
	if actor.Role != db.UserRoleRiskManagement {
		return RiskDTO{}, ErrOnlyRiskManagementAssess
	}

	rUUID, err := parseUUID(id)
	if err != nil {
		return RiskDTO{}, ErrRiskNotFound
	}

	existing, err := s.queries.GetRiskByID(ctx, rUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RiskDTO{}, ErrRiskNotFound
		}
		return RiskDTO{}, fmt.Errorf("get risk: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, existing.ProjectID)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("get project: %w", err)
	}
	if p.Status == db.ProjectStatusArchived {
		return RiskDTO{}, ErrProjectArchived
	}

	score, level, err := CalculateRiskScoreAndLevel(params.Likelihood, params.Impact)
	if err != nil {
		return RiskDTO{}, err
	}

	var mitPlan *string
	if params.MitigationPlan != nil {
		trimmed := strings.TrimSpace(*params.MitigationPlan)
		mitPlan = &trimmed
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return RiskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	actorUUID := mustParseUUID(actor.ID)
	var mitPlanStr pgtype.Text
	if mitPlan != nil {
		mitPlanStr = pgtype.Text{String: *mitPlan, Valid: true}
	}

	updatedRisk, err := qtx.AssessRisk(ctx, db.AssessRiskParams{
		ID:             existing.ID,
		Likelihood:     pgtype.Int4{Int32: params.Likelihood, Valid: true},
		Impact:         pgtype.Int4{Int32: params.Impact, Valid: true},
		Level:          db.NullRiskLevel{RiskLevel: level, Valid: true},
		MitigationPlan: mitPlanStr,
	})
	if err != nil {
		return RiskDTO{}, fmt.Errorf("assess risk: %w", err)
	}

	if s.auditLog != nil {
		before := map[string]any{
			"status": string(existing.Status),
		}
		if existing.Likelihood.Valid {
			before["likelihood"] = existing.Likelihood.Int32
		}
		if existing.Impact.Valid {
			before["impact"] = existing.Impact.Int32
		}
		if existing.Level.Valid {
			before["level"] = string(existing.Level.RiskLevel)
		}

		after := map[string]any{
			"likelihood": params.Likelihood,
			"impact":     params.Impact,
			"score":      score,
			"level":      string(level),
			"status":     string(updatedRisk.Status),
		}
		if mitPlan != nil {
			after["mitigation_plan"] = *mitPlan
		}

		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "risk.assess",
			EntityType: "risk",
			EntityID:   uuid.UUID(existing.ID.Bytes).String(),
			ProjectID:  existing.ProjectID,
			Before:     before,
			After:      after,
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return RiskDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return RiskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	detail, err := s.queries.GetRiskByIDWithDetails(ctx, updatedRisk.ID)
	if err != nil {
		return mapRowToDTO(detail), nil
	}
	return mapRowToDTO(detail), nil
}

// CloseRisk closes a risk.
// Permission: Hanya Risk Management.
// Aturan: Risiko High / Critical tidak dapat ditutup jika mitigation_plan masih kosong.
func (s *Service) CloseRisk(ctx context.Context, actor ActorInfo, id string) (RiskDTO, error) {
	if actor.Role != db.UserRoleRiskManagement {
		return RiskDTO{}, ErrOnlyRiskManagementClose
	}

	rUUID, err := parseUUID(id)
	if err != nil {
		return RiskDTO{}, ErrRiskNotFound
	}

	existing, err := s.queries.GetRiskByID(ctx, rUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RiskDTO{}, ErrRiskNotFound
		}
		return RiskDTO{}, fmt.Errorf("get risk: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, existing.ProjectID)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("get project: %w", err)
	}
	if p.Status == db.ProjectStatusArchived {
		return RiskDTO{}, ErrProjectArchived
	}

	// High / Critical check
	if existing.Level.Valid && (existing.Level.RiskLevel == db.RiskLevelHigh || existing.Level.RiskLevel == db.RiskLevelCritical) {
		if strings.TrimSpace(existing.MitigationPlan) == "" {
			return RiskDTO{}, ErrHighCriticalMitigationRequired
		}
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return RiskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	closedRisk, err := qtx.CloseRisk(ctx, existing.ID)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("close risk: %w", err)
	}

	actorUUID := mustParseUUID(actor.ID)
	if s.auditLog != nil {
		if err := s.auditLog.Log(ctx, auditlog.LogParams{
			ActorID:    actorUUID,
			ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
			Action:     "risk.close",
			EntityType: "risk",
			EntityID:   uuid.UUID(existing.ID.Bytes).String(),
			ProjectID:  existing.ProjectID,
			Before: map[string]any{
				"status": string(existing.Status),
			},
			After: map[string]any{
				"status":    string(closedRisk.Status),
				"closed_at": closedRisk.ClosedAt.Time,
			},
			IP:        actor.IP,
			RequestID: actor.RequestID,
		}); err != nil {
			return RiskDTO{}, fmt.Errorf("audit log: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return RiskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	detail, err := s.queries.GetRiskByIDWithDetails(ctx, closedRisk.ID)
	if err != nil {
		return mapRowToDTO(detail), nil
	}
	return mapRowToDTO(detail), nil
}

// UpdateRisk handles updating risk fields.
// Risk Management: can update all fields (title, description, category, task_id, owner_id, status, mitigation_plan).
// PM: can ONLY update mitigation_plan on projects they own.
func (s *Service) UpdateRisk(ctx context.Context, actor ActorInfo, id string, params UpdateRiskParams) (RiskDTO, error) {
	rUUID, err := parseUUID(id)
	if err != nil {
		return RiskDTO{}, ErrRiskNotFound
	}

	existing, err := s.queries.GetRiskByID(ctx, rUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RiskDTO{}, ErrRiskNotFound
		}
		return RiskDTO{}, fmt.Errorf("get risk: %w", err)
	}

	p, err := s.queries.GetProjectByID(ctx, existing.ProjectID)
	if err != nil {
		return RiskDTO{}, fmt.Errorf("get project: %w", err)
	}
	if p.Status == db.ProjectStatusArchived {
		return RiskDTO{}, ErrProjectArchived
	}

	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return RiskDTO{}, ErrForbidden
	}

	var qtx db.Querier = s.queries
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return RiskDTO{}, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		qtx = db.New(tx)
	}

	var updatedID pgtype.UUID

	switch actor.Role {
	case db.UserRoleRiskManagement:
		title := existing.Title
		if params.Title != nil {
			t := strings.TrimSpace(*params.Title)
			if t == "" {
				return RiskDTO{}, fmt.Errorf("%w: judul risiko tidak boleh kosong", ErrInvalidInput)
			}
			title = t
		}

		desc := existing.Description
		if params.Description != nil {
			desc = strings.TrimSpace(*params.Description)
		}

		category := existing.Category
		if params.Category != nil {
			c, err := parseRiskCategory(*params.Category)
			if err != nil {
				return RiskDTO{}, err
			}
			category = c
		}

		taskUUID := existing.TaskID
		if params.TaskID != nil {
			if strings.TrimSpace(*params.TaskID) == "" {
				taskUUID = pgtype.UUID{Valid: false}
			} else {
				tu, err := parseUUID(*params.TaskID)
				if err != nil {
					return RiskDTO{}, fmt.Errorf("%w: format task_id tidak valid", ErrInvalidInput)
				}
				taskObj, err := s.queries.GetTaskByID(ctx, tu)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return RiskDTO{}, ErrTaskNotFound
					}
					return RiskDTO{}, fmt.Errorf("get task: %w", err)
				}
				if taskObj.ProjectID != p.ID {
					return RiskDTO{}, fmt.Errorf("%w: task tidak berada dalam proyek ini", ErrInvalidInput)
				}
				taskUUID = tu
			}
		}

		ownerUUID := existing.OwnerID
		if params.OwnerID != nil {
			if strings.TrimSpace(*params.OwnerID) == "" {
				ownerUUID = pgtype.UUID{Valid: false}
			} else {
				ou, err := parseUUID(*params.OwnerID)
				if err != nil {
					return RiskDTO{}, fmt.Errorf("%w: format owner_id tidak valid", ErrInvalidInput)
				}
				ownerUUID = ou
			}
		}

		status := existing.Status
		if params.Status != nil {
			st, err := parseRiskStatus(*params.Status)
			if err != nil {
				return RiskDTO{}, err
			}
			status = st
		}

		mitigation := existing.MitigationPlan
		if params.MitigationPlan != nil {
			mitigation = strings.TrimSpace(*params.MitigationPlan)
		}

		updated, err := qtx.UpdateRiskByRiskManagement(ctx, db.UpdateRiskByRiskManagementParams{
			ID:             existing.ID,
			Title:          title,
			Description:    desc,
			Category:       category,
			TaskID:         taskUUID,
			OwnerID:        ownerUUID,
			Status:         status,
			MitigationPlan: mitigation,
		})
		if err != nil {
			return RiskDTO{}, fmt.Errorf("update risk: %w", err)
		}
		updatedID = updated.ID

		if s.auditLog != nil {
			if err := s.auditLog.Log(ctx, auditlog.LogParams{
				ActorID:    actorUUID,
				ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
				Action:     "risk.update",
				EntityType: "risk",
				EntityID:   uuid.UUID(existing.ID.Bytes).String(),
				ProjectID:  existing.ProjectID,
				Before: map[string]any{
					"title":           existing.Title,
					"category":        string(existing.Category),
					"status":          string(existing.Status),
					"mitigation_plan": existing.MitigationPlan,
				},
				After: map[string]any{
					"title":           updated.Title,
					"category":        string(updated.Category),
					"status":          string(updated.Status),
					"mitigation_plan": updated.MitigationPlan,
				},
				IP:        actor.IP,
				RequestID: actor.RequestID,
			}); err != nil {
				return RiskDTO{}, fmt.Errorf("audit log: %w", err)
			}
		}

	case db.UserRoleProjectManager:
		if p.OwnerID != actorUUID {
			return RiskDTO{}, ErrForbidden
		}
		// PM is ONLY allowed to update mitigation plan!
		if params.Title != nil || params.Description != nil || params.Category != nil ||
			params.TaskID != nil || params.OwnerID != nil || params.Status != nil {
			return RiskDTO{}, ErrOnlyPMUpdateMitigation
		}
		if params.MitigationPlan == nil {
			return RiskDTO{}, fmt.Errorf("%w: rencana mitigasi harus disertakan", ErrInvalidInput)
		}

		mitigation := strings.TrimSpace(*params.MitigationPlan)
		updated, err := qtx.UpdateRiskMitigationPlan(ctx, db.UpdateRiskMitigationPlanParams{
			ID:             existing.ID,
			MitigationPlan: mitigation,
		})
		if err != nil {
			return RiskDTO{}, fmt.Errorf("update mitigation plan: %w", err)
		}
		updatedID = updated.ID

		if s.auditLog != nil {
			if err := s.auditLog.Log(ctx, auditlog.LogParams{
				ActorID:    actorUUID,
				ActorRole:  db.NullUserRole{UserRole: actor.Role, Valid: true},
				Action:     "risk.update_mitigation",
				EntityType: "risk",
				EntityID:   uuid.UUID(existing.ID.Bytes).String(),
				ProjectID:  existing.ProjectID,
				Before: map[string]any{
					"mitigation_plan": existing.MitigationPlan,
					"status":          string(existing.Status),
				},
				After: map[string]any{
					"mitigation_plan": updated.MitigationPlan,
					"status":          string(updated.Status),
				},
				IP:        actor.IP,
				RequestID: actor.RequestID,
			}); err != nil {
				return RiskDTO{}, fmt.Errorf("audit log: %w", err)
			}
		}

	default:
		return RiskDTO{}, ErrForbidden
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return RiskDTO{}, fmt.Errorf("commit tx: %w", err)
		}
	}

	detail, err := s.queries.GetRiskByIDWithDetails(ctx, updatedID)
	if err != nil {
		return mapRowToDTO(detail), nil
	}
	return mapRowToDTO(detail), nil
}

// ListRisks returns risks filtered and paginated with scoping for the actor's role.
func (s *Service) ListRisks(ctx context.Context, actor ActorInfo, params ListRisksParams) ([]RiskDTO, int64, error) {
	projUUIDs, err := s.resolveAllowedProjectIDs(ctx, actor, params.ProjectID)
	if err != nil {
		return nil, 0, err
	}

	var specificProjectID pgtype.UUID
	if params.ProjectID != nil && strings.TrimSpace(*params.ProjectID) != "" {
		pu, err := parseUUID(*params.ProjectID)
		if err == nil {
			specificProjectID = pu
		}
	}

	var specificTaskID pgtype.UUID
	if params.TaskID != nil && strings.TrimSpace(*params.TaskID) != "" {
		tu, err := parseUUID(*params.TaskID)
		if err == nil {
			specificTaskID = tu
		}
	}

	var levelFilter db.NullRiskLevel
	if params.Level != nil && strings.TrimSpace(*params.Level) != "" {
		lvl, err := parseRiskLevel(*params.Level)
		if err == nil {
			levelFilter = db.NullRiskLevel{RiskLevel: lvl, Valid: true}
		}
	}

	var statusFilter db.NullRiskStatus
	if params.Status != nil && strings.TrimSpace(*params.Status) != "" {
		st, err := parseRiskStatus(*params.Status)
		if err == nil {
			statusFilter = db.NullRiskStatus{RiskStatus: st, Valid: true}
		}
	}

	var categoryFilter db.NullRiskCategory
	if params.Category != nil && strings.TrimSpace(*params.Category) != "" {
		cat, err := parseRiskCategory(*params.Category)
		if err == nil {
			categoryFilter = db.NullRiskCategory{RiskCategory: cat, Valid: true}
		}
	}

	var searchFilter pgtype.Text
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		searchFilter = pgtype.Text{String: strings.TrimSpace(*params.Search), Valid: true}
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.queries.ListRisksFiltered(ctx, db.ListRisksFilteredParams{
		ProjectIds: projUUIDs,
		ProjectID:  specificProjectID,
		TaskID:     specificTaskID,
		Level:      levelFilter,
		Status:     statusFilter,
		Category:   categoryFilter,
		Search:     searchFilter,
		LimitVal:   limit,
		OffsetVal:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list risks: %w", err)
	}

	total, err := s.queries.CountRisksFiltered(ctx, db.CountRisksFilteredParams{
		ProjectIds: projUUIDs,
		ProjectID:  specificProjectID,
		TaskID:     specificTaskID,
		Level:      levelFilter,
		Status:     statusFilter,
		Category:   categoryFilter,
		Search:     searchFilter,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count risks: %w", err)
	}

	dtos := make([]RiskDTO, len(rows))
	for i, r := range rows {
		dtos[i] = mapListRowToDTO(r)
	}

	return dtos, total, nil
}

// GetRisksSummary aggregates stats and builds the 5x5 heatmap grid for the actor.
func (s *Service) GetRisksSummary(ctx context.Context, actor ActorInfo, projectID *string) (RiskSummaryDTO, error) {
	projUUIDs, err := s.resolveAllowedProjectIDs(ctx, actor, projectID)
	if err != nil {
		return RiskSummaryDTO{}, err
	}

	var specificProjectID pgtype.UUID
	if projectID != nil && strings.TrimSpace(*projectID) != "" {
		pu, err := parseUUID(*projectID)
		if err == nil {
			specificProjectID = pu
		}
	}

	stats, err := s.queries.GetRisksSummaryStats(ctx, db.GetRisksSummaryStatsParams{
		ProjectIds: projUUIDs,
		ProjectID:  specificProjectID,
	})
	if err != nil {
		return RiskSummaryDTO{}, fmt.Errorf("get risks summary stats: %w", err)
	}

	heatmapCounts, err := s.queries.GetRisksHeatmapCounts(ctx, db.GetRisksHeatmapCountsParams{
		ProjectIds:    projUUIDs,
		ProjectID:     specificProjectID,
		IncludeClosed: pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		return RiskSummaryDTO{}, fmt.Errorf("get heatmap counts: %w", err)
	}

	countsMap := make(map[string]int64)
	for _, row := range heatmapCounts {
		if row.Likelihood.Valid && row.Impact.Valid {
			key := fmt.Sprintf("%d_%d", row.Likelihood.Int32, row.Impact.Int32)
			countsMap[key] = row.Count
		}
	}

	// 5x5 grid from likelihood 5 down to 1, impact 1 to 5
	heatmap := make([]HeatmapCellDTO, 0, 25)
	for l := int32(5); l >= 1; l-- {
		for i := int32(1); i <= 5; i++ {
			score, lvl, _ := CalculateRiskScoreAndLevel(l, i)
			key := fmt.Sprintf("%d_%d", l, i)
			heatmap = append(heatmap, HeatmapCellDTO{
				Likelihood: l,
				Impact:     i,
				Score:      score,
				Level:      string(lvl),
				Count:      countsMap[key],
			})
		}
	}

	byLevel := map[string]int64{
		"low":      stats.LowLevelCount,
		"medium":   stats.MediumLevelCount,
		"high":     stats.HighLevelCount,
		"critical": stats.CriticalLevelCount,
	}

	return RiskSummaryDTO{
		TotalRisks:        stats.TotalRisks,
		OpenRisks:         stats.OpenRisks,
		HighCriticalRisks: stats.HighCriticalRisks,
		UnmitigatedRisks:  stats.UnmitigatedRisks,
		ByLevel:           byLevel,
		Heatmap:           heatmap,
	}, nil
}

// CountRisksByTaskIDs returns risk metrics for a list of task IDs.
func (s *Service) CountRisksByTaskIDs(ctx context.Context, taskIDs []string) (map[string]TaskRiskCountDTO, error) {
	if len(taskIDs) == 0 {
		return map[string]TaskRiskCountDTO{}, nil
	}

	uuids := make([]pgtype.UUID, 0, len(taskIDs))
	for _, idStr := range taskIDs {
		u, err := parseUUID(idStr)
		if err == nil {
			uuids = append(uuids, u)
		}
	}

	if len(uuids) == 0 {
		return map[string]TaskRiskCountDTO{}, nil
	}

	rows, err := s.queries.CountRisksByTaskIDs(ctx, uuids)
	if err != nil {
		return nil, fmt.Errorf("count risks by task ids: %w", err)
	}

	result := make(map[string]TaskRiskCountDTO, len(rows))
	for _, r := range rows {
		if r.TaskID.Valid {
			tid := uuid.UUID(r.TaskID.Bytes).String()
			result[tid] = TaskRiskCountDTO{
				TaskID:            tid,
				TotalRisks:        r.Count,
				HighCriticalRisks: r.HighCriticalCount,
			}
		}
	}

	return result, nil
}

// resolveAllowedProjectIDs computes which projects the actor is allowed to view.
func (s *Service) resolveAllowedProjectIDs(ctx context.Context, actor ActorInfo, requestedProjectID *string) ([]pgtype.UUID, error) {
	actorUUID, err := parseUUID(actor.ID)
	if err != nil {
		return nil, ErrForbidden
	}

	switch actor.Role {
	case db.UserRoleAdmin, db.UserRoleAudit, db.UserRoleRiskManagement:
		// Sees all projects; if requestedProjectID is specified, we don't need to constrain the array
		return nil, nil

	case db.UserRoleProjectManager:
		projects, err := s.queries.ListProjectsByOwner(ctx, actorUUID)
		if err != nil {
			return nil, fmt.Errorf("list projects by owner: %w", err)
		}
		if requestedProjectID != nil && strings.TrimSpace(*requestedProjectID) != "" {
			reqUUID, err := parseUUID(*requestedProjectID)
			if err != nil {
				return nil, ErrRiskNotFound
			}
			found := false
			for _, p := range projects {
				if p.ID == reqUUID {
					found = true
					break
				}
			}
			if !found {
				return nil, ErrForbidden
			}
		}
		uuids := make([]pgtype.UUID, len(projects))
		for i, p := range projects {
			uuids[i] = p.ID
		}
		return uuids, nil

	case db.UserRoleTeamLeader:
		projects, err := s.queries.ListProjectsByMember(ctx, actorUUID)
		if err != nil {
			return nil, fmt.Errorf("list projects by member: %w", err)
		}
		if requestedProjectID != nil && strings.TrimSpace(*requestedProjectID) != "" {
			reqUUID, err := parseUUID(*requestedProjectID)
			if err != nil {
				return nil, ErrRiskNotFound
			}
			found := false
			for _, p := range projects {
				if p.ID == reqUUID {
					found = true
					break
				}
			}
			if !found {
				return nil, ErrForbidden
			}
		}
		uuids := make([]pgtype.UUID, len(projects))
		for i, p := range projects {
			uuids[i] = p.ID
		}
		return uuids, nil

	default:
		return nil, ErrForbidden
	}
}

// ─── Helpers & Mappers ────────────────────────────────────────────────────────

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func mustParseUUID(s string) pgtype.UUID {
	u, _ := parseUUID(s)
	return u
}

func parseRiskCategory(s string) (db.RiskCategory, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "operational":
		return db.RiskCategoryOperational, nil
	case "schedule":
		return db.RiskCategorySchedule, nil
	case "technical":
		return db.RiskCategoryTechnical, nil
	case "compliance":
		return db.RiskCategoryCompliance, nil
	case "financial":
		return db.RiskCategoryFinancial, nil
	case "security":
		return db.RiskCategorySecurity, nil
	default:
		return "", fmt.Errorf("%w: kategori risiko '%s' tidak valid", ErrInvalidInput, s)
	}
}

func parseRiskLevel(s string) (db.RiskLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return db.RiskLevelLow, nil
	case "medium":
		return db.RiskLevelMedium, nil
	case "high":
		return db.RiskLevelHigh, nil
	case "critical":
		return db.RiskLevelCritical, nil
	default:
		return "", fmt.Errorf("%w: level risiko '%s' tidak valid", ErrInvalidInput, s)
	}
}

func parseRiskStatus(s string) (db.RiskStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "identified":
		return db.RiskStatusIdentified, nil
	case "assessed":
		return db.RiskStatusAssessed, nil
	case "mitigating":
		return db.RiskStatusMitigating, nil
	case "accepted":
		return db.RiskStatusAccepted, nil
	case "closed":
		return db.RiskStatusClosed, nil
	default:
		return "", fmt.Errorf("%w: status risiko '%s' tidak valid", ErrInvalidInput, s)
	}
}

func mapRowToDTO(r db.GetRiskByIDWithDetailsRow) RiskDTO {
	dto := RiskDTO{
		ID:             uuid.UUID(r.ID.Bytes).String(),
		Code:           r.Code,
		ProjectID:      uuid.UUID(r.ProjectID.Bytes).String(),
		ProjectName:    r.ProjectName,
		ProjectCode:    r.ProjectCode,
		Title:          r.Title,
		Description:    r.Description,
		Category:       string(r.Category),
		Status:         string(r.Status),
		MitigationPlan: r.MitigationPlan,
		RaisedBy:       uuid.UUID(r.RaisedBy.Bytes).String(),
		RaisedByName:   r.RaisedByName,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}

	if r.TaskID.Valid {
		tid := uuid.UUID(r.TaskID.Bytes).String()
		dto.TaskID = &tid
	}
	if r.TaskTitle.Valid {
		dto.TaskTitle = &r.TaskTitle.String
	}
	if r.OwnerID.Valid {
		oid := uuid.UUID(r.OwnerID.Bytes).String()
		dto.OwnerID = &oid
	}
	if r.OwnerName.Valid {
		dto.OwnerName = &r.OwnerName.String
	}
	if r.Likelihood.Valid {
		dto.Likelihood = &r.Likelihood.Int32
	}
	if r.Impact.Valid {
		dto.Impact = &r.Impact.Int32
	}
	if r.Score.Valid {
		dto.Score = &r.Score.Int32
	}
	if r.Level.Valid {
		lvl := string(r.Level.RiskLevel)
		dto.Level = &lvl
	}
	if r.ClosedAt.Valid {
		dto.ClosedAt = &r.ClosedAt.Time
	}

	return dto
}

func mapListRowToDTO(r db.ListRisksFilteredRow) RiskDTO {
	dto := RiskDTO{
		ID:             uuid.UUID(r.ID.Bytes).String(),
		Code:           r.Code,
		ProjectID:      uuid.UUID(r.ProjectID.Bytes).String(),
		ProjectName:    r.ProjectName,
		ProjectCode:    r.ProjectCode,
		Title:          r.Title,
		Description:    r.Description,
		Category:       string(r.Category),
		Status:         string(r.Status),
		MitigationPlan: r.MitigationPlan,
		RaisedBy:       uuid.UUID(r.RaisedBy.Bytes).String(),
		RaisedByName:   r.RaisedByName,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}

	if r.TaskID.Valid {
		tid := uuid.UUID(r.TaskID.Bytes).String()
		dto.TaskID = &tid
	}
	if r.TaskTitle.Valid {
		dto.TaskTitle = &r.TaskTitle.String
	}
	if r.OwnerID.Valid {
		oid := uuid.UUID(r.OwnerID.Bytes).String()
		dto.OwnerID = &oid
	}
	if r.OwnerName.Valid {
		dto.OwnerName = &r.OwnerName.String
	}
	if r.Likelihood.Valid {
		dto.Likelihood = &r.Likelihood.Int32
	}
	if r.Impact.Valid {
		dto.Impact = &r.Impact.Int32
	}
	if r.Score.Valid {
		dto.Score = &r.Score.Int32
	}
	if r.Level.Valid {
		lvl := string(r.Level.RiskLevel)
		dto.Level = &lvl
	}
	if r.ClosedAt.Valid {
		dto.ClosedAt = &r.ClosedAt.Time
	}

	return dto
}
