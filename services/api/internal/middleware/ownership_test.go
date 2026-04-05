package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/gofiber/fiber/v2"
)

// mockAppRepository implements ports.AppRepository for ownership tests.
// Only GetApp is wired; all other methods are no-op stubs.
type mockAppRepository struct {
	apps map[string]*entities.App
	err  error
}

func (m *mockAppRepository) GetApp(_ context.Context, id string) (*entities.App, error) {
	if m.err != nil {
		return nil, m.err
	}
	app, ok := m.apps[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return app, nil
}

// --- Stubs (not used by RequireAppOwnership) ---

func (m *mockAppRepository) CreateApp(_ context.Context, _ *dto.CreateAppInput) (*entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) GetAppBySubdomain(_ context.Context, _ string) (*entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) ListAppsByUser(_ context.Context, _ string) ([]entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) ListAppsByProject(_ context.Context, _ string) ([]entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) UpdateApp(_ context.Context, _ string, _ *dto.UpdateAppInput) (*entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) DeleteApp(_ context.Context, _ string) error     { return nil }
func (m *mockAppRepository) SoftDeleteApp(_ context.Context, _ string) error { return nil }
func (m *mockAppRepository) RestoreApp(_ context.Context, _ string) (*entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) ListDeletedAppsByUser(_ context.Context, _ string) ([]entities.App, error) {
	return nil, nil
}
func (m *mockAppRepository) SetAutoGatewayID(_ context.Context, _, _ string) error { return nil }
func (m *mockAppRepository) CountAppsByUser(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockAppRepository) CountApps(_ context.Context) (int, error)             { return 0, nil }
func (m *mockAppRepository) ListAllApps(_ context.Context) ([]entities.App, error) { return nil, nil }
func (m *mockAppRepository) CreateDeployment(_ context.Context, _, _ string) (*entities.Deployment, error) {
	return nil, nil
}
func (m *mockAppRepository) GetDeployment(_ context.Context, _ string) (*entities.Deployment, error) {
	return nil, nil
}
func (m *mockAppRepository) ListDeployments(_ context.Context, _ string, _ int) ([]entities.Deployment, error) {
	return nil, nil
}
func (m *mockAppRepository) UpdateDeploymentStatus(_ context.Context, _ string, _ entities.DeploymentStatus, _, _ string) error {
	return nil
}
func (m *mockAppRepository) GetActiveDeployment(_ context.Context, _ string) (*entities.Deployment, error) {
	return nil, nil
}
func (m *mockAppRepository) SetEnvVars(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (m *mockAppRepository) GetEnvVars(_ context.Context, _ string) ([]entities.EnvVar, error) {
	return nil, nil
}
func (m *mockAppRepository) DeleteEnvVar(_ context.Context, _, _ string) error { return nil }
func (m *mockAppRepository) SetSecret(_ context.Context, _, _ string, _ []byte) error {
	return nil
}
func (m *mockAppRepository) GetSecrets(_ context.Context, _ string) ([]entities.Secret, error) {
	return nil, nil
}
func (m *mockAppRepository) GetSecretValue(_ context.Context, _, _ string) ([]byte, error) {
	return nil, nil
}
func (m *mockAppRepository) DeleteSecret(_ context.Context, _, _ string) error { return nil }
func (m *mockAppRepository) CreateRelease(_ context.Context, _ string, _ *dto.CreateReleaseInput) (*entities.Release, error) {
	return nil, nil
}
func (m *mockAppRepository) ListReleases(_ context.Context, _ string, _ int) ([]entities.Release, error) {
	return nil, nil
}
func (m *mockAppRepository) GetRelease(_ context.Context, _ string) (*entities.Release, error) {
	return nil, nil
}

// --- Tests ---

func TestRequireAppOwnershipSuccess(t *testing.T) {
	repo := &mockAppRepository{
		apps: map[string]*entities.App{
			"app-1": {ID: "app-1", UserID: "user-owner"},
		},
	}

	var capturedApp *entities.App

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-owner")
		return c.Next()
	})
	app.Get("/apps/:appId", RequireAppOwnership(repo), func(c *fiber.Ctx) error {
		capturedApp, _ = c.Locals("app").(*entities.App)
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/apps/app-1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	if capturedApp == nil || capturedApp.ID != "app-1" {
		t.Error("Expected app to be stored in Locals")
	}
}

func TestRequireAppOwnershipWrongUser(t *testing.T) {
	repo := &mockAppRepository{
		apps: map[string]*entities.App{
			"app-1": {ID: "app-1", UserID: "user-owner"},
		},
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-attacker")
		return c.Next()
	})
	app.Get("/apps/:appId", RequireAppOwnership(repo), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/apps/app-1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	// Returns 404 (not 403) to avoid leaking resource existence
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404 (not 403 to avoid leaking), got %d", resp.StatusCode)
	}
}

func TestRequireAppOwnershipAppNotFound(t *testing.T) {
	repo := &mockAppRepository{
		apps: map[string]*entities.App{},
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return c.Next()
	})
	app.Get("/apps/:appId", RequireAppOwnership(repo), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/apps/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestRequireAppOwnershipNoUserID(t *testing.T) {
	repo := &mockAppRepository{
		apps: map[string]*entities.App{
			"app-1": {ID: "app-1", UserID: "user-owner"},
		},
	}

	app := fiber.New()
	// No user_id set in Locals
	app.Get("/apps/:appId", RequireAppOwnership(repo), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/apps/app-1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestRequireAppOwnershipNoAppIdParam(t *testing.T) {
	repo := &mockAppRepository{
		apps: map[string]*entities.App{},
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return c.Next()
	})
	// Route without :appId param - middleware should pass through
	app.Get("/other", RequireAppOwnership(repo), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/other", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (pass through), got %d", resp.StatusCode)
	}
}

func TestRequireAppOwnershipRepoError(t *testing.T) {
	repo := &mockAppRepository{
		err: errors.New("database connection error"),
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return c.Next()
	})
	app.Get("/apps/:appId", RequireAppOwnership(repo), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/apps/app-1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404 for repo error, got %d", resp.StatusCode)
	}
}
