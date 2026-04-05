package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// mockIDP is a minimal in-memory IdentityProvider for testing.
type mockIDP struct {
	mu     sync.Mutex
	realms map[string]bool
	users  map[string]map[string]*ports.IdentityUser // realmName -> userID -> user
	roles  map[string]map[string]*ports.IdentityRole // realmName -> roleName -> role
	nextID int
}

func newMockIDP() *mockIDP {
	return &mockIDP{
		realms: make(map[string]bool),
		users:  make(map[string]map[string]*ports.IdentityUser),
		roles:  make(map[string]map[string]*ports.IdentityRole),
	}
}

func (m *mockIDP) CreateRealm(_ context.Context, realmName, displayName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.realms[realmName] {
		return fmt.Errorf("realm %s already exists", realmName)
	}
	m.realms[realmName] = true
	m.users[realmName] = make(map[string]*ports.IdentityUser)
	m.roles[realmName] = make(map[string]*ports.IdentityRole)
	return nil
}

func (m *mockIDP) DeleteRealm(_ context.Context, realmName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.realms[realmName] {
		return fmt.Errorf("realm %s not found", realmName)
	}
	delete(m.realms, realmName)
	delete(m.users, realmName)
	delete(m.roles, realmName)
	return nil
}

func (m *mockIDP) CreateClient(_ context.Context, realmName, clientID, redirectURI string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.realms[realmName] {
		return "", fmt.Errorf("realm %s not found", realmName)
	}
	return "mock-client-secret", nil
}

func (m *mockIDP) CreateUser(_ context.Context, realmName, email, password, firstName, lastName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.realms[realmName] {
		return "", fmt.Errorf("realm %s not found", realmName)
	}
	m.nextID++
	id := fmt.Sprintf("user-%d", m.nextID)
	m.users[realmName][id] = &ports.IdentityUser{
		ID: id, Email: email, FirstName: firstName, LastName: lastName, Enabled: true,
	}
	return id, nil
}

func (m *mockIDP) GetUser(_ context.Context, realmName, userID string) (*ports.IdentityUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return nil, fmt.Errorf("realm %s not found", realmName)
	}
	u, ok := users[userID]
	if !ok {
		return nil, fmt.Errorf("user %s not found", userID)
	}
	return u, nil
}

func (m *mockIDP) ListUsers(_ context.Context, realmName string, first, max int) ([]ports.IdentityUser, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return nil, 0, fmt.Errorf("realm %s not found", realmName)
	}
	var result []ports.IdentityUser
	for _, u := range users {
		result = append(result, *u)
	}
	return result, len(result), nil
}

func (m *mockIDP) DeleteUser(_ context.Context, realmName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return fmt.Errorf("realm %s not found", realmName)
	}
	delete(users, userID)
	return nil
}

func (m *mockIDP) DisableUser(_ context.Context, realmName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return fmt.Errorf("realm %s not found", realmName)
	}
	u, ok := users[userID]
	if !ok {
		return fmt.Errorf("user %s not found", userID)
	}
	u.Enabled = false
	return nil
}

func (m *mockIDP) EnableUser(_ context.Context, realmName, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return fmt.Errorf("realm %s not found", realmName)
	}
	u, ok := users[userID]
	if !ok {
		return fmt.Errorf("user %s not found", userID)
	}
	u.Enabled = true
	return nil
}

func (m *mockIDP) CountUsers(_ context.Context, realmName string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return 0, fmt.Errorf("realm %s not found", realmName)
	}
	return len(users), nil
}

