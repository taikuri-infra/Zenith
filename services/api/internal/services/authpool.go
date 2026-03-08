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
	poolRepo    ports.AuthPoolRepository
	planRepo    ports.UserPlanRepository
	idp         ports.IdentityProvider
	keycloakURL string
}

// NewAuthPoolService creates a new AuthPoolService.
func NewAuthPoolService(poolRepo ports.AuthPoolRepository, planRepo ports.UserPlanRepository, idp ports.IdentityProvider, keycloakURL string) *AuthPoolService {
	return &AuthPoolService{
		poolRepo:    poolRepo,
		planRepo:    planRepo,
		idp:         idp,
		keycloakURL: keycloakURL,
	}
}

// CreatePool provisions a new auth pool (Keycloak realm + OIDC client + DB record).
func (s *AuthPoolService) CreatePool(ctx context.Context, userID, projectID, name string) (*entities.AuthPool, error) {
	poolID := uuid.New().String()
	realmName := "zp-" + poolID
	clientID := "zenith-pool-" + poolID

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

	// Create OIDC client in realm
	clientSecret, err := s.idp.CreateClient(ctx, realmName, clientID, "*")
	if err != nil {
		// Clean up realm on failure
		_ = s.idp.DeleteRealm(ctx, realmName)
		return nil, fmt.Errorf("create client: %w", err)
	}

	issuerURL := s.keycloakURL + "/realms/" + realmName

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

// DeletePool removes a pool and its Keycloak realm.
func (s *AuthPoolService) DeletePool(ctx context.Context, pool *entities.AuthPool) error {
	_ = s.poolRepo.UpdatePoolStatus(ctx, pool.ID, entities.AuthPoolStatusDeleting)

	if err := s.idp.DeleteRealm(ctx, pool.RealmName); err != nil {
		_ = s.poolRepo.UpdatePoolStatus(ctx, pool.ID, entities.AuthPoolStatusError)
		return fmt.Errorf("delete realm: %w", err)
	}

	return s.poolRepo.DeletePool(ctx, pool.ID)
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
