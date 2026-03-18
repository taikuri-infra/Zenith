package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
	"github.com/pquerna/otp/totp"
)

// ValidateTOTP validates a TOTP code against a secret.
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}

const (
	AccessTokenExpiry       = 1 * time.Hour
	RefreshTokenExpiry      = 7 * 24 * time.Hour
	VerificationTokenExpiry = 24 * time.Hour

	oauthCodeTTL = 5 * time.Minute
)

// TokenPair holds issued JWT tokens.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// RegisterResult is returned by Register.
// For email/password, Message is set and Tokens is nil (verification required).
// For OAuth, Tokens is set and Message is empty.
type RegisterResult struct {
	Tokens  *TokenPair
	Message string
	UserID  string // populated on successful registration
}

// OAuthConfig holds OAuth provider credentials.
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	AppURL             string // frontend URL for the callback redirect
}

// oauthCodeEntry stores a one-time code mapped to tokens.
type oauthCodeEntry struct {
	tokens    *TokenPair
	expiresAt time.Time
}

// mfaCodeEntry stores a pending MFA login challenge.
type mfaCodeEntry struct {
	userID    string
	expiresAt time.Time
	attempts  int // brute-force protection
}

const mfaTokenTTL = 5 * time.Minute
const mfaMaxAttempts = 5

// oauthHTTPClient is a dedicated HTTP client for OAuth provider requests
// with a 10-second timeout to prevent indefinite hangs.
var oauthHTTPClient = &http.Client{Timeout: 10 * time.Second}

// LoginResult represents the outcome of a login attempt.
// If MFARequired is true, the client must call MFALogin with the MFAToken and TOTP code.
type LoginResult struct {
	Tokens      *TokenPair // nil when MFA is required
	MFARequired bool
	MFAToken    string // short-lived token for MFA verification
}

// AuthService handles authentication business logic.
type AuthService struct {
	users       ports.UserRepository
	planRepo    ports.UserPlanRepository    // nil-safe — skips plan assignment when nil
	projectRepo ports.ProjectRepository     // nil-safe — skips default project creation when nil
	teamRepo    ports.TeamMemberRepository  // nil-safe — skips team member lookup when nil
	mfaRepo     ports.MFARepository         // nil-safe — skips MFA check when nil
	jwtSecret   string
	emailSender ports.EmailSender // nil = skip sending emails (dev mode)
	appURL      string            // frontend URL for verification links
	oauthCfg    *OAuthConfig

	oauthCodesMu sync.Mutex
	oauthCodes   map[string]*oauthCodeEntry

	mfaCodesMu sync.Mutex
	mfaCodes   map[string]*mfaCodeEntry // mfaToken → userID mapping
}

// NewAuthService creates a new AuthService.
func NewAuthService(users ports.UserRepository, jwtSecret string, planRepo ports.UserPlanRepository) *AuthService {
	return &AuthService{
		users:      users,
		jwtSecret:  jwtSecret,
		planRepo:   planRepo,
		oauthCodes: make(map[string]*oauthCodeEntry),
		mfaCodes:   make(map[string]*mfaCodeEntry),
	}
}

// SetProjectRepo configures the project repository for default project creation on registration.
func (s *AuthService) SetProjectRepo(repo ports.ProjectRepository) {
	s.projectRepo = repo
}

// SetMFARepo configures the MFA repository for login MFA challenge.
func (s *AuthService) SetMFARepo(repo ports.MFARepository) {
	s.mfaRepo = repo
}

// SetTeamRepo configures the team member repository for team login enrichment.
func (s *AuthService) SetTeamRepo(repo ports.TeamMemberRepository) {
	s.teamRepo = repo
}

// SetOAuthConfig configures OAuth provider credentials.
func (s *AuthService) SetOAuthConfig(cfg OAuthConfig) {
	s.oauthCfg = &cfg
}