func (m *mockIDP) UpdateUser(_ context.Context, realmName, userID, firstName, lastName string) error {
	return nil
}
func (m *mockIDP) SetPassword(_ context.Context, realmName, userID, password string) error {
	return nil
}
func (m *mockIDP) SendPasswordResetEmail(_ context.Context, realmName, email string) error {
	return nil
}
func (m *mockIDP) ResetPasswordByEmail(_ context.Context, realmName, email, newPassword string) error {
	return nil
}
func (m *mockIDP) CreateRole(_ context.Context, realmName, roleName, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	roles, ok := m.roles[realmName]
	if !ok {
		return fmt.Errorf("realm %s not found", realmName)
	}
	roles[roleName] = &ports.IdentityRole{Name: roleName, Description: description}
	return nil
}
func (m *mockIDP) ListRoles(_ context.Context, realmName string) ([]ports.IdentityRole, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	roles, ok := m.roles[realmName]
	if !ok {
		return nil, fmt.Errorf("realm %s not found", realmName)
	}
	var result []ports.IdentityRole
	for _, r := range roles {
		result = append(result, *r)
	}
	return result, nil
}
func (m *mockIDP) DeleteRole(_ context.Context, realmName, roleName string) error {
	return nil
}
func (m *mockIDP) GetUserRoles(_ context.Context, realmName, userID string) ([]ports.IdentityRole, error) {
	return nil, nil
}
func (m *mockIDP) AssignRoleToUser(_ context.Context, realmName, userID, roleName string) error {
	return nil
}
func (m *mockIDP) RemoveRoleFromUser(_ context.Context, realmName, userID, roleName string) error {
	return nil
}
func (m *mockIDP) SendVerifyEmail(_ context.Context, realmName, userID string) error {
	return nil
}
func (m *mockIDP) GetUserMetadata(_ context.Context, realmName, userID string) (map[string][]string, error) {
	return nil, nil
}
func (m *mockIDP) SetUserMetadata(_ context.Context, realmName, userID string, metadata map[string][]string) error {
	return nil
}
func (m *mockIDP) GetUserCredentials(_ context.Context, realmName, userID string) ([]ports.IdentityCredential, error) {
	return nil, nil
}
func (m *mockIDP) DeleteUserCredential(_ context.Context, realmName, userID, credentialID string) error {
	return nil
}
func (m *mockIDP) GetUserSessions(_ context.Context, realmName, userID string) ([]ports.IdentitySession, error) {
	return nil, nil
}
func (m *mockIDP) RevokeUserSession(_ context.Context, realmName, sessionID string) error {
	return nil
}
func (m *mockIDP) RevokeAllUserSessions(_ context.Context, realmName, userID string) error {
	return nil
}
func (m *mockIDP) CreateIdentityProvider(_ context.Context, realmName string, provider ports.IdentityProviderConfig) error {
	return nil
}
func (m *mockIDP) ListIdentityProviders(_ context.Context, realmName string) ([]ports.IdentityProviderConfig, error) {
	return nil, nil
}
func (m *mockIDP) DeleteIdentityProvider(_ context.Context, realmName, alias string) error {
	return nil
}
func (m *mockIDP) InviteUser(_ context.Context, realmName, email, firstName, lastName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("user-%d", m.nextID)
	if users, ok := m.users[realmName]; ok {
		users[id] = &ports.IdentityUser{ID: id, Email: email, FirstName: firstName, LastName: lastName, Enabled: true}
	}
	return id, nil
}
func (m *mockIDP) FindUserByEmail(_ context.Context, realmName, email string) (*ports.IdentityUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	users, ok := m.users[realmName]
	if !ok {
		return nil, fmt.Errorf("realm not found")
	}
	for _, u := range users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}
func (m *mockIDP) GetEmailSettings(_ context.Context, realmName string) (*ports.EmailSettings, error) {
	return &ports.EmailSettings{}, nil
}
func (m *mockIDP) UpdateEmailSettings(_ context.Context, realmName string, settings *ports.EmailSettings) error {
	return nil
}

// --- helper ---

func newTestAuthPoolService() (*AuthPoolService, *memory.MemoryAuthPoolRepository, *mockIDP) {
	poolRepo := memory.NewMemoryAuthPoolRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	idp := newMockIDP()
	svc := NewAuthPoolService(poolRepo, planRepo, idp, "http://keycloak:8080", "https://auth.example.com")
	return svc, poolRepo, idp
}

// --- NewAuthPoolService tests ---

func TestNewAuthPoolService(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	if svc == nil {
		t.Fatal("Expected non-nil AuthPoolService")
	}
}

// --- SetGatewayDependencies tests ---

func TestSetGatewayDependencies(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	svc.SetGatewayDependencies(nil, nil)
	// No panic means success
}

