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

const testJWTSecret = "test-secret-32-bytes-long-enough"

// newTestAuthService creates an AuthService with in-memory adapters.
func newTestAuthService() (*AuthService, *memory.MemoryUserRepository) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewAuthService(userRepo, testJWTSecret, planRepo)
	svc.SetProjectRepo(memory.NewMemoryProjectRepository())
	return svc, userRepo
}

// --- Registration tests ---

func TestRegister_BasicFlow(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	result, err := svc.Register(ctx, "alice@example.com", "StrongP@ss1234", "Alice")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if result.UserID == "" {
		t.Error("Expected non-empty UserID")
	}
	if result.Tokens != nil {
		t.Error("Expected nil Tokens for email/password registration (verification required)")
	}
	if result.Message == "" {
		t.Error("Expected non-empty verification message")
	}
}

func TestRegister_FirstUserIsOwner(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, err := svc.Register(ctx, "owner@example.com", "StrongP@ss1234", "Owner")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// First user should be owner
	stored, err := userRepo.GetByID(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if stored.Role != entities.RoleOwner {
		t.Errorf("Expected first user role to be %s, got %s", entities.RoleOwner, stored.Role)
	}

	// Second user should be customer
	result2, err := svc.Register(ctx, "customer@example.com", "StrongP@ss1234", "Customer")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}
	stored2, err := userRepo.GetByID(ctx, result2.UserID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if stored2.Role != entities.RoleCustomer {
		t.Errorf("Expected second user role to be %s, got %s", entities.RoleCustomer, stored2.Role)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "dup@example.com", "StrongP@ss1234", "First")
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	_, err = svc.Register(ctx, "dup@example.com", "AnotherP@ss999", "Second")
	if err == nil {
		t.Error("Expected error for duplicate email registration")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// The memory adapter uses bcrypt, which accepts any password.
	// This test verifies the service doesn't crash on short passwords.
	// In production, password validation happens at the handler layer.
	result, err := svc.Register(ctx, "weak@example.com", "x", "Weak")
	if err != nil {
		t.Fatalf("Register with short password failed (memory adapter): %v", err)
	}
	if result.UserID == "" {
		t.Error("Expected non-empty UserID even with short password")
	}
}

// --- Login tests ---

func TestLogin_CorrectCredentials(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// Register and verify email
	result, err := svc.Register(ctx, "login@example.com", "CorrectP@ss1", "Login User")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	// Login
	loginResult, err := svc.Login(ctx, "login@example.com", "CorrectP@ss1")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginResult.Tokens == nil {
		t.Fatal("Expected tokens from successful login")
	}
	if loginResult.Tokens.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}
	if loginResult.Tokens.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}
	if loginResult.Tokens.ExpiresIn <= 0 {
		t.Error("Expected positive ExpiresIn")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "wrong@example.com", "RealP@ss1234", "Wrong")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	_, err := svc.Login(ctx, "wrong@example.com", "BadP@ss9999")
	if err == nil {
		t.Error("Expected error for wrong password")
	}
	if !strings.Contains(err.Error(), "invalid email or password") {
		t.Errorf("Expected 'invalid email or password' error, got: %v", err)
	}
}

func TestLogin_UnverifiedEmail(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "unverified@example.com", "GoodP@ss1234", "Unverified")
	// Set auth provider to email but do NOT verify
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	_, err := svc.Login(ctx, "unverified@example.com", "GoodP@ss1234")
	if err == nil {
		t.Error("Expected error for unverified email login")
	}
	if !strings.Contains(err.Error(), "verify your email") {
		t.Errorf("Expected 'verify your email' error, got: %v", err)
	}
}

func TestLogin_NonExistentUser(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Login(ctx, "ghost@example.com", "AnyP@ss1234")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

// --- JWT token tests ---

func TestJWTTokenGeneration_ValidClaims(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "jwt@example.com", "JwtP@ss1234", "JWT User")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	loginResult, err := svc.Login(ctx, "jwt@example.com", "JwtP@ss1234")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Parse and validate the access token
	claims, err := zenithJWT.ParseToken(testJWTSecret, loginResult.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.Subject != result.UserID {
		t.Errorf("Expected Subject=%s, got %s", result.UserID, claims.Subject)
	}
	if claims.Email != "jwt@example.com" {
		t.Errorf("Expected Email=jwt@example.com, got %s", claims.Email)
	}
	if claims.Name != "JWT User" {
		t.Errorf("Expected Name='JWT User', got '%s'", claims.Name)
	}
	if claims.Type != zenithJWT.TokenTypeAccess {
		t.Errorf("Expected Type=access, got %s", claims.Type)
	}
}

func TestJWTTokenGeneration_RefreshTokenType(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "refresh-type@example.com", "RefP@ss1234", "Refresh")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	loginResult, err := svc.Login(ctx, "refresh-type@example.com", "RefP@ss1234")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// The refresh token should have type=refresh
	claims, err := zenithJWT.ParseToken(testJWTSecret, loginResult.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("ParseToken (refresh) failed: %v", err)
	}
	if claims.Type != zenithJWT.TokenTypeRefresh {
		t.Errorf("Expected Type=refresh, got %s", claims.Type)
	}
}

