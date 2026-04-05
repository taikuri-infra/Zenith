package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestTeamService() (*TeamMemberService, *memory.MemoryUserRepository, *memory.MemoryUserPlanRepository) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, testJWTSecret)
	return svc, userRepo, planRepo
}

func createTestOwner(t *testing.T, userRepo *memory.MemoryUserRepository) *entities.User {
	t.Helper()
	ctx := context.Background()
	user, err := userRepo.Create(ctx, "owner@example.com", "OwnerP@ss1234", "Owner", entities.RoleOwner)
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	_ = userRepo.SetEmailVerified(ctx, user.ID)
	return user
}

// --- InviteMember tests ---

func TestInviteMember_BasicFlow(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	// Set Pro plan (higher team member limit)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanPro)

	member, err := svc.InviteMember(ctx, owner.ID, "dev@example.com", entities.RoleDeveloper)
	if err != nil {
		t.Fatalf("InviteMember failed: %v", err)
	}
	if member.Email != "dev@example.com" {
		t.Errorf("Expected email dev@example.com, got %s", member.Email)
	}
	if member.Role != entities.RoleDeveloper {
		t.Errorf("Expected role developer, got %s", member.Role)
	}
	if member.Status != entities.TeamMemberPending {
		t.Errorf("Expected status pending, got %s", member.Status)
	}
	if member.AccountID != owner.ID {
		t.Errorf("Expected accountID %s, got %s", owner.ID, member.AccountID)
	}
}

func TestInviteMember_CannotInviteAsOwner(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanPro)

	_, err := svc.InviteMember(ctx, owner.ID, "bad@example.com", entities.RoleOwner)
	if err == nil {
		t.Error("Expected error when inviting as owner role")
	}
}

func TestInviteMember_CannotInviteAsCustomer(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanPro)

	_, err := svc.InviteMember(ctx, owner.ID, "bad@example.com", entities.RoleCustomer)
	if err == nil {
		t.Error("Expected error when inviting as customer role")
	}
}

func TestInviteMember_InvalidRole(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanPro)

	_, err := svc.InviteMember(ctx, owner.ID, "bad@example.com", entities.Role("hacker"))
	if err == nil {
		t.Error("Expected error for invalid role")
	}
}

func TestInviteMember_DuplicateEmail(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanPro)

	_, err := svc.InviteMember(ctx, owner.ID, "dup@example.com", entities.RoleDeveloper)
	if err != nil {
		t.Fatalf("First invite failed: %v", err)
	}

	_, err = svc.InviteMember(ctx, owner.ID, "dup@example.com", entities.RoleViewer)
	if err == nil {
		t.Error("Expected error for duplicate email invite")
	}
}

func TestInviteMember_PlanLimitReached(t *testing.T) {
	svc, userRepo, _ := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	// Free plan: max_team_members = 1, but owner counts as 1, so 0 invites allowed

	_, err := svc.InviteMember(ctx, owner.ID, "over@example.com", entities.RoleDeveloper)
	if err == nil {
		t.Error("Expected error when team member limit reached on free plan")
	}
}

func TestInviteMember_AllValidRoles(t *testing.T) {
	validRoles := []entities.Role{entities.RoleAdmin, entities.RoleDeveloper, entities.RoleViewer}

	for _, role := range validRoles {
		svc, userRepo, planRepo := newTestTeamService()
		ctx := context.Background()

		owner := createTestOwner(t, userRepo)
		planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam) // High member limit

		member, err := svc.InviteMember(ctx, owner.ID, "test@example.com", role)
		if err != nil {
			t.Errorf("InviteMember with role %s failed: %v", role, err)
			continue
		}
		if member.Role != role {
			t.Errorf("Expected role %s, got %s", role, member.Role)
		}
	}
}

// --- ListMembers tests ---

func TestListMembers_Empty(t *testing.T) {
	svc, userRepo, _ := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)

	members, err := svc.ListMembers(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("Expected 0 members, got %d", len(members))
	}
}

func TestListMembers_AfterInvites(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam)

	svc.InviteMember(ctx, owner.ID, "a@example.com", entities.RoleDeveloper)
	svc.InviteMember(ctx, owner.ID, "b@example.com", entities.RoleViewer)

	members, err := svc.ListMembers(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

// --- UpdateMemberRole tests ---

func TestUpdateMemberRole_ValidChange(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam)

	member, _ := svc.InviteMember(ctx, owner.ID, "role@example.com", entities.RoleDeveloper)

	err := svc.UpdateMemberRole(ctx, owner.ID, member.ID, entities.RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateMemberRole failed: %v", err)
	}

	members, _ := svc.ListMembers(ctx, owner.ID)
	for _, m := range members {
		if m.ID == member.ID {
			if m.Role != entities.RoleAdmin {
				t.Errorf("Expected role admin, got %s", m.Role)
			}
			return
		}
	}
	t.Error("Member not found after role update")
}

func TestUpdateMemberRole_CannotSetOwner(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam)

	member, _ := svc.InviteMember(ctx, owner.ID, "role@example.com", entities.RoleDeveloper)

	err := svc.UpdateMemberRole(ctx, owner.ID, member.ID, entities.RoleOwner)
	if err == nil {
		t.Error("Expected error when setting role to owner")
	}
}

func TestUpdateMemberRole_WrongAccount(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam)

	member, _ := svc.InviteMember(ctx, owner.ID, "role@example.com", entities.RoleDeveloper)

	err := svc.UpdateMemberRole(ctx, "wrong-account-id", member.ID, entities.RoleAdmin)
	if err == nil {
		t.Error("Expected error when updating member from wrong account")
	}
}

// --- RemoveMember tests ---

func TestRemoveMember_Success(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam)

	member, _ := svc.InviteMember(ctx, owner.ID, "rm@example.com", entities.RoleDeveloper)

	err := svc.RemoveMember(ctx, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	members, _ := svc.ListMembers(ctx, owner.ID)
	if len(members) != 0 {
		t.Errorf("Expected 0 members after removal, got %d", len(members))
	}
}

func TestRemoveMember_WrongAccount(t *testing.T) {
	svc, userRepo, planRepo := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)
	planRepo.SetUserPlan(ctx, owner.ID, entities.PlanTeam)

	member, _ := svc.InviteMember(ctx, owner.ID, "rm@example.com", entities.RoleDeveloper)

	err := svc.RemoveMember(ctx, "wrong-account-id", member.ID)
	if err == nil {
		t.Error("Expected error when removing member from wrong account")
	}
}

func TestRemoveMember_NonExistent(t *testing.T) {
	svc, userRepo, _ := newTestTeamService()
	ctx := context.Background()

	owner := createTestOwner(t, userRepo)

	err := svc.RemoveMember(ctx, owner.ID, "non-existent-id")
	if err == nil {
		t.Error("Expected error when removing non-existent member")
	}
}
