package keycloakclient

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// CreateUser creates a user in a Keycloak realm and sets their password.
func (c *Client) CreateUser(ctx context.Context, realmName, email, password, firstName, lastName string) (string, error) {
	token, err := c.token(ctx)
	if err != nil {
		return "", err
	}

	enabled := true
	userID, err := c.gc.CreateUser(ctx, token, realmName, gocloak.User{
		Username:  &email,
		Email:     &email,
		FirstName: &firstName,
		LastName:  &lastName,
		Enabled:   &enabled,
	})
	if err != nil {
		return "", fmt.Errorf("create user in realm %s: %w", realmName, err)
	}

	if err := c.gc.SetPassword(ctx, token, userID, realmName, password, false); err != nil {
		return "", fmt.Errorf("set password for user %s: %w", userID, err)
	}

	return userID, nil
}

// GetUser retrieves a user by ID from a Keycloak realm.
func (c *Client) GetUser(ctx context.Context, realmName, userID string) (*ports.IdentityUser, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}

	u, err := c.gc.GetUserByID(ctx, token, realmName, userID)
	if err != nil {
		return nil, fmt.Errorf("get user %s in realm %s: %w", userID, realmName, err)
	}

	return toIdentityUser(u), nil
}

// ListUsers returns a paginated list of users and the total count.
func (c *Client) ListUsers(ctx context.Context, realmName string, first, max int) ([]ports.IdentityUser, int, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, 0, err
	}

	users, err := c.gc.GetUsers(ctx, token, realmName, gocloak.GetUsersParams{
		First: &first,
		Max:   &max,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list users in realm %s: %w", realmName, err)
	}

	total, err := c.gc.GetUserCount(ctx, token, realmName, gocloak.GetUsersParams{})
	if err != nil {
		return nil, 0, fmt.Errorf("count users in realm %s: %w", realmName, err)
	}

	result := make([]ports.IdentityUser, len(users))
	for i, u := range users {
		result[i] = *toIdentityUser(u)
	}

	return result, total, nil
}

// DeleteUser deletes a user from a Keycloak realm.
func (c *Client) DeleteUser(ctx context.Context, realmName, userID string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	return c.gc.DeleteUser(ctx, token, realmName, userID)
}

// DisableUser disables a user in a Keycloak realm.
func (c *Client) DisableUser(ctx context.Context, realmName, userID string) error {
	return c.setUserEnabled(ctx, realmName, userID, false)
}

// EnableUser enables a user in a Keycloak realm.
func (c *Client) EnableUser(ctx context.Context, realmName, userID string) error {
	return c.setUserEnabled(ctx, realmName, userID, true)
}

// CountUsers returns the total number of users in a realm.
func (c *Client) CountUsers(ctx context.Context, realmName string) (int, error) {
	token, err := c.token(ctx)
	if err != nil {
		return 0, err
	}
	return c.gc.GetUserCount(ctx, token, realmName, gocloak.GetUsersParams{})
}

// CreateRole creates a realm-level role.
func (c *Client) CreateRole(ctx context.Context, realmName, roleName, description string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	_, err = c.gc.CreateRealmRole(ctx, token, realmName, gocloak.Role{
		Name:        &roleName,
		Description: &description,
	})
	if err != nil {
		return fmt.Errorf("create role %s in realm %s: %w", roleName, realmName, err)
	}
	return nil
}

// ListRoles returns all realm-level roles (excluding built-in ones).
func (c *Client) ListRoles(ctx context.Context, realmName string) ([]ports.IdentityRole, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := c.gc.GetRealmRoles(ctx, token, realmName, gocloak.GetRoleParams{})
	if err != nil {
		return nil, fmt.Errorf("list roles in realm %s: %w", realmName, err)
	}
	var result []ports.IdentityRole
	for _, r := range roles {
		name := ""
		if r.Name != nil {
			name = *r.Name
		}
		// Skip Keycloak built-in roles
		if name == "uma_authorization" || name == "offline_access" || name == "default-roles-"+realmName {
			continue
		}
		role := ports.IdentityRole{Name: name}
		if r.ID != nil {
			role.ID = *r.ID
		}
		if r.Description != nil {
			role.Description = *r.Description
		}
		result = append(result, role)
	}
	return result, nil
}

// DeleteRole deletes a realm-level role.
func (c *Client) DeleteRole(ctx context.Context, realmName, roleName string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	return c.gc.DeleteRealmRole(ctx, token, realmName, roleName)
}

// GetUserRoles returns realm-level roles assigned to a user.
func (c *Client) GetUserRoles(ctx context.Context, realmName, userID string) ([]ports.IdentityRole, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := c.gc.GetRealmRolesByUserID(ctx, token, realmName, userID)
	if err != nil {
		return nil, fmt.Errorf("get roles for user %s: %w", userID, err)
	}
	var result []ports.IdentityRole
	for _, r := range roles {
		name := ""
		if r.Name != nil {
			name = *r.Name
		}
		if name == "uma_authorization" || name == "offline_access" || name == "default-roles-"+realmName {
			continue
		}
		role := ports.IdentityRole{Name: name}
		if r.ID != nil {
			role.ID = *r.ID
		}
		if r.Description != nil {
			role.Description = *r.Description
		}
		result = append(result, role)
	}
	return result, nil
}

