package models

import "testing"

func TestRoleCanPerform(t *testing.T) {
	tests := []struct {
		role   Role
		action string
		want   bool
	}{
		{RoleOwner, "read", true},
		{RoleOwner, "write", true},
		{RoleOwner, "deploy", true},
		{RoleOwner, "delete", true},
		{RoleOwner, "manage_members", true},
		{RoleOwner, "manage_billing", true},
		{RoleAdmin, "read", true},
		{RoleAdmin, "write", true},
		{RoleAdmin, "deploy", true},
		{RoleAdmin, "manage_billing", false},
		{RoleDeveloper, "read", true},
		{RoleDeveloper, "write", true},
		{RoleDeveloper, "deploy", true},
		{RoleDeveloper, "delete", false},
		{RoleDeveloper, "manage_members", false},
		{RoleViewer, "read", true},
		{RoleViewer, "write", false},
		{RoleViewer, "deploy", false},
		{Role("invalid"), "read", false},
	}

	for _, tt := range tests {
		got := tt.role.CanPerform(tt.action)
		if got != tt.want {
			t.Errorf("Role(%q).CanPerform(%q) = %v, want %v", tt.role, tt.action, got, tt.want)
		}
	}
}

func TestRoleIsValid(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleDeveloper, true},
		{RoleViewer, true},
		{Role("invalid"), false},
		{Role(""), false},
	}

	for _, tt := range tests {
		if got := tt.role.IsValid(); got != tt.want {
			t.Errorf("Role(%q).IsValid() = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestAPIKeyHasScope(t *testing.T) {
	key := &APIKey{
		Scopes: []string{"read", "write", "deploy"},
	}

	if !key.HasScope("read") {
		t.Error("Expected key to have 'read' scope")
	}
	if !key.HasScope("deploy") {
		t.Error("Expected key to have 'deploy' scope")
	}
	if key.HasScope("delete") {
		t.Error("Expected key to NOT have 'delete' scope")
	}

	wildcardKey := &APIKey{Scopes: []string{"*"}}
	if !wildcardKey.HasScope("anything") {
		t.Error("Wildcard key should have any scope")
	}
}
