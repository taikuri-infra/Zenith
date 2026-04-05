package services

import (
	"context"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- SetTeamRepo tests ---

func TestSetTeamRepo(t *testing.T) {
	svc, _ := newTestAuthService()
	teamRepo := memory.NewMemoryTeamMemberRepository()
	svc.SetTeamRepo(teamRepo)
	// No panic means success
}

// --- SetOAuthConfig tests ---

func TestSetOAuthConfig(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
		GitHubClientID:     "github-id",
		GitHubClientSecret: "github-secret",
		AppURL:             "https://app.zenith.dev",
	})
	// No panic means success
}

// --- SetEmailSender tests ---

func TestSetEmailSender(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetEmailSender(nil, "https://app.zenith.dev")
	// No panic means success
}

// --- UpdateSignupSource tests ---

func TestUpdateSignupSource_UTMSource(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "utm@example.com", "StrongP@ss1234", "UTM User")

	// Should not panic even if the memory repo doesn't implement the updater interface
	svc.UpdateSignupSource(ctx, result.UserID, "google", "cpc", "spring", "", "", "https://google.com", "1.2.3.4")
}

func TestUpdateSignupSource_ReferralSource(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "ref@example.com", "StrongP@ss1234", "Ref User")

	// Empty UTM but with referrer URL
	svc.UpdateSignupSource(ctx, result.UserID, "", "", "", "", "", "https://blog.example.com", "1.2.3.4")
}

func TestUpdateSignupSource_DirectSource(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "direct@example.com", "StrongP@ss1234", "Direct User")

	// No UTM, no referrer
	svc.UpdateSignupSource(ctx, result.UserID, "", "", "", "", "", "", "1.2.3.4")
}

// --- ProcessReferralCode tests ---

func TestProcessReferralCode_EmptyCode(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// Empty code should be no-op
	svc.ProcessReferralCode(ctx, "user-1", "")
}

func TestProcessReferralCode_NonEmptyCode(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// Non-empty code but memory repo doesn't implement referralLookup
	svc.ProcessReferralCode(ctx, "user-1", "INVITE123")
}

// --- GenerateReferralCode tests ---

func TestGenerateReferralCode(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	code := svc.GenerateReferralCode(ctx, "user-1")
	if len(code) != 8 {
		t.Errorf("Expected 8-char code, got %d chars: '%s'", len(code), code)
	}
}

func TestGenerateReferralCode_Unique(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	code1 := svc.GenerateReferralCode(ctx, "user-1")
	code2 := svc.GenerateReferralCode(ctx, "user-2")
	if code1 == code2 {
		t.Error("Expected different codes for different users")
	}
}

// --- UpdateOnboarding tests ---

func TestUpdateOnboarding(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "onboard@example.com", "StrongP@ss1234", "Onboard User")

	// Should not panic even if memory repo doesn't implement the interface
	svc.UpdateOnboarding(ctx, result.UserID, 1, false)
	svc.UpdateOnboarding(ctx, result.UserID, 3, true)
}

// --- UpdateLastLogin tests ---

func TestUpdateLastLogin(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "lastlogin@example.com", "StrongP@ss1234", "LastLogin")

	// Should not panic even if memory repo doesn't implement the interface
	svc.UpdateLastLogin(ctx, result.UserID)
}

// --- GetUser tests ---

func TestGetUser_Exists(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "getuser@example.com", "StrongP@ss1234", "Get User")

	user, err := svc.GetUser(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Email != "getuser@example.com" {
		t.Errorf("Expected email 'getuser@example.com', got '%s'", user.Email)
	}
	if user.Name != "Get User" {
		t.Errorf("Expected name 'Get User', got '%s'", user.Name)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.GetUser(ctx, "nonexistent-user-id")
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
}

// --- generateShortCode tests ---

func TestGenerateShortCode_Length(t *testing.T) {
	lengths := []int{4, 8, 16, 32}
	for _, l := range lengths {
		code := generateShortCode(l)
		if len(code) != l {
			t.Errorf("generateShortCode(%d) returned %d chars: '%s'", l, len(code), code)
		}
	}
}

func TestGenerateShortCode_AlphaNumeric(t *testing.T) {
	code := generateShortCode(100)
	for _, c := range code {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Errorf("Expected alphanumeric, got '%c' in '%s'", c, code)
		}
	}
}

