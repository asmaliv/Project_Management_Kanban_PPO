package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/finding"
	"ppo-kanban/internal/project"
	"ppo-kanban/internal/risk"
	"ppo-kanban/internal/task"
)

// DashboardHandler handles aggregated role-based dashboard metrics.
type DashboardHandler struct {
	queries        db.Querier
	authService    *auth.Service
	projectService *project.Service
	taskService    *task.Service
	riskService    *risk.Service
	findingService *finding.Service
	auditService   *auditlog.Service
}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler(
	queries db.Querier,
	authService *auth.Service,
	projectService *project.Service,
	taskService *task.Service,
	riskService *risk.Service,
	findingService *finding.Service,
	auditService *auditlog.Service,
) *DashboardHandler {
	return &DashboardHandler{
		queries:        queries,
		authService:    authService,
		projectService: projectService,
		taskService:    taskService,
		riskService:    riskService,
		findingService: findingService,
		auditService:   auditService,
	}
}

// HandleGetDashboard returns role-scoped statistics and widgets.
// GET /api/v1/dashboard
func (h *DashboardHandler) HandleGetDashboard(w http.ResponseWriter, r *http.Request) {
	session := GetSessionInfo(r.Context())
	if session == nil {
		ErrorJSON(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.", nil)
		return
	}

	role := db.UserRole(session.User.Role)
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)

	var projectID *string
	if val := strings.TrimSpace(r.URL.Query().Get("project_id")); val != "" {
		projectID = &val
	}

	switch role {
	case db.UserRoleAdmin:
		h.handleAdminDashboard(w, r)
	case db.UserRoleProjectManager:
		h.handlePMDashboard(w, r, session.User.ID, role, reqID, projectID)
	case db.UserRoleTeamLeader:
		h.handleTeamLeaderDashboard(w, r, session.User.ID, role, reqID, projectID)
	case db.UserRoleAudit:
		h.handleAuditDashboard(w, r, session.User.ID, role, reqID, projectID)
	case db.UserRoleRiskManagement:
		h.handleRiskDashboard(w, r, session.User.ID, role, reqID, projectID)
	default:
		ErrorJSON(w, http.StatusForbidden, "FORBIDDEN", "Role tidak dikenal.", nil)
	}
}

func (h *DashboardHandler) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var userStats map[string]any
	if h.queries != nil {
		stats, err := h.queries.GetUserStats(ctx)
		if err == nil {
			userStats = map[string]any{
				"total_users":     stats.TotalUsers,
				"admin_count":     stats.AdminCount,
				"pm_count":        stats.PmCount,
				"tl_count":        stats.TlCount,
				"audit_count":     stats.AuditCount,
				"risk_count":      stats.RiskCount,
				"locked_count":    stats.LockedCount,
				"inactive_count":  stats.InactiveCount,
			}
		}
	}

	var recentActivities any
	if h.auditService != nil {
		logs, err := h.auditService.Read(ctx, auditlog.ListFilters{
			Limit: 10,
		})
		if err == nil {
			recentActivities = logs
		}
	}

	JSON(w, http.StatusOK, map[string]any{
		"role":              "admin",
		"user_stats":        userStats,
		"recent_activities": recentActivities,
	})
}

func (h *DashboardHandler) handlePMDashboard(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	role db.UserRole,
	reqID string,
	projectID *string,
) {
	ctx := r.Context()
	taskActor := task.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}
	findingActor := finding.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}
	projectActor := project.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}

	// 1. Projects
	projects, _ := h.projectService.ListProjects(ctx, projectActor)

	// 2. Overdue tasks in PM's projects
	isOverdue := true
	overdueTasks, overdueTotal, _ := h.taskService.ListTasks(ctx, taskActor, task.ListTasksFilter{
		ProjectID: projectID,
		IsOverdue: &isOverdue,
		Limit:     20,
	})

	// 3. Tasks in review (waiting approval)
	allTasks, _, _ := h.taskService.ListTasks(ctx, taskActor, task.ListTasksFilter{
		ProjectID: projectID,
		Limit:     100,
	})
	var pendingApprovalTasks []task.TaskDTO
	for _, t := range allTasks {
		colLower := strings.ToLower(t.ColumnName)
		if strings.Contains(colLower, "review") || strings.Contains(colLower, "tinjau") {
			pendingApprovalTasks = append(pendingApprovalTasks, t)
		}
	}

	// 4. Open findings
	findingSummary, _ := h.findingService.GetFindingSummary(ctx, findingActor, projectID)

	activeProjectsCount := int64(0)
	for _, p := range projects {
		if string(p.Status) == "active" {
			activeProjectsCount++
		}
	}

	JSON(w, http.StatusOK, map[string]any{
		"role":     "project_manager",
		"projects": projects,
		"totals": map[string]any{
			"active_projects":  activeProjectsCount,
			"overdue_tasks":    overdueTotal,
			"pending_approval": len(pendingApprovalTasks),
			"open_findings":    findingSummary.OpenFindings,
		},
		"overdue_tasks":  overdueTasks,
		"pending_tasks":  pendingApprovalTasks,
		"finding_stats":  findingSummary,
	})
}

