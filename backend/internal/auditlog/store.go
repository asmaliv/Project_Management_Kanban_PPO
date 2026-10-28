package auditlog

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/db"
)

// Store wraps database queries for audit logs.
type Store struct {
	queries db.Querier
}

// NewStore creates a Store instance.
func NewStore(queries db.Querier) *Store {
	return &Store{queries: queries}
}

// InsertParams holds parameters for creating an audit log entry.
type InsertParams struct {
	ActorID    pgtype.UUID      // nullable
	ActorRole  db.NullUserRole  // nullable
	Action     string           // e.g., "auth.login_success", "user.role_changed"
	EntityType string           // e.g., "user", "session", "task"
	EntityID   string           // nullable, stored as text for flexibility
	ProjectID  pgtype.UUID      // nullable
	Before     []byte           // JSONB, nullable
	After      []byte           // JSONB, nullable
	IP         string
	RequestID  string
}

// Insert writes an audit log entry. Caller must ensure this is called within
// the same transaction as the audited action.
func (s *Store) Insert(ctx context.Context, params InsertParams) error {
	if s == nil || s.queries == nil {
		return nil
	}
	_, err := s.queries.InsertAuditLog(ctx, db.InsertAuditLogParams{
		ActorID:    params.ActorID,
		ActorRole:  params.ActorRole,
		Action:     params.Action,
		EntityType: params.EntityType,
		EntityID:   params.EntityID,
		ProjectID:  params.ProjectID,
		Before:     params.Before,
		After:      params.After,
		Ip:         params.IP,
		RequestID:  params.RequestID,
	})
	return err
}

// ListFilters holds optional filters for listing audit logs.
type ListFilters struct {
	ActorID    pgtype.UUID
	Action     pgtype.Text
	EntityType pgtype.Text
	EntityID   pgtype.Text
	ProjectID  pgtype.UUID
	DateFrom   pgtype.Timestamptz
	DateTo     pgtype.Timestamptz
	Limit      int32
	Offset     int32
}

// List retrieves audit logs matching the given filters.
func (s *Store) List(ctx context.Context, filters ListFilters) ([]db.AuditLog, error) {
	if s == nil || s.queries == nil {
		return []db.AuditLog{}, nil
	}
	return s.queries.ListAuditLogs(ctx, db.ListAuditLogsParams{
		Limit:      filters.Limit,
		Offset:     filters.Offset,
		ActorID:    filters.ActorID,
		Action:     filters.Action,
		EntityType: filters.EntityType,
		EntityID:   filters.EntityID,
		ProjectID:  filters.ProjectID,
		DateFrom:   filters.DateFrom,
		DateTo:     filters.DateTo,
	})
}