// SetEmailSender configures the email sender for verification emails.
func (s *AuthService) SetEmailSender(sender ports.EmailSender, appURL string) {
	s.emailSender = sender
	s.appURL = appURL
}

// UpdateSignupSource stores UTM and signup source data on the user record.
// This is a fire-and-forget operation — errors are logged but not returned.
func (s *AuthService) UpdateSignupSource(ctx context.Context, userID, utmSource, utmMedium, utmCampaign, utmContent, utmTerm, referrerURL, signupIP string) {
	// Determine signup source from UTM
	source := "direct"
	if utmSource != "" {
		source = utmSource
	} else if referrerURL != "" {
		source = "referral"
	}
	type updater interface {
		UpdateSignupSource(ctx context.Context, userID, source, utmSource, utmMedium, utmCampaign, utmContent, utmTerm, referrerURL, signupIP string) error
	}
	if u, ok := s.users.(updater); ok {
		if err := u.UpdateSignupSource(ctx, userID, source, utmSource, utmMedium, utmCampaign, utmContent, utmTerm, referrerURL, signupIP); err != nil {
			slog.Warn("failed to update signup source", "user_id", userID, "error", err)
		}
	}
}

// ProcessReferralCode handles referral code on signup.
func (s *AuthService) ProcessReferralCode(ctx context.Context, userID, referralCode string) {
	if referralCode == "" {
		return
	}
	type referralLookup interface {
		GetByReferralCode(ctx context.Context, code string) (*ports.StoredUser, error)
		SetReferredBy(ctx context.Context, userID, referrerID string) error
	}
	if u, ok := s.users.(referralLookup); ok {
		referrer, err := u.GetByReferralCode(ctx, referralCode)
		if err != nil || referrer == nil {
			return
		}
		_ = u.SetReferredBy(ctx, userID, referrer.ID)
	}
}

// GenerateReferralCode creates and sets a unique referral code for the user.
func (s *AuthService) GenerateReferralCode(ctx context.Context, userID string) string {
	code := generateShortCode(8)
	type codeSetter interface {
		SetReferralCode(ctx context.Context, userID, code string) error
	}
	if u, ok := s.users.(codeSetter); ok {
		_ = u.SetReferralCode(ctx, userID, code)
	}
	return code
}

// UpdateOnboarding updates onboarding progress for a user.
func (s *AuthService) UpdateOnboarding(ctx context.Context, userID string, step int, completed bool) {
	type onboardingUpdater interface {
		UpdateOnboarding(ctx context.Context, userID string, step int, completed bool) error
	}
	if u, ok := s.users.(onboardingUpdater); ok {
		_ = u.UpdateOnboarding(ctx, userID, step, completed)
	}
}

// GetUser returns the user entity by ID.
func (s *AuthService) GetUser(ctx context.Context, userID string) (*entities.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &u.User, nil
}

// UpdateLastLogin updates the last login timestamp.
func (s *AuthService) UpdateLastLogin(ctx context.Context, userID string) {
	type lastLoginUpdater interface {
		UpdateLastLogin(ctx context.Context, userID string) error
	}
	if u, ok := s.users.(lastLoginUpdater); ok {
		_ = u.UpdateLastLogin(ctx, userID)
	}
}

// generateShortCode creates a random alphanumeric code of the given length.
func generateShortCode(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}

// Login validates credentials and returns a LoginResult.
// If MFA is enabled, MFARequired=true and MFAToken is set. Client must call MFALogin next.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil || !s.users.CheckPassword(user, password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Block login if email is not verified (email/password users only)
	if user.AuthProvider == "email" && !user.EmailVerified {
		return nil, fmt.Errorf("please verify your email before logging in")
	}

	// Check if MFA is enabled for this user
	if s.mfaRepo != nil {
		enrollment, err := s.mfaRepo.GetEnrollment(ctx, user.ID)
		if err == nil && enrollment.Status == entities.MFAStatusEnabled {
			// MFA required — issue a short-lived MFA token
			mfaToken, err := generateRandomToken()
			if err != nil {
				return nil, fmt.Errorf("failed to generate MFA token")
			}

			s.mfaCodesMu.Lock()
			s.mfaCodes[mfaToken] = &mfaCodeEntry{
				userID:    user.ID,
				expiresAt: time.Now().Add(mfaTokenTTL),
			}
			s.mfaCodesMu.Unlock()

			return &LoginResult{MFARequired: true, MFAToken: mfaToken}, nil
		}
	}

	tokens, err := s.issueTokens(ctx, &user.User)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Tokens: tokens}, nil
}