func (h *DashboardHandler) handleTeamLeaderDashboard(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	role db.UserRole,
	reqID string,
	projectID *string,
) {
	ctx := r.Context()
	taskActor := task.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}
	projectActor := project.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}

	// 1. Projects assigned to TL
	projects, _ := h.projectService.ListProjects(ctx, projectActor)

	// 2. All team tasks in assigned projects
	allTasks, totalTasks, _ := h.taskService.ListTasks(ctx, taskActor, task.ListTasksFilter{
		ProjectID: projectID,
		Limit:     100,
	})

	// 3. Overdue tasks
	isOverdue := true
	overdueTasks, overdueTotal, _ := h.taskService.ListTasks(ctx, taskActor, task.ListTasksFilter{
		ProjectID: projectID,
		IsOverdue: &isOverdue,
		Limit:     20,
	})

	// 4. Tasks assigned to caller (Tugas Saya)
	isMine := true
	myTasks, _, _ := h.taskService.ListTasks(ctx, taskActor, task.ListTasksFilter{
		ProjectID: projectID,
		Mine:      &isMine,
		Limit:     50,
	})

	// 5. Aggregate by column
	columnCountMap := make(map[string]int)
	columnOrder := []string{}
	for _, t := range allTasks {
		if _, exists := columnCountMap[t.ColumnName]; !exists {
			columnOrder = append(columnOrder, t.ColumnName)
		}
		columnCountMap[t.ColumnName]++
	}
	type columnStat struct {
		ColumnName string `json:"column_name"`
		Count      int    `json:"count"`
	}
	byColumn := make([]columnStat, len(columnOrder))
	for i, col := range columnOrder {
		byColumn[i] = columnStat{
			ColumnName: col,
			Count:      columnCountMap[col],
		}
	}

	// 6. Aggregate workload per member (active tasks, exclude Done)
	type memberWorkload struct {
		UserID      string `json:"user_id"`
		FullName    string `json:"full_name"`
		ActiveTasks int    `json:"active_tasks"`
	}
	workloadMap := make(map[string]*memberWorkload)
	pendingApprovalCount := 0

	for _, t := range allTasks {
		colLower := strings.ToLower(t.ColumnName)
		if strings.Contains(colLower, "review") || strings.Contains(colLower, "tinjau") {
			pendingApprovalCount++
		}

		if strings.Contains(colLower, "done") || strings.Contains(colLower, "selesai") {
			continue // exclude Done from active workload
		}

		if t.AssigneeID != nil && *t.AssigneeID != "" {
			name := "Unknown"
			if t.AssigneeName != nil {
				name = *t.AssigneeName
			}
			if _, exists := workloadMap[*t.AssigneeID]; !exists {
				workloadMap[*t.AssigneeID] = &memberWorkload{
					UserID:      *t.AssigneeID,
					FullName:    name,
					ActiveTasks: 0,
				}
			}
			workloadMap[*t.AssigneeID].ActiveTasks++
		}
	}

	workloadList := make([]memberWorkload, 0, len(workloadMap))
	for _, w := range workloadMap {
		workloadList = append(workloadList, *w)
	}

	JSON(w, http.StatusOK, map[string]any{
		"role":     "team_leader",
		"projects": projects,
		"totals": map[string]any{
			"total_tasks":      totalTasks,
			"overdue_tasks":    overdueTotal,
			"pending_approval": pendingApprovalCount,
		},
		"by_column": byColumn,
		"workload":  workloadList,
		"overdue":   overdueTasks,
		"my_tasks":  myTasks,
	})
}

func (h *DashboardHandler) handleAuditDashboard(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	role db.UserRole,
	reqID string,
	projectID *string,
) {
	ctx := r.Context()
	findingActor := finding.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}

	summary, _ := h.findingService.GetFindingSummary(ctx, findingActor, projectID)

	openStatus := "open"
	openFindings, _, _ := h.findingService.ListFindings(ctx, findingActor, finding.ListFindingsParams{
		ProjectID: projectID,
		Status:    &openStatus,
		Limit:     10,
	})

	var recentActivities any
	if h.auditService != nil {
		logs, err := h.auditService.Read(ctx, auditlog.ListFilters{
			Limit: 10,
		})
		if err == nil {
			recentActivities = logs
		}
	}

	JSON(w, http.StatusOK, map[string]any{
		"role":              "audit",
		"findings_summary":  summary,
		"open_findings":     openFindings,
		"recent_activities": recentActivities,
	})
}

func (h *DashboardHandler) handleRiskDashboard(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	role db.UserRole,
	reqID string,
	projectID *string,
) {
	ctx := r.Context()
	riskActor := risk.ActorInfo{ID: userID, Role: role, RequestID: reqID, IP: r.RemoteAddr}

	summary, _ := h.riskService.GetRisksSummary(ctx, riskActor, projectID)

	critLevel := "critical"
	critRisks, _, _ := h.riskService.ListRisks(ctx, riskActor, risk.ListRisksParams{
		ProjectID: projectID,
		Level:     &critLevel,
		Limit:     10,
	})

	JSON(w, http.StatusOK, map[string]any{
		"role":                 "risk_management",
		"risk_summary":         summary,
		"high_critical_risks":  critRisks,
	})
}
