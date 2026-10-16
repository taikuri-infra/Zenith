package api

import (
	"errors"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/cli/internal/config"
)

// TokenSet is what a login or refresh returns.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// ErrSessionExpired is returned when the access token is rejected and the refresh token no longer works.
var ErrSessionExpired = errors.New("session expired — run zen login")

// ErrMFARequired is returned by LoginWithPassword when the account has MFA; call LoginMFA with the code.
type ErrMFARequired struct{ MFAToken string }

func (e *ErrMFARequired) Error() string { return "multi-factor code required" }

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type mfaRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	MFARequired  bool   `json:"mfa_required"`
	MFAToken     string `json:"mfa_token"`
}

func (r tokenResponse) set() TokenSet {
	return TokenSet{AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: time.Now().Add(time.Duration(r.ExpiresIn) * time.Second)}
}

// SetSession installs tokens from a saved config.
func (c *Client) SetSession(ts TokenSet) {
	c.Token, c.RefreshToken, c.ExpiresAt = ts.AccessToken, ts.RefreshToken, ts.ExpiresAt
}

func (c *Client) adopt(r tokenResponse) TokenSet {
	ts := r.set()
	c.SetSession(ts)
	if c.OnTokens != nil {
		c.OnTokens(ts)
	}
	return ts
}

// LoginWithPassword logs in with email and password. It returns *ErrMFARequired when a code is needed.
func (c *Client) LoginWithPassword(email, password string) (TokenSet, error) {
	var resp tokenResponse
	if _, err := c.doOnce("POST", "/api/v1/auth/login", loginRequest{email, password}, &resp); err != nil {
		return TokenSet{}, err
	}
	if resp.MFARequired {
		return TokenSet{}, &ErrMFARequired{MFAToken: resp.MFAToken}
	}
	if resp.AccessToken == "" {
		return TokenSet{}, fmt.Errorf("login returned no access token")
	}
	return c.adopt(resp), nil
}

// LoginMFA completes a login that required a code.
func (c *Client) LoginMFA(mfaToken, code string) (TokenSet, error) {
	var resp tokenResponse
	if _, err := c.doOnce("POST", "/api/v1/auth/login/mfa", mfaRequest{mfaToken, code}, &resp); err != nil {
		return TokenSet{}, err
	}
	return c.adopt(resp), nil
}

// Refresh exchanges the refresh token for new tokens.
func (c *Client) Refresh() (TokenSet, error) {
	if c.RefreshToken == "" {
		return TokenSet{}, ErrSessionExpired
	}
	var resp tokenResponse
	if _, err := c.doOnce("POST", "/api/v1/auth/refresh", refreshRequest{c.RefreshToken}, &resp); err != nil {
		return TokenSet{}, err
	}
	if resp.RefreshToken == "" {
		resp.RefreshToken = c.RefreshToken // server did not rotate
	}
	return c.adopt(resp), nil
}

// refreshIfExpiring renews the session when the access token expires within a minute. Errors are left to the request.
func (c *Client) refreshIfExpiring() {
	if c.RefreshToken == "" || c.ExpiresAt.IsZero() || time.Until(c.ExpiresAt) > time.Minute {
		return
	}
	_, _ = c.Refresh()
}

// Logout invalidates the access token server-side (best effort).
func (c *Client) Logout() error {
	_, err := c.doOnce("POST", "/api/v1/auth/logout", nil, nil)
	return err
}

// NewClientFromConfig builds a client that keeps the saved session fresh.
func NewClientFromConfig(cfg *config.Config) *Client {
	c := NewClient(cfg.APIEndpoint, cfg.Token)
	c.SetSession(TokenSet{AccessToken: cfg.Token, RefreshToken: cfg.RefreshToken, ExpiresAt: cfg.TokenExpiresAt})
	c.OnTokens = func(ts TokenSet) {
		cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = ts.AccessToken, ts.RefreshToken, ts.ExpiresAt
		_ = cfg.Save()
	}
	return c
}