// MFALogin completes a login that requires MFA verification.
func (s *AuthService) MFALogin(ctx context.Context, mfaToken, code string) (*TokenPair, error) {
	s.mfaCodesMu.Lock()
	entry, exists := s.mfaCodes[mfaToken]
	if !exists || time.Now().After(entry.expiresAt) {
		if exists {
			delete(s.mfaCodes, mfaToken)
		}
		s.mfaCodesMu.Unlock()
		return nil, fmt.Errorf("invalid or expired MFA token")
	}

	// Brute-force protection: max attempts per token
	entry.attempts++
	if entry.attempts > mfaMaxAttempts {
		delete(s.mfaCodes, mfaToken)
		s.mfaCodesMu.Unlock()
		return nil, fmt.Errorf("too many MFA attempts, please login again")
	}
	s.mfaCodesMu.Unlock()

	// Get user
	user, err := s.users.GetByID(ctx, entry.userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get MFA enrollment
	enrollment, err := s.mfaRepo.GetEnrollment(ctx, entry.userID)
	if err != nil || enrollment.Status != entities.MFAStatusEnabled {
		return nil, fmt.Errorf("MFA is not enabled")
	}

	// Try TOTP code first (6-digit)
	valid := false
	if len(code) == 6 {
		valid = ValidateTOTP(code, enrollment.Secret)
	}
	// Try backup code if TOTP failed
	if !valid {
		used, err := s.mfaRepo.UseBackupCode(ctx, entry.userID, code)
		if err == nil && used {
			valid = true
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid MFA code")
	}

	// Success — remove the token
	s.mfaCodesMu.Lock()
	delete(s.mfaCodes, mfaToken)
	s.mfaCodesMu.Unlock()

	return s.issueTokens(ctx, &user.User)
}

// Register creates a new user and sends a verification email.
// Returns a message (not tokens) for email/password registration.
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*RegisterResult, error) {
	role := entities.RoleCustomer
	count, err := s.users.Count(ctx)
	if err == nil && count == 0 {
		role = entities.RoleOwner
	}

	user, err := s.users.Create(ctx, email, password, name, role)
	if err != nil {
		return nil, err
	}

	// Auto-assign free plan for new customers
	if s.planRepo != nil {
		_, _ = s.planRepo.SetUserPlan(ctx, user.ID, entities.PlanFree)
	}

	// Create default project
	if s.projectRepo != nil {
		_, _ = s.projectRepo.CreateProject(ctx, user.ID, "Default", "default", "")
	}

	// Generate verification token
	rawToken, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}
	tokenHash := hashToken(rawToken)

	if err := s.users.CreateVerificationToken(ctx, user.ID, tokenHash, time.Now().Add(VerificationTokenExpiry)); err != nil {
		return nil, fmt.Errorf("failed to store verification token: %w", err)
	}

	// Send verification email (skip if no email sender configured)
	if s.emailSender != nil && s.appURL != "" {
		verificationURL := s.appURL + "/verify-email?token=" + rawToken
		if err := s.emailSender.SendVerificationEmail(ctx, email, name, verificationURL); err != nil {
			// Log but don't fail — user can resend
			slog.Warn("failed to send verification email", "email", email, "error", err)
		}
	}

	return &RegisterResult{
		UserID:  user.ID,
		Message: "Please check your email to verify your account",
	}, nil
}

