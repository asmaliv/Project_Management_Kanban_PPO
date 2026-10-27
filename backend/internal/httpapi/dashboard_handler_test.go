package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/finding"
	"ppo-kanban/internal/project"
	"ppo-kanban/internal/risk"
	"ppo-kanban/internal/task"
)

func setupDashboardHTTPServer(t *testing.T) (http.Handler, *httpFindingMockStore, *auth.Service, string) {
	t.Helper()
	secret := "test-secret-key-at-least-32-chars-long"
	store := newHTTPFindingMockStore()

	auditSvc := auditlog.NewService(nil, nil)
	authSvc := auth.NewService(store, secret, nil, nil)
	projSvc := project.NewService(nil, store, nil, nil)
	taskSvc := task.NewService(nil, store, nil, nil)
	riskSvc := risk.NewService(nil, store, nil, nil)
	findingSvc := finding.NewService(nil, store, nil, nil)

	router := NewRouter(RouterParams{
		Queries:        store,
		AuthService:    authSvc,
		AuditService:   auditSvc,
		ProjectService: projSvc,
		TaskService:    taskSvc,
		RiskService:    riskSvc,
		FindingService: findingSvc,
		SessionSecret:  secret,
		CookieSecure:   false,
	})

	return router, store, authSvc, secret
}

func TestDashboardHTTP_AllRoles(t *testing.T) {
	handler, store, authSvc, _ := setupDashboardHTTPServer(t)

	adminCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleAdmin)
	pmCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleProjectManager)
	tlCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleTeamLeader)
	auditCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleAudit)
	rmCookie, _ := createFindingUserSession(t, store, authSvc, db.UserRoleRiskManagement)

	t.Run("Anonim -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Admin Dashboard -> 200 with admin role data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		req.AddCookie(adminCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp.Data.(map[string]any)
		if data["role"] != "admin" {
			t.Errorf("expected role admin, got %v", data["role"])
		}
	})

	t.Run("PM Dashboard -> 200 with pm role data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		req.AddCookie(pmCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp.Data.(map[string]any)
		if data["role"] != "project_manager" {
			t.Errorf("expected role project_manager, got %v", data["role"])
		}
	})

	t.Run("Team Leader Dashboard -> 200 with team_leader role data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		req.AddCookie(tlCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp.Data.(map[string]any)
		if data["role"] != "team_leader" {
			t.Errorf("expected role team_leader, got %v", data["role"])
		}
		if _, ok := data["totals"]; !ok {
			t.Errorf("expected totals map for team leader")
		}
	})

	t.Run("Audit Dashboard -> 200 with audit role data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		req.AddCookie(auditCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp.Data.(map[string]any)
		if data["role"] != "audit" {
			t.Errorf("expected role audit, got %v", data["role"])
		}
	})

	t.Run("Risk Management Dashboard -> 200 with risk_management role data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		req.AddCookie(rmCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp ApiResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp.Data.(map[string]any)
		if data["role"] != "risk_management" {
			t.Errorf("expected role risk_management, got %v", data["role"])
		}
	})
}