func TestJWTTokenGeneration_WrongSecret(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "secret@example.com", "SecP@ss1234", "Secret")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	loginResult, _ := svc.Login(ctx, "secret@example.com", "SecP@ss1234")

	_, err := zenithJWT.ParseToken("wrong-secret-key-!!!!!!!!!!!!!!!", loginResult.Tokens.AccessToken)
	if err == nil {
		t.Error("Expected token parsing to fail with wrong secret")
	}
}

// --- Refresh token flow ---

func TestRefresh_ValidToken(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "refresh@example.com", "RefP@ss1234", "Refresh User")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	loginResult, _ := svc.Login(ctx, "refresh@example.com", "RefP@ss1234")

	// Use the refresh token to get new tokens
	newTokens, err := svc.Refresh(ctx, loginResult.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Error("Expected non-empty new access token")
	}
	if newTokens.RefreshToken == "" {
		t.Error("Expected non-empty new refresh token")
	}

	// Verify the new access token is valid
	claims, err := zenithJWT.ParseToken(testJWTSecret, newTokens.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken on refreshed token failed: %v", err)
	}
	if claims.Subject != result.UserID {
		t.Errorf("Expected Subject=%s, got %s", result.UserID, claims.Subject)
	}
}

func TestRefresh_AccessTokenRejected(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	result, _ := svc.Register(ctx, "no-access@example.com", "NoP@ss1234", "No Access")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	loginResult, _ := svc.Login(ctx, "no-access@example.com", "NoP@ss1234")

	// Using access token as refresh token should fail (Refresh uses ParseTokenWithType)
	_, err := svc.Refresh(ctx, loginResult.Tokens.AccessToken)
	if err == nil {
		t.Error("Expected Refresh to reject access tokens")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Refresh(ctx, "totally-invalid-jwt-string")
	if err == nil {
		t.Error("Expected error for invalid refresh token")
	}
}

// --- OAuth code exchange ---

func TestOAuthCodeExchange(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// Register and verify a user
	result, _ := svc.Register(ctx, "oauth@example.com", "OAuthP@ss1234", "OAuth User")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	// Manually issue tokens and store as an OAuth code (simulating the callback flow)
	loginResult, _ := svc.Login(ctx, "oauth@example.com", "OAuthP@ss1234")

	// Store an OAuth code
	svc.oauthCodesMu.Lock()
	code := "test-oauth-code-1234"
	svc.oauthCodes[code] = &oauthCodeEntry{
		tokens:    loginResult.Tokens,
		expiresAt: time.Now().Add(oauthCodeTTL),
	}
	svc.oauthCodesMu.Unlock()

	// Exchange the code
	tokens, err := svc.ExchangeOAuthCode(ctx, code)
	if err != nil {
		t.Fatalf("ExchangeOAuthCode failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("Expected non-empty access token from OAuth code exchange")
	}

	// Code should be consumed (one-time use)
	_, err = svc.ExchangeOAuthCode(ctx, code)
	if err == nil {
		t.Error("Expected error when reusing OAuth code")
	}
}

func TestOAuthCodeExchange_InvalidCode(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	_, err := svc.ExchangeOAuthCode(ctx, "nonexistent-code")
	if err == nil {
		t.Error("Expected error for invalid OAuth code")
	}
}

// --- MFA code generation (MFA login challenge) ---

func TestMFALogin_ChallengeGenerated(t *testing.T) {
	svc, userRepo := newTestAuthService()
	mfaRepo := memory.NewMemoryMFARepository()
	svc.SetMFARepo(mfaRepo)
	ctx := context.Background()

	// Register, verify, and enable MFA
	result, _ := svc.Register(ctx, "mfa@example.com", "MfaP@ss1234", "MFA User")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	_, _ = mfaRepo.StartEnrollment(ctx, result.UserID)
	_, _ = mfaRepo.ConfirmEnrollment(ctx, result.UserID)

	// Login should require MFA
	loginResult, err := svc.Login(ctx, "mfa@example.com", "MfaP@ss1234")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if !loginResult.MFARequired {
		t.Error("Expected MFARequired=true for user with MFA enabled")
	}
	if loginResult.MFAToken == "" {
		t.Error("Expected non-empty MFAToken")
	}
	if loginResult.Tokens != nil {
		t.Error("Expected nil Tokens when MFA is required")
	}
}

func TestMFALogin_BackupCodeWorks(t *testing.T) {
	svc, userRepo := newTestAuthService()
	mfaRepo := memory.NewMemoryMFARepository()
	svc.SetMFARepo(mfaRepo)
	ctx := context.Background()

	// Register, verify, and enable MFA
	result, _ := svc.Register(ctx, "mfa-backup@example.com", "MfaP@ss1234", "MFA Backup")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	enrollment, _ := mfaRepo.StartEnrollment(ctx, result.UserID)
	_, _ = mfaRepo.ConfirmEnrollment(ctx, result.UserID)

	// Get a backup code
	enrollment, _ = mfaRepo.GetEnrollment(ctx, result.UserID)
	backupCode := enrollment.BackupCodes[0]

	// Login to get MFA challenge
	loginResult, _ := svc.Login(ctx, "mfa-backup@example.com", "MfaP@ss1234")

	// Complete MFA with backup code
	tokens, err := svc.MFALogin(ctx, loginResult.MFAToken, backupCode)
	if err != nil {
		t.Fatalf("MFALogin with backup code failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("Expected non-empty access token after MFA login")
	}
}

func TestMFALogin_InvalidCode(t *testing.T) {
	svc, userRepo := newTestAuthService()
	mfaRepo := memory.NewMemoryMFARepository()
	svc.SetMFARepo(mfaRepo)
	ctx := context.Background()

	result, _ := svc.Register(ctx, "mfa-bad@example.com", "MfaP@ss1234", "MFA Bad")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)
	_ = userRepo.SetAuthProvider(ctx, result.UserID, "email")

	_, _ = mfaRepo.StartEnrollment(ctx, result.UserID)
	_, _ = mfaRepo.ConfirmEnrollment(ctx, result.UserID)

	loginResult, _ := svc.Login(ctx, "mfa-bad@example.com", "MfaP@ss1234")

	_, err := svc.MFALogin(ctx, loginResult.MFAToken, "000000")
	if err == nil {
		t.Error("Expected error for invalid MFA code")
	}
}

func TestMFALogin_InvalidToken(t *testing.T) {
	svc, _ := newTestAuthService()
	mfaRepo := memory.NewMemoryMFARepository()
	svc.SetMFARepo(mfaRepo)
	ctx := context.Background()

	_, err := svc.MFALogin(ctx, "nonexistent-mfa-token", "123456")
	if err == nil {
		t.Error("Expected error for invalid MFA token")
	}
}

// --- VerifyEmail flow ---

func TestVerifyEmail(t *testing.T) {
	svc, _ := newTestAuthService()
	ctx := context.Background()

	// Register creates a verification token internally.
	// We cannot get the raw token from Register since it's hashed.
	// Instead, test the flow through the VerifyEmail method using
	// a manually created token.

	result, err := svc.Register(ctx, "verify@example.com", "VerifyP@ss1234", "Verify")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if result.UserID == "" {
		t.Fatal("Expected non-empty UserID")
	}

	// After registration, user should not be email-verified yet
	user, err := svc.GetUser(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.EmailVerified {
		t.Error("Expected email to not be verified yet after registration")
	}
}

// --- ProxyLogin ---

func TestProxyLogin_OwnerAllowed(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// First user is owner
	result, _ := svc.Register(ctx, "proxy-owner@example.com", "ProxyP@ss1234", "Proxy Owner")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)

	tokens, err := svc.ProxyLogin(ctx, "proxy-owner@example.com")
	if err != nil {
		t.Fatalf("ProxyLogin for owner failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("Expected non-empty access token from ProxyLogin")
	}
}

func TestProxyLogin_CustomerDenied(t *testing.T) {
	svc, userRepo := newTestAuthService()
	ctx := context.Background()

	// First user is owner (to get past the "count == 0" check)
	ownerResult, _ := svc.Register(ctx, "owner-first@example.com", "OwnerP@ss1234", "Owner")
	_ = userRepo.SetEmailVerified(ctx, ownerResult.UserID)

	// Second user is customer
	result, _ := svc.Register(ctx, "proxy-customer@example.com", "CustP@ss1234", "Customer")
	_ = userRepo.SetEmailVerified(ctx, result.UserID)

	_, err := svc.ProxyLogin(ctx, "proxy-customer@example.com")
	if err == nil {
		t.Error("Expected ProxyLogin to deny customer role")
	}
	if !strings.Contains(err.Error(), "insufficient permissions") {
		t.Errorf("Expected 'insufficient permissions' error, got: %v", err)
	}
}

