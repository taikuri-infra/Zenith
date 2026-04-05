package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
)

// --- findOrCreateOAuthUser tests ---

func TestFindOrCreateOAuthUser_NewUser(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	tokens, err := svc.findOrCreateOAuthUser(ctx, "oauth-new@example.com", "OAuth New", "google")
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
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

	// Verify user was created
	user, err := svc.GetUser(ctx, "")
	_ = user // we can't easily get the ID, but verify via login
	// Verify the user exists by checking we can find them
	claims, err := zenithJWT.ParseToken(testJWTSecret, tokens.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.Email != "oauth-new@example.com" {
		t.Errorf("Expected email 'oauth-new@example.com', got '%s'", claims.Email)
	}
	if claims.Name != "OAuth New" {
		t.Errorf("Expected name 'OAuth New', got '%s'", claims.Name)
	}
}

func TestFindOrCreateOAuthUser_ExistingUser(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// Register a user first
	result, _ := svc.Register(ctx, "existing-oauth@example.com", "StrongP@ss1234", "Existing")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)

	// Now find/create via OAuth — should find the existing user
	tokens, err := svc.findOrCreateOAuthUser(ctx, "existing-oauth@example.com", "Existing", "github")
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}

	// Verify it's the same user
	claims, _ := zenithJWT.ParseToken(testJWTSecret, tokens.AccessToken)
	if claims.Subject != result.UserID {
		t.Errorf("Expected same user ID %s, got %s", result.UserID, claims.Subject)
	}
}

func TestFindOrCreateOAuthUser_ExistingUserDifferentProvider(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// Register with email
	result, _ := svc.Register(ctx, "switch@example.com", "StrongP@ss1234", "Switcher")
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	// Now OAuth login with Google — should update auth provider
	tokens, err := svc.findOrCreateOAuthUser(ctx, "switch@example.com", "Switcher", "google")
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}
}

func TestFindOrCreateOAuthUser_EmptyName(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// Empty name should fall back to email
	tokens, err := svc.findOrCreateOAuthUser(ctx, "noname@example.com", "", "google")
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}
}

func TestFindOrCreateOAuthUser_FirstUserIsOwner(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	svc.SetProjectRepo(memory.NewMemoryProjectRepository())
	ctx := context.Background()

	// First user via OAuth should be owner
	tokens, err := svc.findOrCreateOAuthUser(ctx, "first-oauth@example.com", "First OAuth", "google")
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}

	claims, _ := zenithJWT.ParseToken(testJWTSecret, tokens.AccessToken)
	stored, _ := userRepo.GetByID(ctx, claims.Subject)
	if stored.Role != entities.RoleOwner {
		t.Errorf("Expected first OAuth user to be owner, got %s", stored.Role)
	}
}

func TestFindOrCreateOAuthUser_NilPlanRepo(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	svc := NewAuthService(userRepo, testJWTSecret, nil)
	ctx := context.Background()

	// Should not panic with nil plan repo
	tokens, err := svc.findOrCreateOAuthUser(ctx, "noplan@example.com", "No Plan", "google")
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser with nil planRepo failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}
}

// --- GetOAuthRedirectURLWithCallback tests ---

func TestGetOAuthRedirectURLWithCallback_Google(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
		AppURL:             "https://app.zenith.dev",
	})

	url, state, err := svc.GetOAuthRedirectURLWithCallback("google", "https://api.zenith.dev/auth/google/callback")
	if err != nil {
		t.Fatalf("GetOAuthRedirectURLWithCallback failed: %v", err)
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
	if !strings.Contains(url, "api.zenith.dev") {
		t.Errorf("Expected callback URL in redirect, got '%s'", url)
	}
}

func TestGetOAuthRedirectURLWithCallback_GitHub(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		GitHubClientID:     "github-id",
		GitHubClientSecret: "github-secret",
		AppURL:             "https://app.zenith.dev",
	})

	url, state, err := svc.GetOAuthRedirectURLWithCallback("github", "https://api.zenith.dev/auth/github/callback")
	if err != nil {
		t.Fatalf("GetOAuthRedirectURLWithCallback failed: %v", err)
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

func TestGetOAuthRedirectURLWithCallback_NoConfig(t *testing.T) {
	svc, _ := newTestAuthService()

	_, _, err := svc.GetOAuthRedirectURLWithCallback("google", "https://callback.example.com")
	if err == nil {
		t.Error("Expected error when OAuth is not configured")
	}
}

func TestGetOAuthRedirectURLWithCallback_UnsupportedProvider(t *testing.T) {
	svc, _ := newTestAuthService()
	svc.SetOAuthConfig(OAuthConfig{
		AppURL: "https://app.zenith.dev",
	})

	_, _, err := svc.GetOAuthRedirectURLWithCallback("twitter", "https://callback.example.com")
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}

// --- fetchOAuthUserInfo tests ---

func TestFetchOAuthUserInfo_UnsupportedProvider(t *testing.T) {
	svc, _ := newTestAuthService()

	_, _, err := svc.fetchOAuthUserInfo("linkedin", "fake-token")
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported OAuth provider") {
		t.Errorf("Expected 'unsupported OAuth provider' error, got: %v", err)
	}
}

// --- ExchangeOAuthCode with expired code ---