// VerifyEmail validates a verification token and returns a token pair.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) (*TokenPair, error) {
	tokenHash := hashToken(token)

	userID, err := s.users.GetVerificationToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired verification token")
	}

	if err := s.users.SetEmailVerified(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	// Clean up tokens
	_ = s.users.DeleteVerificationTokens(ctx, userID)

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueTokens(ctx, &user.User)
}

// ResendVerification generates a new verification token and sends the email.
func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists
		return nil
	}

	if user.EmailVerified {
		return nil // Already verified, no-op
	}

	// Delete old tokens
	_ = s.users.DeleteVerificationTokens(ctx, user.ID)

	// Generate new token
	rawToken, err := generateRandomToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}
	tokenHash := hashToken(rawToken)

	if err := s.users.CreateVerificationToken(ctx, user.ID, tokenHash, time.Now().Add(VerificationTokenExpiry)); err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	if s.emailSender != nil && s.appURL != "" {
		verificationURL := s.appURL + "/verify-email?token=" + rawToken
		if err := s.emailSender.SendVerificationEmail(ctx, email, user.Name, verificationURL); err != nil {
			return fmt.Errorf("failed to send verification email: %w", err)
		}
	}

	return nil
}

// Refresh validates a refresh token and returns a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := zenithJWT.ParseTokenWithType(s.jwtSecret, refreshToken, zenithJWT.TokenTypeRefresh)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	user, err := s.users.GetByID(ctx, claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueTokens(ctx, &user.User)
}

// GetOAuthRedirectURL builds the provider authorization URL and returns a random state.
func (s *AuthService) GetOAuthRedirectURL(provider string) (string, string, error) {
	cfg, err := s.oauth2Config(provider)
	if err != nil {
		return "", "", err
	}

	state, err := generateRandomToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	url := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, state, nil
}

// HandleOAuthCallback exchanges the authorization code, fetches user info,
// creates/finds the user, and returns a one-time code for token exchange.
func (s *AuthService) HandleOAuthCallback(ctx context.Context, provider, code string) (string, error) {
	cfg, err := s.oauth2Config(provider)
	if err != nil {
		return "", err
	}

	// Exchange code for access token
	oauthToken, err := cfg.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	// Fetch user info from provider
	email, name, err := s.fetchOAuthUserInfo(provider, oauthToken.AccessToken)
	if err != nil {
		return "", err
	}

	// Find or create user
	tokens, err := s.findOrCreateOAuthUser(ctx, email, name, provider)
	if err != nil {
		return "", err
	}

	// Store tokens with a one-time code
	oneTimeCode, err := generateRandomToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate one-time code: %w", err)
	}

	s.oauthCodesMu.Lock()
	s.oauthCodes[oneTimeCode] = &oauthCodeEntry{
		tokens:    tokens,
		expiresAt: time.Now().Add(oauthCodeTTL),
	}
	s.oauthCodesMu.Unlock()

	return oneTimeCode, nil
}

// ExchangeOAuthCode looks up the one-time code and returns the stored tokens.
func (s *AuthService) ExchangeOAuthCode(_ context.Context, code string) (*TokenPair, error) {
	s.oauthCodesMu.Lock()
	entry, ok := s.oauthCodes[code]
	if ok {
		delete(s.oauthCodes, code)
	}
	s.oauthCodesMu.Unlock()

	if !ok {
		return nil, fmt.Errorf("invalid or expired code")
	}

	if time.Now().After(entry.expiresAt) {
		return nil, fmt.Errorf("invalid or expired code")
	}

	return entry.tokens, nil
}

