package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/db"
)

// Service provides high-level audit logging operations.
type Service struct {
	store  *Store
	logger *slog.Logger
}

// NewService creates an audit log service.
func NewService(store *Store, logger *slog.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

// LogParams holds context for an audit log entry.
type LogParams struct {
	ActorID    pgtype.UUID      // nullable for login failures
	ActorRole  db.NullUserRole  // nullable
	Action     string
	EntityType string
	EntityID   string           // nullable, stored as text
	ProjectID  pgtype.UUID      // nullable
	Before     map[string]any   // will be marshaled to JSONB, MUST NOT contain secrets
	After      map[string]any   // will be marshaled to JSONB, MUST NOT contain secrets
	IP         string
	RequestID  string
}

// Log writes an audit log entry. Must be called within the same transaction
// as the audited action when applicable.
func (s *Service) Log(ctx context.Context, params LogParams) error {
	var beforeJSON, afterJSON []byte
	var err error

	if params.Before != nil {
		beforeJSON, err = json.Marshal(params.Before)
		if err != nil {
			return fmt.Errorf("marshal before: %w", err)
		}
	}

	if params.After != nil {
		afterJSON, err = json.Marshal(params.After)
		if err != nil {
			return fmt.Errorf("marshal after: %w", err)
		}
	}

	if err := s.store.Insert(ctx, InsertParams{
		ActorID:    params.ActorID,
		ActorRole:  params.ActorRole,
		Action:     params.Action,
		EntityType: params.EntityType,
		EntityID:   params.EntityID,
		ProjectID:  params.ProjectID,
		Before:     beforeJSON,
		After:      afterJSON,
		IP:         params.IP,
		RequestID:  params.RequestID,
	}); err != nil {
		s.logger.Error("failed to write audit log",
			slog.String("action", params.Action),
			slog.String("entity_type", params.EntityType),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("write audit log: %w", err)
	}

	s.logger.Debug("audit log written",
		slog.String("action", params.Action),
		slog.String("entity_type", params.EntityType),
	)
	return nil
}

// Read retrieves audit logs with the given filters.
// Only accessible by admin and audit roles (enforced at handler layer).
func (s *Service) Read(ctx context.Context, filters ListFilters) ([]db.AuditLog, error) {
	logs, err := s.store.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, nil
}
