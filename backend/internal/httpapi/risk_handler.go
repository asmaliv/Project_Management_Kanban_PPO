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
	"ppo-kanban/internal/risk"
)

// RiskHandler handles HTTP requests for risk register and assessment.
type RiskHandler struct {
	service *risk.Service
}

// NewRiskHandler creates a RiskHandler.
func NewRiskHandler(service *risk.Service) *RiskHandler {
	return &RiskHandler{service: service}
}

func (h *RiskHandler) actor(r *http.Request) (risk.ActorInfo, bool) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		return risk.ActorInfo{}, false
	}
	return risk.ActorInfo{
		ID:        session.User.ID,
		Role:      db.UserRole(session.User.Role),
		RequestID: middleware.GetReqID(r.Context()),
		IP:        r.RemoteAddr,
	}, true
}

func mapRiskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, risk.ErrRiskNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Risiko tidak ditemukan.", nil)
	case errors.Is(err, risk.ErrProjectNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
	case errors.Is(err, risk.ErrTaskNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Task tidak ditemukan.", nil)
	case errors.Is(err, risk.ErrForbidden):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki izin untuk melakukan aksi ini.", nil)
	case errors.Is(err, risk.ErrProjectArchived):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Proyek sudah diarsipkan dan bersifat read-only.", nil)
	case errors.Is(err, risk.ErrOnlyRiskManagementAssess):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Hanya Risk Management yang dapat menilai risiko.", nil)
	case errors.Is(err, risk.ErrOnlyRiskManagementClose):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Hanya Risk Management yang dapat menutup risiko.", nil)
	case errors.Is(err, risk.ErrOnlyPMUpdateMitigation):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Project Manager hanya dapat memperbarui rencana mitigasi pada proyek miliknya.", nil)
	case errors.Is(err, risk.ErrHighCriticalMitigationRequired):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Risiko High/Critical tidak dapat ditutup tanpa rencana mitigasi terisi.", nil)
	case errors.Is(err, risk.ErrInvalidInput):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	default:
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan pada server.", nil)
	}
}

// ─── HTTP Endpoints ──────────────────────────────────────────────────────────

// HandleListRisks returns risks filtered and scoped according to role.
// GET /api/v1/risks
func (h *RiskHandler) HandleListRisks(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	q := r.URL.Query()
	var projID *string
	if pid := strings.TrimSpace(q.Get("project_id")); pid != "" {
		projID = &pid
	}

	var taskID *string
	if tid := strings.TrimSpace(q.Get("task_id")); tid != "" {
		taskID = &tid
	}

	var level *string
	if lvl := strings.TrimSpace(q.Get("level")); lvl != "" {
		level = &lvl
	}

	var status *string
	if st := strings.TrimSpace(q.Get("status")); st != "" {
		status = &st
	}

	var category *string
	if cat := strings.TrimSpace(q.Get("category")); cat != "" {
		category = &cat
	}

	var search *string
	if s := strings.TrimSpace(q.Get("search")); s != "" {
		search = &s
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
	}

	limit := int32(perPage)
	offset := int32((page - 1) * perPage)

	risks, total, err := h.service.ListRisks(r.Context(), actor, risk.ListRisksParams{
		ProjectID: projID,
		TaskID:    taskID,
		Level:     level,
		Status:    status,
		Category:  category,
		Search:    search,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"risks":    risks,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// HandleCreateRisk handles raising a new risk.
// POST /api/v1/risks
func (h *RiskHandler) HandleCreateRisk(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	var body struct {
		ProjectID   string  `json:"project_id"`
		TaskID      *string `json:"task_id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Category    string  `json:"category"`
		OwnerID     *string `json:"owner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_JSON", "Format JSON tidak valid.", nil)
		return
	}

	created, err := h.service.CreateRisk(r.Context(), actor, risk.CreateRiskParams{
		ProjectID:   body.ProjectID,
		TaskID:      body.TaskID,
		Title:       body.Title,
		Description: body.Description,
		Category:    body.Category,
		OwnerID:     body.OwnerID,
	})
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusCreated, created)
}

// HandleGetRisk returns a single risk detail.
// GET /api/v1/risks/{id}
func (h *RiskHandler) HandleGetRisk(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	riskID := chi.URLParam(r, "id")
	detail, err := h.service.GetRiskByID(r.Context(), actor, riskID)
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, detail)
}

// HandleUpdateRisk updates risk details (RM all fields, PM mitigation_plan only).
// PATCH /api/v1/risks/{id} or PUT /api/v1/risks/{id}
func (h *RiskHandler) HandleUpdateRisk(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	riskID := chi.URLParam(r, "id")

	var body risk.UpdateRiskParams
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_JSON", "Format JSON tidak valid.", nil)
		return
	}

	updated, err := h.service.UpdateRisk(r.Context(), actor, riskID, body)
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, updated)
}

// HandleAssessRisk calculates and sets likelihood and impact (RM only).
// POST /api/v1/risks/{id}/assess
func (h *RiskHandler) HandleAssessRisk(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	riskID := chi.URLParam(r, "id")

	var body struct {
		Likelihood     int32   `json:"likelihood"`
		Impact         int32   `json:"impact"`
		MitigationPlan *string `json:"mitigation_plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_JSON", "Format JSON tidak valid.", nil)
		return
	}

	assessed, err := h.service.AssessRisk(r.Context(), actor, riskID, risk.AssessRiskParams{
		Likelihood:     body.Likelihood,
		Impact:         body.Impact,
		MitigationPlan: body.MitigationPlan,
	})
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, assessed)
}

// HandleCloseRisk closes a risk (RM only).
// POST /api/v1/risks/{id}/close
func (h *RiskHandler) HandleCloseRisk(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	riskID := chi.URLParam(r, "id")
	closed, err := h.service.CloseRisk(r.Context(), actor, riskID)
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, closed)
}

// HandleGetRiskSummary returns heatmap and aggregated stats for risks.
// GET /api/v1/risks/summary
func (h *RiskHandler) HandleGetRiskSummary(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	var projID *string
	if pid := strings.TrimSpace(r.URL.Query().Get("project_id")); pid != "" {
		projID = &pid
	}

	summary, err := h.service.GetRisksSummary(r.Context(), actor, projID)
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, summary)
}

// HandleGetTaskRiskCounts returns risk indicators for a set of tasks.
// GET /api/v1/risks/task-counts?task_ids=uuid1,uuid2
func (h *RiskHandler) HandleGetTaskRiskCounts(w http.ResponseWriter, r *http.Request) {
	_, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	rawIDs := r.URL.Query().Get("task_ids")
	if strings.TrimSpace(rawIDs) == "" {
		JSON(w, http.StatusOK, map[string]any{"counts": map[string]any{}})
		return
	}

	split := strings.Split(rawIDs, ",")
	counts, err := h.service.CountRisksByTaskIDs(r.Context(), split)
	if err != nil {
		mapRiskError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{"counts": counts})
}