// --- CreatePool tests ---

func TestCreatePool_Success(t *testing.T) {
	svc, _, idp := newTestAuthPoolService()
	ctx := context.Background()

	pool, err := svc.CreatePool(ctx, "user-1", "project-1", "My Pool")
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}
	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}
	if pool.Name != "My Pool" {
		t.Errorf("Expected name 'My Pool', got '%s'", pool.Name)
	}
	if pool.Status != entities.AuthPoolStatusActive {
		t.Errorf("Expected status active, got '%s'", pool.Status)
	}
	if pool.IssuerURL == "" {
		t.Error("Expected non-empty issuer URL")
	}
	if pool.ClientSecret != "mock-client-secret" {
		t.Errorf("Expected client secret 'mock-client-secret', got '%s'", pool.ClientSecret)
	}

	// Verify realm was created in IDP
	idp.mu.Lock()
	if !idp.realms[pool.RealmName] {
		t.Errorf("Expected realm '%s' to exist in IDP", pool.RealmName)
	}
	idp.mu.Unlock()
}

func TestCreatePool_UsesExternalURL(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, err := svc.CreatePool(ctx, "user-1", "project-1", "External URL Pool")
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}
	// Should use the external URL (https://auth.example.com) not the internal one
	if pool.IssuerURL == "" {
		t.Fatal("Expected non-empty issuer URL")
	}
	expected := "https://auth.example.com/realms/" + pool.RealmName
	if pool.IssuerURL != expected {
		t.Errorf("Expected issuer URL '%s', got '%s'", expected, pool.IssuerURL)
	}
}

// --- GetPool tests ---

func TestGetPool_Success(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	created, _ := svc.CreatePool(ctx, "user-1", "project-1", "Get Pool Test")
	pool, err := svc.GetPool(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}
	if pool.Name != "Get Pool Test" {
		t.Errorf("Expected name 'Get Pool Test', got '%s'", pool.Name)
	}
}

func TestGetPool_NotFound(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	_, err := svc.GetPool(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error for nonexistent pool")
	}
}

// --- ListPools tests ---

func TestListPools_Empty(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pools, err := svc.ListPools(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListPools failed: %v", err)
	}
	if len(pools) != 0 {
		t.Errorf("Expected 0 pools, got %d", len(pools))
	}
}

func TestListPools_MultiplePoolsForUser(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	svc.CreatePool(ctx, "user-1", "project-1", "Pool A")
	svc.CreatePool(ctx, "user-1", "project-1", "Pool B")
	svc.CreatePool(ctx, "user-2", "project-2", "Pool C") // different user

	pools, err := svc.ListPools(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListPools failed: %v", err)
	}
	if len(pools) != 2 {
		t.Errorf("Expected 2 pools for user-1, got %d", len(pools))
	}
}

// --- DeletePool tests ---

func TestDeletePool_Success(t *testing.T) {
	svc, _, idp := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Delete Me")
	realmName := pool.RealmName

	err := svc.DeletePool(ctx, pool)
	if err != nil {
		t.Fatalf("DeletePool failed: %v", err)
	}

	// Verify realm was deleted in IDP
	idp.mu.Lock()
	if idp.realms[realmName] {
		t.Error("Expected realm to be deleted from IDP")
	}
	idp.mu.Unlock()

	// Verify pool was deleted from repo
	_, err = svc.GetPool(ctx, pool.ID)
	if err == nil {
		t.Error("Expected error getting deleted pool")
	}
}

// --- CreateUser tests ---

func TestCreateUser_Success(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "User Pool")

	user, err := svc.CreateUser(ctx, pool, "test@example.com", "P@ss1234", "John", "Doe")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}
	if user.FirstName != "John" {
		t.Errorf("Expected firstName 'John', got '%s'", user.FirstName)
	}
}

func TestCreateUser_LimitReached(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Limited Pool")
	pool.MaxUsers = 1
	pool.UserCount = 1

	_, err := svc.CreateUser(ctx, pool, "over@example.com", "P@ss1234", "Over", "Limit")
	if err == nil {
		t.Error("Expected error when pool user limit reached")
	}
}

