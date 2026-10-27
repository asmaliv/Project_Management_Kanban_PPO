package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"ppo-kanban/internal/auditlog"
)

type AuditLogHandler struct {
	auditService *auditlog.Service
}

func NewAuditLogHandler(auditService *auditlog.Service) *AuditLogHandler {
	return &AuditLogHandler{auditService: auditService}
}

// HandleListAuditLogs returns audit logs with optional filters.
// Query params: actor_id, action, entity_type, entity_id, project_id, date_from (RFC3339), date_to (RFC3339), page, page_size
func (h *AuditLogHandler) HandleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse filters
	filters := auditlog.ListFilters{
		Limit:  20, // default
		Offset: 0,
	}

	if actorIDStr := query.Get("actor_id"); actorIDStr != "" {
		parsed, err := parseUUID(actorIDStr)
		if err == nil {
			filters.ActorID = parsed
		}
	}

	if action := query.Get("action"); action != "" {
		filters.Action = pgtype.Text{String: action, Valid: true}
	}

	if entityType := query.Get("entity_type"); entityType != "" {
		filters.EntityType = pgtype.Text{String: entityType, Valid: true}
	}

	if entityIDStr := query.Get("entity_id"); entityIDStr != "" {
		filters.EntityID = pgtype.Text{String: entityIDStr, Valid: true}
	}

	if projectIDStr := query.Get("project_id"); projectIDStr != "" {
		parsed, err := parseUUID(projectIDStr)
		if err == nil {
			filters.ProjectID = parsed
		}
	}

	if dateFromStr := query.Get("date_from"); dateFromStr != "" {
		t, err := time.Parse(time.RFC3339, dateFromStr)
		if err == nil {
			filters.DateFrom = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	if dateToStr := query.Get("date_to"); dateToStr != "" {
		t, err := time.Parse(time.RFC3339, dateToStr)
		if err == nil {
			filters.DateTo = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	if pageStr := query.Get("page"); pageStr != "" {
		page, _ := strconv.Atoi(pageStr)
		if page > 0 {
			filters.Offset = int32((page - 1) * int(filters.Limit))
		}
	}

	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		pageSize, _ := strconv.Atoi(pageSizeStr)
		if pageSize > 0 && pageSize <= 100 {
			filters.Limit = int32(pageSize)
		}
	}

	logs, err := h.auditService.Read(r.Context(), filters)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal membaca audit log.", nil)
		return
	}

	JSON(w, http.StatusOK, logs)
}

// HandleExportAuditLogsCSV streams audit logs as a downloadable CSV file.
// Strictly restricted to audit role. Maximum 5000 records per export.
func (h *AuditLogHandler) HandleExportAuditLogsCSV(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filters := auditlog.ListFilters{
		Limit:  5000,
		Offset: 0,
	}

	if actorIDStr := query.Get("actor_id"); actorIDStr != "" {
		parsed, err := parseUUID(actorIDStr)
		if err == nil {
			filters.ActorID = parsed
		}
	}

	if action := query.Get("action"); action != "" {
		filters.Action = pgtype.Text{String: action, Valid: true}
	}

	if entityType := query.Get("entity_type"); entityType != "" {
		filters.EntityType = pgtype.Text{String: entityType, Valid: true}
	}

	if entityIDStr := query.Get("entity_id"); entityIDStr != "" {
		filters.EntityID = pgtype.Text{String: entityIDStr, Valid: true}
	}

	if projectIDStr := query.Get("project_id"); projectIDStr != "" {
		parsed, err := parseUUID(projectIDStr)
		if err == nil {
			filters.ProjectID = parsed
		}
	}

	if dateFromStr := query.Get("date_from"); dateFromStr != "" {
		t, err := time.Parse(time.RFC3339, dateFromStr)
		if err == nil {
			filters.DateFrom = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	if dateToStr := query.Get("date_to"); dateToStr != "" {
		t, err := time.Parse(time.RFC3339, dateToStr)
		if err == nil {
			filters.DateTo = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	logs, err := h.auditService.Read(r.Context(), filters)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Gagal membaca audit log untuk ekspor.", nil)
		return
	}

	filename := fmt.Sprintf("audit-trail-%s.csv", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// CSV Header (Safe columns only - never include passwords, hashes, tokens, or raw secrets)
	_ = csvWriter.Write([]string{
		"ID",
		"Waktu (UTC)",
		"Actor ID",
		"Role",
		"Aksi",
		"Tipe Entitas",
		"ID Entitas",
		"Project ID",
		"IP",
		"Request ID",
	})

	for _, log := range logs {
		actorID := ""
		if log.ActorID.Valid {
			actorID = uuid.UUID(log.ActorID.Bytes).String()
		}
		actorRole := ""
		if log.ActorRole.Valid {
			actorRole = string(log.ActorRole.UserRole)
		}
		entityID := log.EntityID
		projectID := ""
		if log.ProjectID.Valid {
			projectID = uuid.UUID(log.ProjectID.Bytes).String()
		}
		ip := log.Ip
		reqID := log.RequestID

		_ = csvWriter.Write([]string{
			strconv.FormatInt(log.ID, 10),
			log.OccurredAt.Time.UTC().Format(time.RFC3339),
			actorID,
			actorRole,
			log.Action,
			log.EntityType,
			entityID,
			projectID,
			ip,
			reqID,
		})
	}
}

// parseUUID is a helper to convert string to pgtype.UUID
func parseUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
