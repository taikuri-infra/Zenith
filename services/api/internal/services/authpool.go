package services

import (
	"context"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

// AuthPoolService orchestrates auth pool operations between Keycloak and the database.
type AuthPoolService struct {
	poolRepo           ports.AuthPoolRepository
	planRepo           ports.UserPlanRepository
	idp                ports.IdentityProvider
	keycloakURL        string // internal URL for API→Keycloak calls
	keycloakExternalURL string // public URL for issuer URLs shown to users
	gwRepo             ports.GatewayRepository
	gwSvc              *GatewayService
}

// SetGatewayDependencies wires gateway repo and service (breaks import cycle).
func (s *AuthPoolService) SetGatewayDependencies(gwRepo ports.GatewayRepository, gwSvc *GatewayService) {
	s.gwRepo = gwRepo
	s.gwSvc = gwSvc
}

// NewAuthPoolService creates a new AuthPoolService.
func NewAuthPoolService(poolRepo ports.AuthPoolRepository, planRepo ports.UserPlanRepository, idp ports.IdentityProvider, keycloakURL, keycloakExternalURL string) *AuthPoolService {
	return &AuthPoolService{
		poolRepo:           poolRepo,
		planRepo:           planRepo,
		idp:                idp,
		keycloakURL:        keycloakURL,
		keycloakExternalURL: keycloakExternalURL,
	}
}

// CreatePool provisions a new auth pool (Keycloak realm + OIDC client + DB record).
func (s *AuthPoolService) CreatePool(ctx context.Context, userID, projectID, name string) (*entities.AuthPool, error) {
	poolID := uuid.New().String()
	shortID := poolID[:8] // use first 8 chars for cleaner realm names
	realmName := "zp-" + shortID
	clientID := "zenith-pool-" + shortID

	// Get plan limits for max users
	maxUsers := 1000
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err == nil {
		maxUsers = plan.Limits.MaxAuthPoolUsers
	}

	// Create Keycloak realm
	if err := s.idp.CreateRealm(ctx, realmName, name); err != nil {
		return nil, fmt.Errorf("create realm: %w", err)
	}

	// Create OIDC client in realm (restricted redirect URI — no wildcards)
	clientSecret, err := s.idp.CreateClient(ctx, realmName, clientID, "urn:ietf:wg:oauth:2.0:oob")
	if err != nil {
		// Clean up realm on failure
		_ = s.idp.DeleteRealm(ctx, realmName)
		return nil, fmt.Errorf("create client: %w", err)
	}

	// Use external URL for user-facing issuer, fall back to internal
	baseURL := s.keycloakExternalURL
	if baseURL == "" {
		baseURL = s.keycloakURL
	}
	issuerURL := baseURL + "/realms/" + realmName

	// Persist to database
	pool, err := s.poolRepo.CreatePool(ctx, poolID, userID, projectID, name, realmName, clientID, clientSecret, issuerURL, maxUsers)
	if err != nil {
		_ = s.idp.DeleteRealm(ctx, realmName)
		return nil, err
	}

	// Mark active
	_ = s.poolRepo.UpdatePoolStatus(ctx, poolID, entities.AuthPoolStatusActive)
	pool.Status = entities.AuthPoolStatusActive

	return pool, nil
}

// GetPool returns a pool by ID.
func (s *AuthPoolService) GetPool(ctx context.Context, id string) (*entities.AuthPool, error) {
	return s.poolRepo.GetPool(ctx, id)
}

// ListPools returns all pools for a user.
func (s *AuthPoolService) ListPools(ctx context.Context, userID string) ([]entities.AuthPool, error) {
	return s.poolRepo.ListPoolsByUser(ctx, userID)
}

// DeletePool removes a pool and its Keycloak realm, clearing any gateway routes that reference it.
func (s *AuthPoolService) DeletePool(ctx context.Context, pool *entities.AuthPool) error {
	_ = s.poolRepo.UpdatePoolStatus(ctx, pool.ID, entities.AuthPoolStatusDeleting)

	// Clear auth pool references from gateway routes before deleting
	var affectedGwIDs []string
	if s.gwRepo != nil {
		ids, err := s.gwRepo.ClearAuthPoolFromRoutes(ctx, pool.ID)
		if err == nil {
			affectedGwIDs = ids
		}
	}

	if err := s.idp.DeleteRealm(ctx, pool.RealmName); err != nil {
		_ = s.poolRepo.UpdatePoolStatus(ctx, pool.ID, entities.AuthPoolStatusError)
		return fmt.Errorf("delete realm: %w", err)
	}

	if err := s.poolRepo.DeletePool(ctx, pool.ID); err != nil {
		return err
	}

	// Rebuild CRDs for affected gateways
	if s.gwSvc != nil && len(affectedGwIDs) > 0 {
		s.gwSvc.HandleAuthPoolDeleted(ctx, affectedGwIDs)
	}

	return nil
}

// CreateUser creates a user in a pool's Keycloak realm.
func (s *AuthPoolService) CreateUser(ctx context.Context, pool *entities.AuthPool, email, password, firstName, lastName string) (*ports.IdentityUser, error) {
	if pool.UserCount >= pool.MaxUsers {
		return nil, fmt.Errorf("pool user limit reached (%d/%d). Upgrade your plan for more.", pool.UserCount, pool.MaxUsers)
	}

	userID, err := s.idp.CreateUser(ctx, pool.RealmName, email, password, firstName, lastName)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	_ = s.poolRepo.UpdatePoolUserCount(ctx, pool.ID, 1)

	return &ports.IdentityUser{
		ID:        userID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Enabled:   true,
	}, nil
}

// GetUser retrieves a user from a pool's Keycloak realm.
func (s *AuthPoolService) GetUser(ctx context.Context, pool *entities.AuthPool, userID string) (*ports.IdentityUser, error) {
	return s.idp.GetUser(ctx, pool.RealmName, userID)
}

// ListUsers returns a paginated list of users in a pool.
func (s *AuthPoolService) ListUsers(ctx context.Context, pool *entities.AuthPool, first, max int) ([]ports.IdentityUser, int, error) {
	return s.idp.ListUsers(ctx, pool.RealmName, first, max)
}

// DeleteUser removes a user from a pool's Keycloak realm.
func (s *AuthPoolService) DeleteUser(ctx context.Context, pool *entities.AuthPool, userID string) error {
	if err := s.idp.DeleteUser(ctx, pool.RealmName, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	_ = s.poolRepo.UpdatePoolUserCount(ctx, pool.ID, -1)
	return nil
}

// DisableUser disables a user in a pool's Keycloak realm.
func (s *AuthPoolService) DisableUser(ctx context.Context, pool *entities.AuthPool, userID string) error {
	return s.idp.DisableUser(ctx, pool.RealmName, userID)
}

// EnableUser enables a user in a pool's Keycloak realm.
func (s *AuthPoolService) EnableUser(ctx context.Context, pool *entities.AuthPool, userID string) error {
	return s.idp.EnableUser(ctx, pool.RealmName, userID)
}

// CreateRole creates a realm-level role in a pool.
func (s *AuthPoolService) CreateRole(ctx context.Context, pool *entities.AuthPool, roleName, description string) error {
	return s.idp.CreateRole(ctx, pool.RealmName, roleName, description)
}

// ListRoles returns all custom roles in a pool.
func (s *AuthPoolService) ListRoles(ctx context.Context, pool *entities.AuthPool) ([]ports.IdentityRole, error) {
	return s.idp.ListRoles(ctx, pool.RealmName)
}

// DeleteRole removes a role from a pool.
func (s *AuthPoolService) DeleteRole(ctx context.Context, pool *entities.AuthPool, roleName string) error {
	return s.idp.DeleteRole(ctx, pool.RealmName, roleName)
}

// GetUserRoles returns roles assigned to a user.
func (s *AuthPoolService) GetUserRoles(ctx context.Context, pool *entities.AuthPool, userID string) ([]ports.IdentityRole, error) {
	return s.idp.GetUserRoles(ctx, pool.RealmName, userID)
}

// AssignRoleToUser assigns a role to a user.
func (s *AuthPoolService) AssignRoleToUser(ctx context.Context, pool *entities.AuthPool, userID, roleName string) error {
	return s.idp.AssignRoleToUser(ctx, pool.RealmName, userID, roleName)
}

// RemoveRoleFromUser removes a role from a user.
func (s *AuthPoolService) RemoveRoleFromUser(ctx context.Context, pool *entities.AuthPool, userID, roleName string) error {
	return s.idp.RemoveRoleFromUser(ctx, pool.RealmName, userID, roleName)
}

// UpdateUser updates a user's profile (first name, last name).
func (s *AuthPoolService) UpdateUser(ctx context.Context, pool *entities.AuthPool, userID, firstName, lastName string) error {
	return s.idp.UpdateUser(ctx, pool.RealmName, userID, firstName, lastName)
}

// SetUserPassword sets a user's password (admin action).
func (s *AuthPoolService) SetUserPassword(ctx context.Context, pool *entities.AuthPool, userID, password string) error {
	return s.idp.SetPassword(ctx, pool.RealmName, userID, password)
}

// SendPasswordReset triggers a password reset email for a user.
func (s *AuthPoolService) SendPasswordReset(ctx context.Context, pool *entities.AuthPool, email string) error {
	return s.idp.SendPasswordResetEmail(ctx, pool.RealmName, email)
}

// ResetPassword resets a user's password (with token or admin override).
func (s *AuthPoolService) ResetPassword(ctx context.Context, pool *entities.AuthPool, email, newPassword, resetToken string) error {
	// For now, use admin API to set password directly (simplified flow)
	// In production, we'd validate the reset token first
	return s.idp.ResetPasswordByEmail(ctx, pool.RealmName, email, newPassword)
}