// oauth2Config returns the oauth2.Config for the given provider.
func (s *AuthService) oauth2Config(provider string) (*oauth2.Config, error) {
	if s.oauthCfg == nil {
		return nil, fmt.Errorf("OAuth is not configured")
	}

	appURL := s.oauthCfg.AppURL
	// Derive the API base URL from the app URL for the redirect URI.
	// The callback URL is on the API, not the frontend.
	// We use the APP_URL's scheme and swap the hostname prefix.
	// e.g. https://stage.freezenith.com -> the redirect is registered on the API domain.
	// However, the redirect_uri must match what's registered in the provider console.
	// The handler constructs the redirect URI from the request's own host.

	switch provider {
	case "google":
		if s.oauthCfg.GoogleClientID == "" || s.oauthCfg.GoogleClientSecret == "" {
			return nil, fmt.Errorf("Google OAuth is not configured")
		}
		return &oauth2.Config{
			ClientID:     s.oauthCfg.GoogleClientID,
			ClientSecret: s.oauthCfg.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
			// RedirectURL is set dynamically in the handler based on the request host
			RedirectURL: appURL, // placeholder, overridden by handler
		}, nil

	case "github":
		if s.oauthCfg.GitHubClientID == "" || s.oauthCfg.GitHubClientSecret == "" {
			return nil, fmt.Errorf("GitHub OAuth is not configured")
		}
		return &oauth2.Config{
			ClientID:     s.oauthCfg.GitHubClientID,
			ClientSecret: s.oauthCfg.GitHubClientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"user:email", "read:user"},
			RedirectURL:  appURL, // placeholder, overridden by handler
		}, nil

	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// GetOAuthRedirectURLWithCallback builds the redirect URL using the given callback URL.
func (s *AuthService) GetOAuthRedirectURLWithCallback(provider, callbackURL string) (string, string, error) {
	cfg, err := s.oauth2Config(provider)
	if err != nil {
		return "", "", err
	}

	cfg.RedirectURL = callbackURL

	state, err := generateRandomToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	url := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, state, nil
}

// HandleOAuthCallbackWithURL is like HandleOAuthCallback but uses a specific redirect URL.
func (s *AuthService) HandleOAuthCallbackWithURL(ctx context.Context, provider, code, callbackURL string) (string, error) {
	cfg, err := s.oauth2Config(provider)
	if err != nil {
		return "", err
	}
	cfg.RedirectURL = callbackURL

	oauthToken, err := cfg.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	email, name, err := s.fetchOAuthUserInfo(provider, oauthToken.AccessToken)
	if err != nil {
		return "", err
	}

	tokens, err := s.findOrCreateOAuthUser(ctx, email, name, provider)
	if err != nil {
		return "", err
	}

	oneTimeCode, err := generateRandomToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate one-time code: %w", err)
	}

	s.oauthCodesMu.Lock()
	s.oauthCodes[oneTimeCode] = &oauthCodeEntry{
		tokens:    tokens,
		expiresAt: time.Now().Add(oauthCodeTTL),
	}
	s.oauthCodesMu.Unlock()

	return oneTimeCode, nil
}

// fetchOAuthUserInfo fetches the user's email and name from the provider's API.
func (s *AuthService) fetchOAuthUserInfo(provider, accessToken string) (email, name string, err error) {
	switch provider {
	case "google":
		return s.fetchGoogleUserInfo(accessToken)
	case "github":
		return s.fetchGitHubUserInfo(accessToken)
	default:
		return "", "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

func (s *AuthService) fetchGoogleUserInfo(accessToken string) (string, string, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Google userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch Google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Google userinfo returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read Google userinfo response: %w", err)
	}

	var info struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", "", fmt.Errorf("failed to parse Google userinfo: %w", err)
	}

	if info.Email == "" || !info.VerifiedEmail {
		return "", "", fmt.Errorf("Google account email not verified")
	}

	return info.Email, info.Name, nil
}

