package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"ppo-kanban/internal/db"
	"ppo-kanban/internal/project"
)

// ProjectHandler handles HTTP requests for projects, members, and board columns.
type ProjectHandler struct {
	service *project.Service
}

// NewProjectHandler creates a ProjectHandler.
func NewProjectHandler(service *project.Service) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func (h *ProjectHandler) actor(r *http.Request) (project.ActorInfo, bool) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		return project.ActorInfo{}, false
	}
	return project.ActorInfo{
		ID:        session.User.ID,
		Role:      db.UserRole(session.User.Role),
		RequestID: middleware.GetReqID(r.Context()),
		IP:        r.RemoteAddr,
	}, true
}

func mapProjectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, project.ErrProjectNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
	case errors.Is(err, project.ErrForbidden):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki izin untuk melakukan aksi ini.", nil)
	case errors.Is(err, project.ErrProjectArchived):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Proyek sudah diarsipkan dan bersifat read-only.", nil)
	case errors.Is(err, project.ErrColumnDoneProtected):
		ErrorJSON(w, http.StatusConflict, "CONFLICT", "Kolom Done tidak dapat dihapus.", nil)
	case errors.Is(err, project.ErrOneDoneColumnOnly):
		ErrorJSON(w, http.StatusConflict, "CONFLICT", "Proyek sudah memiliki kolom Done.", nil)
	case errors.Is(err, project.ErrMemberNotTeamLeader):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Hanya user dengan role team_leader yang dapat ditambahkan sebagai member.", nil)
	case errors.Is(err, project.ErrTransferTargetNotPM):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Target transfer harus memiliki role project_manager.", nil)
	case errors.Is(err, project.ErrColumnNotEmpty):
		ErrorJSON(w, http.StatusConflict, "CONFLICT", "Kolom masih berisi task, tidak dapat dihapus.", nil)
	default:
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan pada server.", nil)
	}
}

// ─── Projects ────────────────────────────────────────────────────────────────

// HandleListProjects returns all projects visible to the authenticated user.
func (h *ProjectHandler) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	projects, err := h.service.ListProjects(r.Context(), actor)
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, projects)
}

type createProjectReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
}

// HandleCreateProject handles project creation. Only project_manager role.
func (h *ProjectHandler) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	var req createProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Nama proyek wajib diisi.", map[string]string{"name": "required"})
		return
	}

	var start, end *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		t, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format start_date harus YYYY-MM-DD.", map[string]string{"start_date": "invalid_date_format"})
			return
		}
		start = &t
	}
	if req.EndDate != nil && *req.EndDate != "" {
		t, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format end_date harus YYYY-MM-DD.", map[string]string{"end_date": "invalid_date_format"})
			return
		}
		end = &t
	}

	p, err := h.service.CreateProject(r.Context(), actor, project.CreateProjectParams{
		Name:        name,
		Description: req.Description,
		StartDate:   start,
		EndDate:     end,
	})
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusCreated, p)
}

// HandleGetProject returns project details by ID.
func (h *ProjectHandler) HandleGetProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	p, err := h.service.GetProject(r.Context(), actor, id)
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, p)
}

type updateProjectReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
}

// HandleUpdateProject updates project metadata. Only PM owner.
func (h *ProjectHandler) HandleUpdateProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	var req updateProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Nama proyek wajib diisi.", map[string]string{"name": "required"})
		return
	}

	status := db.ProjectStatus(req.Status)
	switch status {
	case db.ProjectStatusActive, db.ProjectStatusOnHold, db.ProjectStatusCompleted, db.ProjectStatusArchived:
	default:
		status = db.ProjectStatusActive
	}

	var start, end *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		t, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format start_date harus YYYY-MM-DD.", map[string]string{"start_date": "invalid_date_format"})
			return
		}
		start = &t
	}
	if req.EndDate != nil && *req.EndDate != "" {
		t, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format end_date harus YYYY-MM-DD.", map[string]string{"end_date": "invalid_date_format"})
			return
		}
		end = &t
	}

	p, err := h.service.UpdateProject(r.Context(), actor, id, project.UpdateProjectParams{
		Name:        name,
		Description: req.Description,
		Status:      status,
		StartDate:   start,
		EndDate:     end,
	})
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, p)
}

