package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ppo-kanban/internal/db"
	"ppo-kanban/internal/task"
)

// TaskHandler handles HTTP requests for tasks, movements, and comments.
type TaskHandler struct {
	service *task.Service
}

// NewTaskHandler creates a TaskHandler.
func NewTaskHandler(service *task.Service) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) actor(r *http.Request) (task.ActorInfo, bool) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		return task.ActorInfo{}, false
	}
	return task.ActorInfo{
		ID:        session.User.ID,
		Role:      db.UserRole(session.User.Role),
		RequestID: middleware.GetReqID(r.Context()),
		IP:        r.RemoteAddr,
	}, true
}

func mapTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, task.ErrTaskNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Task tidak ditemukan.", nil)
	case errors.Is(err, task.ErrProjectNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Proyek tidak ditemukan.", nil)
	case errors.Is(err, task.ErrColumnNotFound):
		ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Kolom tidak ditemukan.", nil)
	case errors.Is(err, task.ErrForbidden):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki izin untuk melakukan aksi ini.", nil)
	case errors.Is(err, task.ErrProjectArchived):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Proyek sudah diarsipkan dan bersifat read-only.", nil)
	case errors.Is(err, task.ErrMakerCheckerDone):
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Hanya Project Manager yang dapat memindahkan task ke atau dari kolom Done.", nil)
	case errors.Is(err, task.ErrVersionConflict):
		ErrorJSON(w, http.StatusConflict, "VERSION_CONFLICT", "Task sudah diubah oleh pengguna lain. Muat ulang papan.", nil)
	case errors.Is(err, task.ErrWIPLimitExceeded):
		ErrorJSON(w, http.StatusConflict, "WIP_LIMIT_EXCEEDED", "WIP limit pada kolom tujuan telah tercapai.", nil)
	case errors.Is(err, task.ErrAssigneeNotMember):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Assignee harus merupakan member atau owner proyek.", nil)
	case errors.Is(err, task.ErrCommentBodyEmpty):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Komentar tidak boleh kosong.", nil)
	case errors.Is(err, task.ErrCommentBodyTooLong):
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Komentar maksimal 5000 karakter.", nil)
	default:
		ErrorJSON(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan pada server.", nil)
	}
}

// ─── Task Handlers ───────────────────────────────────────────────────────────