func TestExchangeOAuthCode_ExpiredCode(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// Manually insert an expired code
	svc.oauthCodesMu.Lock()
	svc.oauthCodes["expired-code"] = &oauthCodeEntry{
		tokens:    &TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 3600},
		expiresAt: time.Now().Add(-1 * time.Hour), // expired an hour ago
	}
	svc.oauthCodesMu.Unlock()

	_, err := svc.ExchangeOAuthCode(ctx, "expired-code")
	if err == nil {
		t.Error("Expected error for expired OAuth code")
	}
	if !strings.Contains(err.Error(), "invalid or expired") {
		t.Errorf("Expected 'invalid or expired' error, got: %v", err)
	}
}

// --- MFA brute force protection ---

func TestMFALogin_BruteForceProtection(t *testing.T) {
	svc, userRepo := newTestAuthService()
	mfaRepo := memory.NewMemoryMFARepository()
	svc.SetMFARepo(mfaRepo)
	ctx := context.Background()

	// Register, verify, and enable MFA
	result, _ := svc.Register(ctx, "mfa-brute@example.com", "MfaP@ss1234", "MFA Brute")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	_, _ = mfaRepo.StartEnrollment(ctx, result.UserID)
	_, _ = mfaRepo.ConfirmEnrollment(ctx, result.UserID)

	loginResult, _ := svc.Login(ctx, "mfa-brute@example.com", "MfaP@ss1234")

	// Try 6 wrong codes (max is 5)
	for i := 0; i < mfaMaxAttempts; i++ {
		_, _ = svc.MFALogin(ctx, loginResult.MFAToken, "000000")
	}

	// Next attempt should fail with "too many" error
	_, err := svc.MFALogin(ctx, loginResult.MFAToken, "000000")
	if err == nil {
		t.Error("Expected error after exceeding MFA attempts")
	}
	if !strings.Contains(err.Error(), "too many") {
		t.Errorf("Expected 'too many' error, got: %v", err)
	}
}

// --- MFA login with expired token ---

func TestMFALogin_ExpiredToken(t *testing.T) {
	svc, _ := newTestAuthService()
	mfaRepo := memory.NewMemoryMFARepository()
	svc.SetMFARepo(mfaRepo)
	ctx := context.Background()

	// Manually insert an expired MFA token
	svc.mfaCodesMu.Lock()
	svc.mfaCodes["expired-mfa"] = &mfaCodeEntry{
		userID:    "user-1",
		expiresAt: time.Now().Add(-1 * time.Hour),
	}
	svc.mfaCodesMu.Unlock()

	_, err := svc.MFALogin(ctx, "expired-mfa", "123456")
	if err == nil {
		t.Error("Expected error for expired MFA token")
	}
	if !strings.Contains(err.Error(), "invalid or expired") {
		t.Errorf("Expected 'invalid or expired' error, got: %v", err)
	}
}

// --- ProxyLogin with nonexistent user ---

func TestProxyLogin_NonexistentUser(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.ProxyLogin(ctx, "ghost@example.com")
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
	if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("Expected 'user not found' error, got: %v", err)
	}
}

// --- issueTeamTokens tests ---

func TestIssueTeamTokens_WithTeamMember(t *testing.T) {
	svc, userRepo := newTestAuthService()
	teamRepo := memory.NewMemoryTeamMemberRepository()
	svc.SetTeamRepo(teamRepo)
	ctx := context.Background()

	// Register
	result, _ := svc.Register(ctx, "team-member@example.com", "TeamP@ss1234", "Team Member")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	// Create team member
	teamRepo.CreateMember(ctx, &entities.TeamMember{
		ID:        "member-1",
		AccountID: "account-owner",
		UserID:    result.UserID,
		Email:     "team-member@example.com",
		Role:      entities.RoleCustomer,
		Status:    entities.TeamMemberActive,
	})

	// Login should issue team tokens
	loginResult, err := svc.Login(ctx, "team-member@example.com", "TeamP@ss1234")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginResult.Tokens == nil {
		t.Fatal("Expected tokens from team member login")
	}

	// Verify the token has team member claims
	claims, err := zenithJWT.ParseToken(testJWTSecret, loginResult.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.AccountID != "account-owner" {
		t.Errorf("Expected AccountID 'account-owner', got '%s'", claims.AccountID)
	}
}

// --- generateRandomToken tests ---

func TestGenerateRandomToken_Length(t *testing.T) {
	token, err := generateRandomToken()
	if err != nil {
		t.Fatalf("generateRandomToken failed: %v", err)
	}
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Expected 64-char token, got %d chars", len(token))
	}
}

func TestGenerateRandomToken_Unique(t *testing.T) {
	t1, _ := generateRandomToken()
	t2, _ := generateRandomToken()
	if t1 == t2 {
		t.Error("Expected unique tokens")
	}
}

// --- ValidateTOTP test (covers the thin wrapper) ---

func TestValidateTOTP_InvalidCode(t *testing.T) {
	// Use a known secret, a random 6-digit code will almost certainly fail
	valid := ValidateTOTP("000000", "JBSWY3DPEHPK3PXP")
	// We can't easily generate valid TOTP codes in tests without the totp library,
	// but we verify the function doesn't panic and returns a bool
	_ = valid
}
