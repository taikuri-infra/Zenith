package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- AcceptInvite tests ---

func TestAcceptInvite_InvalidToken(t *testing.T) {
	svc, _, _ := newTestTeamService()
	ctx := context.Background()

	_, err := svc.AcceptInvite(ctx, "bogus-token", "dev@example.com", "P@ssword123", "Developer")
	if err == nil {
		t.Error("Expected error for invalid invite token")
	}
}

func TestAcceptInvite_Success(t *testing.T) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, testJWTSecret)
	ctx := context.Background()

	// Create a pending team member directly with a known token hash
	rawToken := "known-test-token-12345678"
	tokenHash := hashInviteToken(rawToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	member := &entities.TeamMember{
		ID:              "member-1",
		AccountID:       "owner-1",
		Email:           "accept@example.com",
		Role:            entities.RoleDeveloper,
		Status:          entities.TeamMemberPending,
		InviteTokenHash: tokenHash,
		InviteExpiresAt: &expiresAt,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	teamRepo.CreateMember(ctx, member)

	tokens, err := svc.AcceptInvite(ctx, rawToken, "accept@example.com", "S3cureP@ss1234", "Dev User")
	if err != nil {
		t.Fatalf("AcceptInvite failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil token pair")
	}
	if tokens.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}
	if tokens.ExpiresIn <= 0 {
		t.Error("Expected positive ExpiresIn")
	}
}

func TestAcceptInvite_AlreadyAccepted(t *testing.T) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, testJWTSecret)
	ctx := context.Background()

	rawToken := "accept-once-token"
	tokenHash := hashInviteToken(rawToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	member := &entities.TeamMember{
		ID:              "member-2",
		AccountID:       "owner-1",
		Email:           "once@example.com",
		Role:            entities.RoleDeveloper,
		Status:          entities.TeamMemberPending,
		InviteTokenHash: tokenHash,
		InviteExpiresAt: &expiresAt,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	teamRepo.CreateMember(ctx, member)

	// First accept
	_, err := svc.AcceptInvite(ctx, rawToken, "once@example.com", "P@ss12345", "User")
	if err != nil {
		t.Fatalf("First AcceptInvite failed: %v", err)
	}

	// Second accept should fail — member is no longer pending, and hash is cleared
	_, err = svc.AcceptInvite(ctx, rawToken, "once@example.com", "P@ss12345", "User")
	if err == nil {
		t.Error("Expected error for already-accepted invite")
	}
}

func TestAcceptInvite_ExpiredToken(t *testing.T) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, testJWTSecret)
	ctx := context.Background()

	rawToken := "expired-token"
	tokenHash := hashInviteToken(rawToken)
	expired := time.Now().Add(-1 * time.Hour) // already expired

	member := &entities.TeamMember{
		ID:              "member-3",
		AccountID:       "owner-1",
		Email:           "expired@example.com",
		Role:            entities.RoleDeveloper,
		Status:          entities.TeamMemberPending,
		InviteTokenHash: tokenHash,
		InviteExpiresAt: &expired,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	teamRepo.CreateMember(ctx, member)

	_, err := svc.AcceptInvite(ctx, rawToken, "expired@example.com", "P@ss12345", "User")
	if err == nil {
		t.Error("Expected error for expired invite token")
	}
}

func TestAcceptInvite_ExistingUser(t *testing.T) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, testJWTSecret)
	ctx := context.Background()

	// Create user first
	userRepo.Create(ctx, "existing@example.com", "ExistingP@ss1", "Existing User", entities.RoleCustomer)

	rawToken := "existing-user-token"
	tokenHash := hashInviteToken(rawToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	member := &entities.TeamMember{
		ID:              "member-4",
		AccountID:       "owner-1",
		Email:           "existing@example.com",
		Role:            entities.RoleViewer,
		Status:          entities.TeamMemberPending,
		InviteTokenHash: tokenHash,
		InviteExpiresAt: &expiresAt,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	teamRepo.CreateMember(ctx, member)

	tokens, err := svc.AcceptInvite(ctx, rawToken, "existing@example.com", "ExistingP@ss1", "Existing User")
	if err != nil {
		t.Fatalf("AcceptInvite with existing user failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil token pair")
	}
}

// --- GetMemberForLogin tests ---

func TestGetMemberForLogin_NoMember(t *testing.T) {
	svc, _, _ := newTestTeamService()
	ctx := context.Background()

	_, err := svc.GetMemberForLogin(ctx, "nonexistent-user")
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
}

func TestGetMemberForLogin_ActiveMember(t *testing.T) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, testJWTSecret)
	ctx := context.Background()

	// Set up an active member with a user_id
	member := &entities.TeamMember{
		ID:        "member-active",
		AccountID: "owner-1",
		UserID:    "user-42",
		Email:     "active@example.com",
		Role:      entities.RoleDeveloper,
		Status:    entities.TeamMemberActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	teamRepo.CreateMember(ctx, member)

	found, err := svc.GetMemberForLogin(ctx, "user-42")
	if err != nil {
		t.Fatalf("GetMemberForLogin failed: %v", err)
	}
	if found.ID != "member-active" {
		t.Errorf("Expected member ID 'member-active', got '%s'", found.ID)
	}
	if found.Role != entities.RoleDeveloper {
		t.Errorf("Expected role developer, got %s", found.Role)
	}
}

// --- hashInviteToken tests ---

func TestHashInviteToken_Deterministic(t *testing.T) {
	h1 := hashInviteToken("test-token")
	h2 := hashInviteToken("test-token")
	if h1 != h2 {
		t.Error("Expected same hash for same input")
	}
}

func TestHashInviteToken_DifferentInputs(t *testing.T) {
	h1 := hashInviteToken("token-a")
	h2 := hashInviteToken("token-b")
	if h1 == h2 {
		t.Error("Expected different hashes for different inputs")
	}
}

func TestHashInviteToken_Length(t *testing.T) {
	h := hashInviteToken("test")
	// SHA-256 = 32 bytes = 64 hex chars
	if len(h) != 64 {
		t.Errorf("Expected 64-char hex hash, got %d chars", len(h))
	}
}

// --- generateInviteToken tests ---

func TestGenerateInviteToken(t *testing.T) {
	token, err := generateInviteToken()
	if err != nil {
		t.Fatalf("generateInviteToken failed: %v", err)
	}
	// 32 bytes = 64 hex chars
	if len(token) != 64 {
		t.Errorf("Expected 64-char token, got %d chars", len(token))
	}
}

func TestGenerateInviteToken_Unique(t *testing.T) {
	t1, _ := generateInviteToken()
	t2, _ := generateInviteToken()
	if t1 == t2 {
		t.Error("Expected unique tokens")
	}
}