// HandleListTasks returns cross-project tasks filtered according to role permissions.
func (h *TaskHandler) HandleListTasks(w http.ResponseWriter, r *http.Request) {
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
	var assigneeID *string
	if val := strings.TrimSpace(q.Get("assignee_id")); val != "" {
		assigneeID = &val
	}
	var columnID *string
	if val := strings.TrimSpace(q.Get("column_id")); val != "" {
		columnID = &val
	}
	var isOverdue *bool
	if val := strings.TrimSpace(q.Get("overdue")); val == "true" {
		t := true
		isOverdue = &t
	}
	var mine *bool
	if val := strings.TrimSpace(q.Get("mine")); val == "true" {
		t := true
		mine = &t
	}

	limit := int32(20)
	if val := q.Get("limit"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	offset := int32(0)
	if val := q.Get("offset"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	tasks, total, err := h.service.ListTasks(r.Context(), actor, task.ListTasksFilter{
		ProjectID:  projectID,
		AssigneeID: assigneeID,
		ColumnID:   columnID,
		IsOverdue:  isOverdue,
		Mine:       mine,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HandleListBoardTasks returns all active tasks for a project board.
func (h *TaskHandler) HandleListBoardTasks(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	projectID := chi.URLParam(r, "id")
	tasks, err := h.service.ListBoardTasks(r.Context(), actor, projectID)
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, tasks)
}

type createTaskReq struct {
	ColumnID        string   `json:"column_id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	AssigneeID      *string  `json:"assignee_id"`
	DueDate         *string  `json:"due_date"`
	ProgressPercent int32    `json:"progress_percent"`
	Labels          []string `json:"labels"`
}

// HandleCreateTask creates a new task in a project.
func (h *TaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	projectID := chi.URLParam(r, "id")
	var req createTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Judul task wajib diisi.", map[string]string{"title": "required"})
		return
	}

	if strings.TrimSpace(req.ColumnID) == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Kolom tujuan wajib dipilih.", map[string]string{"column_id": "required"})
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.DueDate))
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format due_date harus YYYY-MM-DD.", nil)
			return
		}
		dueDate = &parsed
	}

	labels := req.Labels
	if labels == nil {
		labels = []string{}
	}

	created, err := h.service.CreateTask(r.Context(), actor, projectID, task.CreateTaskParams{
		ProjectID:       projectID,
		ColumnID:        req.ColumnID,
		Title:           req.Title,
		Description:     req.Description,
		Priority:        db.TaskPriority(req.Priority),
		AssigneeID:      req.AssigneeID,
		DueDate:         dueDate,
		ProgressPercent: req.ProgressPercent,
		Labels:          labels,
	})
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusCreated, created)
}

// HandleGetTask returns a single task by ID.
func (h *TaskHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	taskID := chi.URLParam(r, "id")
	t, err := h.service.GetTask(r.Context(), actor, taskID)
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, t)
}

type updateTaskReq struct {
	Version         int32    `json:"version"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	AssigneeID      *string  `json:"assignee_id"`
	DueDate         *string  `json:"due_date"`
	ProgressPercent int32    `json:"progress_percent"`
	Labels          []string `json:"labels"`
}

// HandleUpdateTask updates a task with optimistic locking.
func (h *TaskHandler) HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	taskID := chi.URLParam(r, "id")
	var req updateTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	if req.Version <= 0 {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Version wajib disertakan untuk pembaruan.", map[string]string{"version": "required"})
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Judul task wajib diisi.", map[string]string{"title": "required"})
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.DueDate))
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format due_date harus YYYY-MM-DD.", nil)
			return
		}
		dueDate = &parsed
	}

	labels := req.Labels
	if labels == nil {
		labels = []string{}
	}

	updated, err := h.service.UpdateTask(r.Context(), actor, taskID, task.UpdateTaskParams{
		Version:         req.Version,
		Title:           req.Title,
		Description:     req.Description,
		Priority:        db.TaskPriority(req.Priority),
		AssigneeID:      req.AssigneeID,
		DueDate:         dueDate,
		ProgressPercent: req.ProgressPercent,
		Labels:          labels,
	})
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, updated)
}

type moveTaskReq struct {
	ToColumnID string `json:"to_column_id"`
	Position   int32  `json:"position"`
	Version    int32  `json:"version"`
}

// HandleMoveTask moves a task to a different position and/or column.
func (h *TaskHandler) HandleMoveTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	taskID := chi.URLParam(r, "id")
	var req moveTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	if strings.TrimSpace(req.ToColumnID) == "" {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Kolom tujuan wajib diisi.", map[string]string{"to_column_id": "required"})
		return
	}

	if req.Position <= 0 {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Posisi harus berupa angka positif.", map[string]string{"position": "must be >= 1"})
		return
	}

	if req.Version <= 0 {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Version wajib disertakan.", map[string]string{"version": "required"})
		return
	}

	moved, err := h.service.MoveTask(r.Context(), actor, taskID, task.MoveTaskParams{
		ToColumnID: req.ToColumnID,
		Position:   req.Position,
		Version:    req.Version,
	})
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, moved)
}

// HandleDeleteTask soft-deletes a task (only PM owner).
func (h *TaskHandler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	taskID := chi.URLParam(r, "id")
	if err := h.service.DeleteTask(r.Context(), actor, taskID); err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{"success": true})
}

// ─── Comments Handlers ───────────────────────────────────────────────────────

type createCommentReq struct {
	Body string `json:"body"`
}

// HandleCreateComment adds a comment to a task.
func (h *TaskHandler) HandleCreateComment(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	taskID := chi.URLParam(r, "id")
	var req createCommentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "INVALID_INPUT", "Format request JSON tidak valid.", nil)
		return
	}

	comment, err := h.service.CreateComment(r.Context(), actor, taskID, req.Body)
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusCreated, comment)
}

// HandleListComments returns all comments for a task.
func (h *TaskHandler) HandleListComments(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	taskID := chi.URLParam(r, "id")
	comments, err := h.service.ListComments(r.Context(), actor, taskID)
	if err != nil {
		mapTaskError(w, err)
		return
	}

	JSON(w, http.StatusOK, comments)
}