func TestGenerateShortCode_Unique(t *testing.T) {
	code1 := generateShortCode(16)
	code2 := generateShortCode(16)
	if code1 == code2 {
		t.Error("Expected different codes on consecutive calls")
	}
}

// --- hashToken tests ---

func TestHashToken_Consistent(t *testing.T) {
	h1 := hashToken("same-token")
	h2 := hashToken("same-token")
	if h1 != h2 {
		t.Error("Expected same hash for same input")
	}
}

func TestHashToken_Different(t *testing.T) {
	h1 := hashToken("token-a")
	h2 := hashToken("token-b")
	if h1 == h2 {
		t.Error("Expected different hashes for different inputs")
	}
}

func TestHashToken_NotEmpty(t *testing.T) {
	h := hashToken("test")
	if h == "" {
		t.Error("Expected non-empty hash")
	}
	if len(h) != 64 { // SHA-256 = 32 bytes = 64 hex chars
		t.Errorf("Expected 64-char hex hash, got %d chars", len(h))
	}
}

// --- Team member login (issueTokens with team repo) ---

func TestLogin_WithTeamRepo_NoTeamMember(t *testing.T) {
	svc, userRepo := newTestAuthService()
	teamRepo := memory.NewMemoryTeamMemberRepository()
	svc.SetTeamRepo(teamRepo)
	ctx := context.Background()

	result, _ := svc.Register(ctx, "team-no-member@example.com", "StrongP@ss1234", "NoTeam")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	loginResult, err := svc.Login(ctx, "team-no-member@example.com", "StrongP@ss1234")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginResult.Tokens == nil {
		t.Fatal("Expected tokens from login")
	}
	if loginResult.Tokens.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}
}

// --- VerifyEmail tests ---

func TestVerifyEmail_InvalidToken(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.VerifyEmail(ctx, "invalid-verification-token")
	if err == nil {
		t.Error("Expected error for invalid verification token")
	}
	if !strings.Contains(err.Error(), "invalid or expired") {
		t.Errorf("Expected 'invalid or expired' error, got: %v", err)
	}
}

// --- ResendVerification tests ---

func TestResendVerification_NonExistentEmail(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// Should not return error (to avoid revealing whether email exists)
	err := svc.ResendVerification(ctx, "ghost@example.com")
	if err != nil {
		t.Errorf("Expected nil error for non-existent email, got: %v", err)
	}
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "already-verified@example.com", "StrongP@ss1234", "Verified")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)

	// Should be a no-op
	err := svc.ResendVerification(ctx, "already-verified@example.com")
	if err != nil {
		t.Errorf("Expected nil error for already verified email, got: %v", err)
	}
}

func TestResendVerification_UnverifiedUser(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, "needs-verify@example.com", "StrongP@ss1234", "NeedsVerify")

	// Without email sender configured, this should store a new token but not fail
	err := svc.ResendVerification(ctx, "needs-verify@example.com")
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

// --- GetOAuthRedirectURL tests ---

func TestGetOAuthRedirectURL_NoConfig(t *testing.T) {
	svc, _ := newTestAuthService()

	_, _, err := svc.GetOAuthRedirectURL("google")
	if err == nil {
		t.Error("Expected error when OAuth is not configured")
	}
}

func TestGetOAuthRedirectURL_Google(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
		AppURL:             "https://app.zenith.dev",
	})

	url, state, err := svc.GetOAuthRedirectURL("google")
	if err != nil {
		t.Fatalf("GetOAuthRedirectURL failed: %v", err)
	}
	if url == "" {
		t.Error("Expected non-empty redirect URL")
	}
	if state == "" {
		t.Error("Expected non-empty state")
	}
	if !strings.Contains(url, "accounts.google.com") {
		t.Errorf("Expected Google OAuth URL, got '%s'", url)
	}
}