// --- GetUser tests ---

func TestGetUser_FromPool(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "get@example.com", "P@ss1234", "Get", "User")

	user, err := svc.GetUser(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Email != "get@example.com" {
		t.Errorf("Expected email 'get@example.com', got '%s'", user.Email)
	}
}

// --- ListUsers tests ---

func TestListUsers_FromPool(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	svc.CreateUser(ctx, pool, "a@example.com", "P@ss1234", "A", "User")
	svc.CreateUser(ctx, pool, "b@example.com", "P@ss1234", "B", "User")

	users, total, err := svc.ListUsers(ctx, pool, 0, 100)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
}

// --- DeleteUser tests ---

func TestDeleteUser_Success(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "del@example.com", "P@ss1234", "Del", "User")

	err := svc.DeleteUser(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
}

// --- DisableUser / EnableUser tests ---

func TestDisableEnableUser(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "toggle@example.com", "P@ss1234", "Toggle", "User")

	err := svc.DisableUser(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("DisableUser failed: %v", err)
	}

	err = svc.EnableUser(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("EnableUser failed: %v", err)
	}
}

// --- Role tests ---

func TestCreateRole(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.CreateRole(ctx, pool, "admin", "Administrator role")
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
}

func TestListRoles(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	svc.CreateRole(ctx, pool, "admin", "Admin")
	svc.CreateRole(ctx, pool, "viewer", "Viewer")

	roles, err := svc.ListRoles(ctx, pool)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}
}

func TestDeleteRole(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.DeleteRole(ctx, pool, "admin")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
}

// --- UpdateUser / SetUserPassword tests ---

func TestUpdateUser(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "upd@example.com", "P@ss1234", "Old", "Name")

	err := svc.UpdateUser(ctx, pool, created.ID, "New", "Name")
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
}

func TestSetUserPassword(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "pwd@example.com", "P@ss1234", "Pwd", "User")

	err := svc.SetUserPassword(ctx, pool, created.ID, "NewP@ss5678")
	if err != nil {
		t.Fatalf("SetUserPassword failed: %v", err)
	}
}

// --- SendPasswordReset / ResetPassword tests ---

func TestSendPasswordReset(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.SendPasswordReset(ctx, pool, "user@example.com")
	if err != nil {
		t.Fatalf("SendPasswordReset failed: %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.ResetPassword(ctx, pool, "user@example.com", "NewP@ss", "reset-token")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}
}

// --- InviteUser tests ---

func TestInviteUser_Success(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	user, err := svc.InviteUser(ctx, pool, "invite@example.com", "Invite", "User")
	if err != nil {
		t.Fatalf("InviteUser failed: %v", err)
	}
	if user.Email != "invite@example.com" {
		t.Errorf("Expected email 'invite@example.com', got '%s'", user.Email)
	}
}

func TestInviteUser_LimitReached(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	pool.MaxUsers = 0
	pool.UserCount = 0

	_, err := svc.InviteUser(ctx, pool, "invite@example.com", "Invite", "User")
	if err == nil {
		t.Error("Expected error when pool limit reached")
	}
}

// --- FindUserByEmail tests ---

func TestFindUserByEmail(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	svc.CreateUser(ctx, pool, "find@example.com", "P@ss1234", "Find", "Me")

	user, err := svc.FindUserByEmail(ctx, pool, "find@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail failed: %v", err)
	}
	if user.Email != "find@example.com" {
		t.Errorf("Expected email 'find@example.com', got '%s'", user.Email)
	}
}

// --- CreateAnonymousUser tests ---

func TestCreateAnonymousUser_Success(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	user, password, err := svc.CreateAnonymousUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateAnonymousUser failed: %v", err)
	}
	if user.FirstName != "Anonymous" {
		t.Errorf("Expected firstName 'Anonymous', got '%s'", user.FirstName)
	}
	if password == "" {
		t.Error("Expected non-empty password")
	}
}

func TestCreateAnonymousUser_LimitReached(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	pool.MaxUsers = 0
	pool.UserCount = 0

	_, _, err := svc.CreateAnonymousUser(ctx, pool)
	if err == nil {
		t.Error("Expected error when pool limit reached")
	}
}