// AssignRoleToUser assigns a realm-level role to a user.
func (c *Client) AssignRoleToUser(ctx context.Context, realmName, userID, roleName string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	role, err := c.gc.GetRealmRole(ctx, token, realmName, roleName)
	if err != nil {
		return fmt.Errorf("get role %s: %w", roleName, err)
	}
	return c.gc.AddRealmRoleToUser(ctx, token, realmName, userID, []gocloak.Role{*role})
}

// RemoveRoleFromUser removes a realm-level role from a user.
func (c *Client) RemoveRoleFromUser(ctx context.Context, realmName, userID, roleName string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	role, err := c.gc.GetRealmRole(ctx, token, realmName, roleName)
	if err != nil {
		return fmt.Errorf("get role %s: %w", roleName, err)
	}
	return c.gc.DeleteRealmRoleFromUser(ctx, token, realmName, userID, []gocloak.Role{*role})
}

func (c *Client) setUserEnabled(ctx context.Context, realmName, userID string, enabled bool) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}

	u, err := c.gc.GetUserByID(ctx, token, realmName, userID)
	if err != nil {
		return fmt.Errorf("get user %s: %w", userID, err)
	}

	u.Enabled = gocloak.BoolP(enabled)
	return c.gc.UpdateUser(ctx, token, realmName, *u)
}

func toIdentityUser(u *gocloak.User) *ports.IdentityUser {
	iu := &ports.IdentityUser{
		Enabled: true,
	}
	if u.ID != nil {
		iu.ID = *u.ID
	}
	if u.Email != nil {
		iu.Email = *u.Email
	}
	if u.FirstName != nil {
		iu.FirstName = *u.FirstName
	}
	if u.LastName != nil {
		iu.LastName = *u.LastName
	}
	if u.Enabled != nil {
		iu.Enabled = *u.Enabled
	}
	if u.EmailVerified != nil {
		iu.EmailVerified = *u.EmailVerified
	}
	if u.CreatedTimestamp != nil {
		iu.CreatedAt = *u.CreatedTimestamp
	}
	return iu
}

// MemoryKeycloakClient is a no-op implementation for dev/test.
type MemoryKeycloakClient struct {
	mu    sync.Mutex
	users map[string]map[string]*ports.IdentityUser // realm -> userID -> user
}

func NewMemoryClient() *MemoryKeycloakClient {
	return &MemoryKeycloakClient{
		users: make(map[string]map[string]*ports.IdentityUser),
	}
}

func (m *MemoryKeycloakClient) CreateRealm(_ context.Context, _, _ string) error { return nil }
func (m *MemoryKeycloakClient) DeleteRealm(_ context.Context, realmName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, realmName)
	return nil
}
func (m *MemoryKeycloakClient) CreateClient(_ context.Context, _, _, _ string) (string, error) {
	return "fake-client-secret", nil
}

func (m *MemoryKeycloakClient) CreateUser(_ context.Context, realmName, email, _, firstName, lastName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.users[realmName] == nil {
		m.users[realmName] = make(map[string]*ports.IdentityUser)
	}
	id := fmt.Sprintf("mem-user-%d", len(m.users[realmName])+1)
	m.users[realmName][id] = &ports.IdentityUser{
		ID: id, Email: email, FirstName: firstName, LastName: lastName,
		Enabled: true, CreatedAt: time.Now().UnixMilli(),
	}
	return id, nil
}

func (m *MemoryKeycloakClient) GetUser(_ context.Context, realmName, userID string) (*ports.IdentityUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ru, ok := m.users[realmName]; ok {
		if u, ok := ru[userID]; ok {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found: %s", userID)
}

func (m *MemoryKeycloakClient) ListUsers(_ context.Context, realmName string, first, max int) ([]ports.IdentityUser, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ru := m.users[realmName]
	all := make([]ports.IdentityUser, 0, len(ru))
	for _, u := range ru {
		all = append(all, *u)
	}
	total := len(all)
	if first >= len(all) {
		return nil, total, nil
	}
	end := first + max
	if end > len(all) {
		end = len(all)
	}
	return all[first:end], total, nil
}

func (m *MemoryKeycloakClient) DeleteUser(_ context.Context, realmName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ru, ok := m.users[realmName]; ok {
		delete(ru, userID)
	}
	return nil
}

func (m *MemoryKeycloakClient) DisableUser(_ context.Context, realmName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ru, ok := m.users[realmName]; ok {
		if u, ok := ru[userID]; ok {
			u.Enabled = false
		}
	}
	return nil
}

func (m *MemoryKeycloakClient) EnableUser(_ context.Context, realmName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ru, ok := m.users[realmName]; ok {
		if u, ok := ru[userID]; ok {
			u.Enabled = true
		}
	}
	return nil
}

func (m *MemoryKeycloakClient) CountUsers(_ context.Context, realmName string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.users[realmName]), nil
}

func (m *MemoryKeycloakClient) CreateRole(_ context.Context, _, _, _ string) error { return nil }
func (m *MemoryKeycloakClient) ListRoles(_ context.Context, _ string) ([]ports.IdentityRole, error) {
	return []ports.IdentityRole{}, nil
}
func (m *MemoryKeycloakClient) DeleteRole(_ context.Context, _, _ string) error { return nil }
func (m *MemoryKeycloakClient) GetUserRoles(_ context.Context, _, _ string) ([]ports.IdentityRole, error) {
	return []ports.IdentityRole{}, nil
}
func (m *MemoryKeycloakClient) AssignRoleToUser(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *MemoryKeycloakClient) RemoveRoleFromUser(_ context.Context, _, _, _ string) error {
	return nil
}
