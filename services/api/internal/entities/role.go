package entities

import "time"

// Permission represents a granular access permission.
type Permission string

const (
	PermDeploy        Permission = "deploy"
	PermViewLogs      Permission = "view_logs"
	PermManageDB      Permission = "manage_db"
	PermManageTeam    Permission = "manage_team"
	PermManageBilling Permission = "manage_billing"
	PermAdmin         Permission = "admin"
)

// AllPermissions returns all available permissions.
func AllPermissions() []Permission {
	return []Permission{PermDeploy, PermViewLogs, PermManageDB, PermManageTeam, PermManageBilling, PermAdmin}
}

// CustomRole is a user-defined role with specific permissions.
type CustomRole struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"` // owner/creator
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// RoleAssignment links a team member to a custom role.
type RoleAssignment struct {
	ID         string    `json:"id"`
	RoleID     string    `json:"role_id"`
	MemberID   string    `json:"member_id"` // the user being assigned
	AssignedBy string    `json:"assigned_by"`
	CreatedAt  time.Time `json:"created_at"`
}