func (s *AuthService) fetchGitHubUserInfo(accessToken string) (string, string, error) {
	// Fetch user profile
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GitHub user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub user API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read GitHub user response: %w", err)
	}

	var user struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", "", fmt.Errorf("failed to parse GitHub user: %w", err)
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}

	// If email is not public, fetch from /user/emails
	email := user.Email
	if email == "" {
		email, err = s.fetchGitHubPrimaryEmail(accessToken)
		if err != nil {
			return "", "", err
		}
	}

	return email, name, nil
}

func (s *AuthService) fetchGitHubPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub emails request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch GitHub emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub emails API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub emails response: %w", err)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("failed to parse GitHub emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified primary email found on GitHub account")
}

// findOrCreateOAuthUser finds an existing user or creates one.
func (s *AuthService) findOrCreateOAuthUser(ctx context.Context, email, name, provider string) (*TokenPair, error) {
	// Check if user exists
	existing, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		// Existing user — update auth provider if needed
		if existing.AuthProvider != provider {
			_ = s.users.SetAuthProvider(ctx, existing.ID, provider)
		}
		if !existing.EmailVerified {
			_ = s.users.SetEmailVerified(ctx, existing.ID)
		}
		return s.issueTokens(ctx, &existing.User)
	}

	// Create new user with random password (OAuth users don't use passwords)
	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random password")
	}
	randomPassword := hex.EncodeToString(randBytes)

	if name == "" {
		name = email
	}

	role := entities.RoleCustomer
	count, err := s.users.Count(ctx)
	if err == nil && count == 0 {
		role = entities.RoleOwner
	}

	user, err := s.users.Create(ctx, email, randomPassword, name, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Auto-assign free plan for new customers
	if s.planRepo != nil {
		_, _ = s.planRepo.SetUserPlan(ctx, user.ID, entities.PlanFree)
	}

	// Create default project
	if s.projectRepo != nil {
		_, _ = s.projectRepo.CreateProject(ctx, user.ID, "Default", "default", "")
	}

	// OAuth-verified users are auto-verified
	_ = s.users.SetEmailVerified(ctx, user.ID)
	_ = s.users.SetAuthProvider(ctx, user.ID, provider)

	user.EmailVerified = true
	user.AuthProvider = provider

	return s.issueTokens(ctx, user)
}

// ProxyLogin authenticates a user by trusted email header (Cloudflare Access).
// Only allows owner/admin roles. Used when the service is behind Zero Trust.
func (s *AuthService) ProxyLogin(ctx context.Context, email string) (*TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.Role != entities.RoleOwner && user.Role != entities.RoleAdmin {
		return nil, fmt.Errorf("insufficient permissions")
	}

	return s.issueTokens(ctx, &user.User)
}

func (s *AuthService) issueTokens(ctx context.Context, user *entities.User) (*TokenPair, error) {
	// Check if user is a team member — issue tokens with AccountID if so
	if s.teamRepo != nil {
		member, err := s.teamRepo.GetMemberByUserID(ctx, user.ID)
		if err == nil && member != nil {
			return s.issueTeamTokens(user, member)
		}
	}

	accessToken, err := zenithJWT.GenerateToken(s.jwtSecret, user, AccessTokenExpiry, zenithJWT.TokenTypeAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token")
	}

	refreshToken, err := zenithJWT.GenerateToken(s.jwtSecret, user, RefreshTokenExpiry, zenithJWT.TokenTypeRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
	}, nil
}

func (s *AuthService) issueTeamTokens(user *entities.User, member *entities.TeamMember) (*TokenPair, error) {
	overrides := zenithJWT.TeamMemberOverrides{
		AccountID: member.AccountID,
		MemberID:  member.ID,
		Role:      member.Role,
	}

	accessToken, err := zenithJWT.GenerateTeamMemberToken(s.jwtSecret, user, AccessTokenExpiry, overrides, zenithJWT.TokenTypeAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token")
	}

	refreshToken, err := zenithJWT.GenerateTeamMemberToken(s.jwtSecret, user, RefreshTokenExpiry, overrides, zenithJWT.TokenTypeRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
	}, nil
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