// --- Email settings tests ---

func TestGetEmailSettings(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	settings, err := svc.GetEmailSettings(ctx, pool)
	if err != nil {
		t.Fatalf("GetEmailSettings failed: %v", err)
	}
	if settings == nil {
		t.Error("Expected non-nil settings")
	}
}

func TestUpdateEmailSettings(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.UpdateEmailSettings(ctx, pool, &ports.EmailSettings{Host: "smtp.example.com"})
	if err != nil {
		t.Fatalf("UpdateEmailSettings failed: %v", err)
	}
}

// --- Session and credential tests ---

func TestSendVerifyEmail(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "verify@example.com", "P@ss1234", "V", "U")

	err := svc.SendVerifyEmail(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("SendVerifyEmail failed: %v", err)
	}
}

func TestGetUserCredentials(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "cred@example.com", "P@ss1234", "C", "U")

	creds, err := svc.GetUserCredentials(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("GetUserCredentials failed: %v", err)
	}
	_ = creds // no credentials in mock
}

func TestGetUserSessions(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "sess@example.com", "P@ss1234", "S", "U")

	sessions, err := svc.GetUserSessions(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("GetUserSessions failed: %v", err)
	}
	_ = sessions
}

func TestRevokeUserSession(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.RevokeUserSession(ctx, pool, "session-id")
	if err != nil {
		t.Fatalf("RevokeUserSession failed: %v", err)
	}
}

func TestRevokeAllUserSessions(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.RevokeAllUserSessions(ctx, pool, "some-user-id")
	if err != nil {
		t.Fatalf("RevokeAllUserSessions failed: %v", err)
	}
}

// --- Identity provider tests ---

func TestCreateIdentityProvider(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.CreateIdentityProvider(ctx, pool, ports.IdentityProviderConfig{
		Alias:      "google",
		ProviderID: "google",
		ClientID:   "google-id",
	})
	if err != nil {
		t.Fatalf("CreateIdentityProvider failed: %v", err)
	}
}

func TestListIdentityProviders(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	providers, err := svc.ListIdentityProviders(ctx, pool)
	if err != nil {
		t.Fatalf("ListIdentityProviders failed: %v", err)
	}
	_ = providers
}

func TestDeleteIdentityProvider(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.DeleteIdentityProvider(ctx, pool, "google")
	if err != nil {
		t.Fatalf("DeleteIdentityProvider failed: %v", err)
	}
}

// --- GetUserMetadata / SetUserMetadata ---

func TestGetSetUserMetadata(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "meta@example.com", "P@ss1234", "M", "U")

	err := svc.SetUserMetadata(ctx, pool, created.ID, map[string][]string{"role": {"admin"}})
	if err != nil {
		t.Fatalf("SetUserMetadata failed: %v", err)
	}

	_, err = svc.GetUserMetadata(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("GetUserMetadata failed: %v", err)
	}
}

// --- GetUserRoles / AssignRole / RemoveRole ---

func TestAssignAndRemoveRoles(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	created, _ := svc.CreateUser(ctx, pool, "roles@example.com", "P@ss1234", "R", "U")

	err := svc.AssignRoleToUser(ctx, pool, created.ID, "admin")
	if err != nil {
		t.Fatalf("AssignRoleToUser failed: %v", err)
	}

	roles, err := svc.GetUserRoles(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}
	_ = roles

	err = svc.RemoveRoleFromUser(ctx, pool, created.ID, "admin")
	if err != nil {
		t.Fatalf("RemoveRoleFromUser failed: %v", err)
	}
}

// --- DeleteUserCredential ---

func TestDeleteUserCredential(t *testing.T) {
	svc, _, _ := newTestAuthPoolService()
	ctx := context.Background()

	pool, _ := svc.CreatePool(ctx, "user-1", "project-1", "Pool")
	err := svc.DeleteUserCredential(ctx, pool, "user-id", "cred-id")
	if err != nil {
		t.Fatalf("DeleteUserCredential failed: %v", err)
	}
}
