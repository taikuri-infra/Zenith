package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/gofiber/fiber/v2"
)

const testSecret = "test-secret-key-for-testing"

func setupAuthApp() *fiber.App {
	return fiber.New()
}

func TestGenerateAndValidateToken(t *testing.T) {
	user := &models.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      models.RoleDeveloper,
		ProjectID: "proj-456",
	}

	token, err := GenerateToken(testSecret, user, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated token is empty")
	}
}

func TestJWTAuthMissingHeader(t *testing.T) {
	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthInvalidFormat(t *testing.T) {
	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthInvalidToken(t *testing.T) {
	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthValidToken(t *testing.T) {
	user := &models.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      models.RoleDeveloper,
		ProjectID: "proj-456",
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		email := c.Locals("email").(string)
		role := c.Locals("role").(models.Role)
		return c.JSON(fiber.Map{"email": email, "role": role})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestJWTAuthExpiredToken(t *testing.T) {
	user := &models.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  models.RoleDeveloper,
	}

	token, _ := GenerateToken(testSecret, user, -1*time.Hour) // expired

	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 for expired token, got %d", resp.StatusCode)
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name     string
		userRole models.Role
		minRole  models.Role
		wantCode int
	}{
		{"owner can access admin", models.RoleOwner, models.RoleAdmin, 200},
		{"admin can access admin", models.RoleAdmin, models.RoleAdmin, 200},
		{"developer cannot access admin", models.RoleDeveloper, models.RoleAdmin, 403},
		{"viewer cannot access developer", models.RoleViewer, models.RoleDeveloper, 403},
		{"viewer can access viewer", models.RoleViewer, models.RoleViewer, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupAuthApp()
			// Inject role
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("role", tt.userRole)
				return c.Next()
			})
			app.Use(RequireRole(tt.minRole))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.wantCode {
				t.Errorf("Expected %d, got %d", tt.wantCode, resp.StatusCode)
			}
		})
	}
}

func TestRequireRoleNoRole(t *testing.T) {
	app := setupAuthApp()
	app.Use(RequireRole(models.RoleViewer))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for no role, got %d", resp.StatusCode)
	}
}

func TestConstantTimeCompare(t *testing.T) {
	if !ConstantTimeCompare("abc", "abc") {
		t.Error("Expected equal strings to match")
	}
	if ConstantTimeCompare("abc", "def") {
		t.Error("Expected different strings to not match")
	}
}
