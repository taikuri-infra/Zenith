package keycloakclient

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// KeycloakAPI is an alias for ports.IdentityProvider. Kept for backward compatibility.
type KeycloakAPI = ports.IdentityProvider

// Compile-time checks.
var _ ports.IdentityProvider = (*Client)(nil)
var _ ports.IdentityProvider = (*MemoryKeycloakClient)(nil)

// Client implements KeycloakAPI using gocloak.
type Client struct {
	gc       *gocloak.GoCloak
	user     string
	password string
}

// NewClient creates a new Keycloak admin client.
func NewClient(url, adminUser, adminPassword string) *Client {
	gc := gocloak.NewClient(url)
	return &Client{
		gc:       gc,
		user:     adminUser,
		password: adminPassword,
	}
}

func (c *Client) token(ctx context.Context) (string, error) {
	tok, err := c.gc.LoginAdmin(ctx, c.user, c.password, "master")
	if err != nil {
		return "", fmt.Errorf("keycloak admin login: %w", err)
	}
	return tok.AccessToken, nil
}

// CreateRealm creates a new Keycloak realm for a tenant.
func (c *Client) CreateRealm(ctx context.Context, realmName, displayName string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}

	enabled := true
	_, err = c.gc.CreateRealm(ctx, token, gocloak.RealmRepresentation{
		Realm:       &realmName,
		DisplayName: &displayName,
		Enabled:     &enabled,
	})
	if err != nil {
		return fmt.Errorf("create realm %s: %w", realmName, err)
	}
	return nil
}

// DeleteRealm deletes a Keycloak realm.
func (c *Client) DeleteRealm(ctx context.Context, realmName string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	return c.gc.DeleteRealm(ctx, token, realmName)
}

// CreateClient creates an OIDC client in a realm and returns its secret.
func (c *Client) CreateClient(ctx context.Context, realmName, clientID, redirectURI string) (string, error) {
	token, err := c.token(ctx)
	if err != nil {
		return "", err
	}

	protocol := "openid-connect"
	accessType := "confidential"
	enabled := true
	id, err := c.gc.CreateClient(ctx, token, realmName, gocloak.Client{
		ClientID:                &clientID,
		Protocol:                &protocol,
		PublicClient:            gocloak.BoolP(false),
		DirectAccessGrantsEnabled: &enabled,
		Enabled:                 &enabled,
		RedirectURIs:            &[]string{redirectURI},
		ClientAuthenticatorType: &accessType,
	})
	if err != nil {
		return "", fmt.Errorf("create client %s in realm %s: %w", clientID, realmName, err)
	}

	cred, err := c.gc.GetClientSecret(ctx, token, realmName, id)
	if err != nil {
		return "", fmt.Errorf("get client secret for %s: %w", clientID, err)
	}

	if cred.Value == nil {
		return "", fmt.Errorf("client secret is nil for %s", clientID)
	}
	return *cred.Value, nil
}

// MemoryKeycloakClient is a no-op implementation for dev/test.
type MemoryKeycloakClient struct{}

func NewMemoryClient() *MemoryKeycloakClient { return &MemoryKeycloakClient{} }

func (m *MemoryKeycloakClient) CreateRealm(_ context.Context, _, _ string) error { return nil }
func (m *MemoryKeycloakClient) DeleteRealm(_ context.Context, _ string) error    { return nil }
func (m *MemoryKeycloakClient) CreateClient(_ context.Context, _, _, _ string) (string, error) {
	return "fake-client-secret", nil
}
