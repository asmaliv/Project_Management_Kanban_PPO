package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ppo-kanban/internal/db"
	"ppo-kanban/internal/finding"
)

// FindingHandler handles HTTP requests for audit findings.
type FindingHandler struct {
	service *finding.Service
}

// NewFindingHandler creates a FindingHandler.
func NewFindingHandler(service *finding.Service) *FindingHandler {
	return &FindingHandler{service: service}
}

func (h *FindingHandler) actor(r *http.Request) (finding.ActorInfo, bool) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		return finding.ActorInfo{}, false
	}
	return finding.ActorInfo{
		ID:        session.User.ID,
		Role:      db.UserRole(session.User.Role),
		RequestID: middleware.GetReqID(r.Context()),
		IP:        r.RemoteAddr,
	}, true
}

func mapFindingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, finding.ErrFindingNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Temuan audit tidak ditemukan.", nil)
	case errors.Is(err, finding.ErrProjectNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
	case errors.Is(err, finding.ErrTaskNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Task tidak ditemukan.", nil)
	case errors.Is(err, finding.ErrForbidden):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, finding.ErrProjectArchived):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Proyek sudah diarsipkan dan bersifat read-only.", nil)
	case errors.Is(err, finding.ErrInvalidStatusTransition):
		ErrorJSON(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
	case errors.Is(err, finding.ErrInvalidInput):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	default:
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan pada server.", nil)
	}
}

// HandleListFindings returns findings filtered according to role permissions.
func (h *FindingHandler) HandleListFindings(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	q := r.URL.Query()
	var projectID *string
	if val := strings.TrimSpace(q.Get("project_id")); val != "" {
		projectID = &val
	}
	var taskID *string
	if val := strings.TrimSpace(q.Get("task_id")); val != "" {
		taskID = &val
	}
	var severity *string
	if val := strings.TrimSpace(q.Get("severity")); val != "" {
		severity = &val
	}
	var status *string
	if val := strings.TrimSpace(q.Get("status")); val != "" {
		status = &val
	}
	var search *string
	if val := strings.TrimSpace(q.Get("search")); val != "" {
		search = &val
	}

	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}

	perPage := 20
	if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 {
		perPage = pp
		if perPage > 100 {
			perPage = 100
		}
	} else if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 {
		perPage = l
		if perPage > 100 {
			perPage = 100
		}
	}

	limit := int32(perPage)
	offset := int32((page - 1) * perPage)
	if val := q.Get("offset"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	findings, total, err := h.service.ListFindings(r.Context(), actor, finding.ListFindingsParams{
		ProjectID: projectID,
		TaskID:    taskID,
		Severity:  severity,
		Status:    status,
		Search:    search,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"findings": findings,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

type createFindingReq struct {
	ProjectID   string  `json:"project_id"`
	TaskID      *string `json:"task_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
}

// HandleCreateFinding creates a new audit finding (audit role only).
func (h *FindingHandler) HandleCreateFinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	var req createFindingReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid.", nil)
		return
	}

	res, err := h.service.CreateFinding(r.Context(), actor, finding.CreateFindingParams{
		ProjectID:   req.ProjectID,
		TaskID:      req.TaskID,
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
	})
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusCreated, res)
}

// HandleGetFinding retrieves a finding by ID.
func (h *FindingHandler) HandleGetFinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	findingID := chi.URLParam(r, "id")
	res, err := h.service.GetFinding(r.Context(), actor, findingID)
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusOK, res)
}

type respondFindingReq struct {
	Response string `json:"response"`
}

// HandleRespondFinding responds to an open finding (PM owner only).
func (h *FindingHandler) HandleRespondFinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	findingID := chi.URLParam(r, "id")
	var req respondFindingReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid.", nil)
		return
	}

	res, err := h.service.RespondFinding(r.Context(), actor, findingID, finding.RespondFindingParams{
		Response: req.Response,
	})
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusOK, res)
}

// HandleReopenFinding reopens a responded or closed finding (audit only).
func (h *FindingHandler) HandleReopenFinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	findingID := chi.URLParam(r, "id")
	res, err := h.service.ReopenFinding(r.Context(), actor, findingID)
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusOK, res)
}

// HandleCloseFinding closes a responded finding (audit only).
func (h *FindingHandler) HandleCloseFinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	findingID := chi.URLParam(r, "id")
	res, err := h.service.CloseFinding(r.Context(), actor, findingID)
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusOK, res)
}

// HandleGetFindingSummary returns summary statistics for findings.
func (h *FindingHandler) HandleGetFindingSummary(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	var projectID *string
	if val := strings.TrimSpace(r.URL.Query().Get("project_id")); val != "" {
		projectID = &val
	}

	summary, err := h.service.GetFindingSummary(r.Context(), actor, projectID)
	if err != nil {
		mapFindingError(w, err)
		return
	}

	JSON(w, http.StatusOK, summary)
}