func TestGetOAuthRedirectURL_GitHub(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		GitHubClientID:     "github-id",
		GitHubClientSecret: "github-secret",
		AppURL:             "https://app.zenith.dev",
	})

	url, state, err := svc.GetOAuthRedirectURL("github")
	if err != nil {
		t.Fatalf("GetOAuthRedirectURL failed: %v", err)
	}
	if url == "" {
		t.Error("Expected non-empty redirect URL")
	}
	if state == "" {
		t.Error("Expected non-empty state")
	}
	if !strings.Contains(url, "github.com") {
		t.Errorf("Expected GitHub OAuth URL, got '%s'", url)
	}
}

func TestGetOAuthRedirectURL_UnsupportedProvider(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		AppURL: "https://app.zenith.dev",
	})

	_, _, err := svc.GetOAuthRedirectURL("facebook")
	if err == nil {
		t.Error("Expected error for unsupported OAuth provider")
	}
}

func TestGetOAuthRedirectURL_GoogleNotConfigured(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		// Google credentials missing
		AppURL: "https://app.zenith.dev",
	})

	_, _, err := svc.GetOAuthRedirectURL("google")
	if err == nil {
		t.Error("Expected error when Google OAuth is not configured")
	}
}

func TestGetOAuthRedirectURL_GitHubNotConfigured(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		// GitHub credentials missing
		AppURL: "https://app.zenith.dev",
	})

	_, _, err := svc.GetOAuthRedirectURL("github")
	if err == nil {
		t.Error("Expected error when GitHub OAuth is not configured")
	}
}

// --- Register with email sender ---

func TestRegister_WithEmailSender(t *testing.T) {
	svc, _ := newTestAuthService()
	emailSender := &mockEmailSender{}
	svc.SetEmailSender(emailSender, "https://app.zenith.dev")
	ctx := context.Background()

	result, err := svc.Register(ctx, "emailed@example.com", "StrongP@ss1234", "Emailed")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if result.UserID == "" {
		t.Error("Expected non-empty UserID")
	}
	// Email sender should have been called
	if emailSender.verifyCount == 0 {
		t.Error("Expected verification email to be sent")
	}
}

// mockEmailSender implements ports.EmailSender for testing.
type mockEmailSender struct {
	verifyCount int
}

func (m *mockEmailSender) SendVerificationEmail(_ context.Context, to, name, url string) error {
	m.verifyCount++
	return nil
}

func (m *mockEmailSender) SendTeamInviteEmail(_ context.Context, to, inviterName, teamName, inviteURL string) error {
	return nil
}

func (m *mockEmailSender) SendSupportTicketNotification(_ context.Context, to, ticketSubject, ticketURL string) error {
	return nil
}

func (m *mockEmailSender) SendSupportReplyNotification(_ context.Context, to, userName, ticketSubject, ticketURL string) error {
	return nil
}

func (m *mockEmailSender) SendGenericEmail(_ context.Context, to, subject, htmlBody string) error {
	return nil
}

// --- Register roles ---

func TestRegister_ThirdUserIsCustomer(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// First user = owner
	r1, _ := svc.Register(ctx, "owner@example.com", "StrongP@ss1234", "Owner")
	u1, _ := userRepo.GetByID(ctx, r1.UserID)
	if u1.Role != entities.RoleOwner {
		t.Errorf("Expected first user to be owner, got %s", u1.Role)
	}

	// Second user = customer
	r2, _ := svc.Register(ctx, "second@example.com", "StrongP@ss1234", "Second")
	u2, _ := userRepo.GetByID(ctx, r2.UserID)
	if u2.Role != entities.RoleCustomer {
		t.Errorf("Expected second user to be customer, got %s", u2.Role)
	}

	// Third user = customer
	r3, _ := svc.Register(ctx, "third@example.com", "StrongP@ss1234", "Third")
	u3, _ := userRepo.GetByID(ctx, r3.UserID)
	if u3.Role != entities.RoleCustomer {
		t.Errorf("Expected third user to be customer, got %s", u3.Role)
	}
}
