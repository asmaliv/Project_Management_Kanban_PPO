package rbac

import "ppo-kanban/internal/db"

const (
	RoleAdmin          = string(db.UserRoleAdmin)
	RoleProjectManager = string(db.UserRoleProjectManager)
	RoleTeamLeader     = string(db.UserRoleTeamLeader)
	RoleAudit          = string(db.UserRoleAudit)
	RoleRiskManagement = string(db.UserRoleRiskManagement)
)

var validRoles = map[string]bool{
	RoleAdmin:          true,
	RoleProjectManager: true,
	RoleTeamLeader:     true,
	RoleAudit:          true,
	RoleRiskManagement: true,
}

// IsValidRole checks if the provided role string is one of the 5 canonical roles.
func IsValidRole(role string) bool {
	return validRoles[role]
}

// AllRoles returns all canonical role strings.
func AllRoles() []string {
	return []string{
		RoleAdmin,
		RoleProjectManager,
		RoleTeamLeader,
		RoleAudit,
		RoleRiskManagement,
	}
}