// HandleArchiveProject archives a project. Only PM owner.
func (h *ProjectHandler) HandleArchiveProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	p, err := h.service.ArchiveProject(r.Context(), actor, id)
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, p)
}

type transferOwnerReq struct {
	NewOwnerID string `json:"new_owner_id"`
}

// HandleTransferOwner transfers project ownership. Only admin role.
func (h *ProjectHandler) HandleTransferOwner(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	var req transferOwnerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	if _, err := uuid.Parse(req.NewOwnerID); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "new_owner_id harus berupa UUID yang valid.", map[string]string{"new_owner_id": "invalid_uuid"})
		return
	}

	p, err := h.service.TransferOwner(r.Context(), actor, id, req.NewOwnerID)
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, p)
}

// ─── Members ─────────────────────────────────────────────────────────────────

// HandleListMembers returns members of a project.
func (h *ProjectHandler) HandleListMembers(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	members, err := h.service.ListMembers(r.Context(), actor, id)
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, members)
}

type addMemberReq struct {
	UserID string `json:"user_id"`
}

// HandleAddMember adds a team leader to the project. Only PM owner.
func (h *ProjectHandler) HandleAddMember(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	var req addMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	if _, err := uuid.Parse(req.UserID); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "user_id harus berupa UUID yang valid.", map[string]string{"user_id": "invalid_uuid"})
		return
	}

	if err := h.service.AddMember(r.Context(), actor, id, req.UserID); err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusCreated, map[string]string{"message": "Member berhasil ditambahkan."})
}

// HandleRemoveMember removes a member from the project. Only PM owner.
func (h *ProjectHandler) HandleRemoveMember(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	userID := chi.URLParam(r, "userId")
	if _, err := uuid.Parse(userID); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "userId harus berupa UUID yang valid.", nil)
		return
	}

	if err := h.service.RemoveMember(r.Context(), actor, id, userID); err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Member berhasil dihapus."})
}

// ─── Columns ─────────────────────────────────────────────────────────────────

// HandleListColumns returns all columns for a project board.
func (h *ProjectHandler) HandleListColumns(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	cols, err := h.service.ListColumns(r.Context(), actor, id)
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, cols)
}

type createColumnReq struct {
	Name         string `json:"name"`
	Position     int32  `json:"position"`
	IsDoneColumn bool   `json:"is_done_column"`
	WipLimit     *int32 `json:"wip_limit"`
}

// HandleCreateColumn creates a new column for a project board. Only PM owner.
func (h *ProjectHandler) HandleCreateColumn(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
		return
	}

	var req createColumnReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Nama kolom wajib diisi.", map[string]string{"name": "required"})
		return
	}

	col, err := h.service.CreateColumn(r.Context(), actor, id, project.CreateColumnParams{
		Name:         name,
		Position:     req.Position,
		IsDoneColumn: req.IsDoneColumn,
		WipLimit:     req.WipLimit,
	})
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusCreated, col)
}

type updateColumnReq struct {
	Name     string `json:"name"`
	Position int32  `json:"position"`
	WipLimit *int32 `json:"wip_limit"`
}

// HandleUpdateColumn updates a column's name, position, or WIP limit. Only PM owner.
func (h *ProjectHandler) HandleUpdateColumn(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	colID := chi.URLParam(r, "columnId")
	if colID == "" {
		colID = chi.URLParam(r, "id")
	}
	if _, err := uuid.Parse(colID); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "ID kolom harus berupa UUID yang valid.", nil)
		return
	}

	var req updateColumnReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Nama kolom wajib diisi.", map[string]string{"name": "required"})
		return
	}

	col, err := h.service.UpdateColumn(r.Context(), actor, colID, project.UpdateColumnParams{
		Name:     name,
		Position: req.Position,
		WipLimit: req.WipLimit,
	})
	if err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, col)
}

// HandleDeleteColumn deletes a board column. Done column cannot be deleted.
func (h *ProjectHandler) HandleDeleteColumn(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	colID := chi.URLParam(r, "columnId")
	if colID == "" {
		colID = chi.URLParam(r, "id")
	}
	if _, err := uuid.Parse(colID); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "ID kolom harus berupa UUID yang valid.", nil)
		return
	}

	if err := h.service.DeleteColumn(r.Context(), actor, colID); err != nil {
		mapProjectError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Kolom berhasil dihapus."})
}
