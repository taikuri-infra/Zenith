package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

// --- Full VerifyEmail flow ---

func TestVerifyEmail_FullFlow(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	svc.SetProjectRepo(memory.NewMemoryProjectRepository())
	ctx := context.Background()

	// Register user
	result, err := svc.Register(ctx, "verify-full@example.com", "StrongP@ss1234", "VerifyFull")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// The Register function internally creates a verification token.
	// We need to create a known token that we can use to verify.
	rawToken := "known-verification-token-1234567890abcdef"
	tokenHash := hashToken(rawToken)
	err = userRepo.CreateVerificationToken(ctx, result.UserID, tokenHash, time.Now().Add(VerificationTokenExpiry))
	if err != nil {
		t.Fatalf("CreateVerificationToken failed: %v", err)
	}

	// Verify using the raw token
	tokens, err := svc.VerifyEmail(ctx, rawToken)
	if err != nil {
		t.Fatalf("VerifyEmail failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}
	if tokens.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}

	// User should now be verified
	user, _ := svc.GetUser(ctx, result.UserID)
	if !user.EmailVerified {
		t.Error("Expected email to be verified after VerifyEmail")
	}
}

// --- VerifyEmail with expired token ---

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	svc.SetProjectRepo(memory.NewMemoryProjectRepository())
	ctx := context.Background()

	result, _ := svc.Register(ctx, "verify-expired@example.com", "StrongP@ss1234", "VerifyExpired")

	// Create an expired token
	rawToken := "expired-token-abcdef1234567890"
	tokenHash := hashToken(rawToken)
	err := userRepo.CreateVerificationToken(ctx, result.UserID, tokenHash, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("CreateVerificationToken failed: %v", err)
	}

	// Verification should fail with expired token
	_, err = svc.VerifyEmail(ctx, rawToken)
	if err == nil {
		t.Error("Expected error for expired verification token")
	}
}

// --- ResendVerification with email sender ---

func TestResendVerification_WithEmailSender(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	svc.SetProjectRepo(memory.NewMemoryProjectRepository())
	emailSender := &mockEmailSenderVerify{}
	svc.SetEmailSender(emailSender, "https://app.zenith.dev")
	ctx := context.Background()

	_, _ = svc.Register(ctx, "resend@example.com", "StrongP@ss1234", "Resend")

	err := svc.ResendVerification(ctx, "resend@example.com")
	if err != nil {
		t.Fatalf("ResendVerification failed: %v", err)
	}
	// The register already sends one verification email, resend sends another
	if emailSender.verifyCount < 2 {
		t.Errorf("Expected at least 2 verification emails sent, got %d", emailSender.verifyCount)
	}
}

// --- Login after verify ---

func TestLogin_AfterVerifyEmail(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	svc.SetProjectRepo(memory.NewMemoryProjectRepository())
	ctx := context.Background()

	result, _ := svc.Register(ctx, "login-verify@example.com", "StrongP@ss1234", "LoginVerify")

	// Manually verify and set provider
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	// Create a verification token for full flow
	rawToken := "full-flow-token-1234567890"
	tokenHash := hashToken(rawToken)
	_ = userRepo.CreateVerificationToken(ctx, result.UserID, tokenHash, time.Now().Add(VerificationTokenExpiry))

	// Verify
	tokens, err := svc.VerifyEmail(ctx, rawToken)
	if err != nil {
		t.Fatalf("VerifyEmail failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil tokens after verification")
	}
}

// --- Refresh with user not found ---

func TestRefresh_UserNotFound(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	ctx := context.Background()

	// Register user
	result, _ := svc.Register(ctx, "refresh-gone@example.com", "StrongP@ss1234", "Gone")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	// Login to get refresh token
	loginResult, _ := svc.Login(ctx, "refresh-gone@example.com", "StrongP@ss1234")

	// Delete the user from repo (simulating user deletion)
	// Memory repo doesn't have a delete, but we can test the path by creating
	// a fresh service with an empty repo after getting the token
	svc2 := NewAuthService(memory.NewMemoryUserRepository(), testJWTSecret, planRepo)

	// The refresh token contains the user ID, but the user doesn't exist in svc2
	_, err := svc2.Refresh(ctx, loginResult.Tokens.RefreshToken)
	if err == nil {
		t.Error("Expected error when user not found during refresh")
	}
}

type mockEmailSenderVerify struct {
	verifyCount int
}

func (m *mockEmailSenderVerify) SendVerificationEmail(_ context.Context, to, name, url string) error {
	m.verifyCount++
	return nil
}
func (m *mockEmailSenderVerify) SendTeamInviteEmail(_ context.Context, to, inviterName, teamName, inviteURL string) error {
	return nil
}
func (m *mockEmailSenderVerify) SendSupportTicketNotification(_ context.Context, to, ticketSubject, ticketURL string) error {
	return nil
}
func (m *mockEmailSenderVerify) SendSupportReplyNotification(_ context.Context, to, userName, ticketSubject, ticketURL string) error {
	return nil
}
func (m *mockEmailSenderVerify) SendGenericEmail(_ context.Context, to, subject, htmlBody string) error {
	return nil
}
