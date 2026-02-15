package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
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

func TestJWTAuthWrongSecret(t *testing.T) {
	user := &models.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  models.RoleDeveloper,
	}

	// Generate token with one secret, validate with another
	token, _ := GenerateToken("secret-one", user, 1*time.Hour)

	app := setupAuthApp()
	app.Use(JWTAuth("secret-two"))
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
		t.Errorf("Expected 401 for wrong secret, got %d", resp.StatusCode)
	}
}

func TestJWTAuthMalformedBearer(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"only bearer keyword", "Bearer"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"empty bearer value", "Bearer "},
		{"no space after bearer", "Bearertoken123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupAuthApp()
			app.Use(JWTAuth(testSecret))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != 401 {
				t.Errorf("Expected 401, got %d", resp.StatusCode)
			}
		})
	}
}

func TestJWTAuthSetsLocals(t *testing.T) {
	user := &models.User{
		ID:        "user-999",
		Email:     "admin@zenith.dev",
		Name:      "Admin User",
		Role:      models.RoleOwner,
		ProjectID: "proj-001",
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	var capturedUserID, capturedEmail, capturedName, capturedProjectID string
	var capturedRole models.Role

	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedUserID, _ = c.Locals("user_id").(string)
		capturedEmail, _ = c.Locals("email").(string)
		capturedName, _ = c.Locals("name").(string)
		capturedRole, _ = c.Locals("role").(models.Role)
		capturedProjectID, _ = c.Locals("project_id").(string)
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	if capturedUserID != "user-999" {
		t.Errorf("Expected user_id 'user-999', got '%s'", capturedUserID)
	}
	if capturedEmail != "admin@zenith.dev" {
		t.Errorf("Expected email 'admin@zenith.dev', got '%s'", capturedEmail)
	}
	if capturedName != "Admin User" {
		t.Errorf("Expected name 'Admin User', got '%s'", capturedName)
	}
	if capturedRole != models.RoleOwner {
		t.Errorf("Expected role 'owner', got '%s'", capturedRole)
	}
	if capturedProjectID != "proj-001" {
		t.Errorf("Expected project_id 'proj-001', got '%s'", capturedProjectID)
	}
}

func TestAPIKeyAuthValid(t *testing.T) {
	validKey := &models.APIKey{
		UserID:    "user-api-001",
		ProjectID: "proj-api-001",
		Scopes:    []string{"read", "write"},
	}

	validator := func(key string) (*models.APIKey, error) {
		if key == "valid-api-key-123" {
			return validKey, nil
		}
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid")
	}

	var capturedUserID, capturedProjectID string
	var capturedRole models.Role

	app := setupAuthApp()
	app.Use(APIKeyAuth(validator))
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedUserID, _ = c.Locals("user_id").(string)
		capturedProjectID, _ = c.Locals("project_id").(string)
		capturedRole, _ = c.Locals("role").(models.Role)
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-api-key-123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	if capturedUserID != "user-api-001" {
		t.Errorf("Expected user_id 'user-api-001', got '%s'", capturedUserID)
	}
	if capturedProjectID != "proj-api-001" {
		t.Errorf("Expected project_id 'proj-api-001', got '%s'", capturedProjectID)
	}
	if capturedRole != models.RoleDeveloper {
		t.Errorf("Expected role 'developer' (API key default), got '%s'", capturedRole)
	}
}

func TestAPIKeyAuthInvalid(t *testing.T) {
	validator := func(key string) (*models.APIKey, error) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid")
	}

	app := setupAuthApp()
	app.Use(APIKeyAuth(validator))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "bad-key")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestAPIKeyAuthMissingHeaderPassesThrough(t *testing.T) {
	validator := func(key string) (*models.APIKey, error) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid")
	}

	app := setupAuthApp()
	app.Use(APIKeyAuth(validator))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// No X-API-Key header - should pass through to next handler
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (pass through), got %d", resp.StatusCode)
	}
}

func TestRequireAuthWithAPIKey(t *testing.T) {
	app := setupAuthApp()
	// Simulate API key already set by prior middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "api-user-001")
		return c.Next()
	})
	app.Use(RequireAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireAuthWithJWT(t *testing.T) {
	user := &models.User{
		ID:    "user-jwt",
		Email: "jwt@test.com",
		Role:  models.RoleDeveloper,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	app := setupAuthApp()
	app.Use(RequireAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
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

func TestRequireAuthNoAuth(t *testing.T) {
	app := setupAuthApp()
	app.Use(RequireAuth(testSecret))
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

func TestRequireRoleAllCombinations(t *testing.T) {
	tests := []struct {
		name     string
		userRole models.Role
		minRole  models.Role
		wantCode int
	}{
		// Owner tests
		{"owner can access owner", models.RoleOwner, models.RoleOwner, 200},
		{"owner can access admin", models.RoleOwner, models.RoleAdmin, 200},
		{"owner can access developer", models.RoleOwner, models.RoleDeveloper, 200},
		{"owner can access viewer", models.RoleOwner, models.RoleViewer, 200},
		// Admin tests
		{"admin cannot access owner", models.RoleAdmin, models.RoleOwner, 403},
		{"admin can access admin", models.RoleAdmin, models.RoleAdmin, 200},
		{"admin can access developer", models.RoleAdmin, models.RoleDeveloper, 200},
		{"admin can access viewer", models.RoleAdmin, models.RoleViewer, 200},
		// Developer tests
		{"developer cannot access owner", models.RoleDeveloper, models.RoleOwner, 403},
		{"developer cannot access admin", models.RoleDeveloper, models.RoleAdmin, 403},
		{"developer can access developer", models.RoleDeveloper, models.RoleDeveloper, 200},
		{"developer can access viewer", models.RoleDeveloper, models.RoleViewer, 200},
		// Viewer tests
		{"viewer cannot access owner", models.RoleViewer, models.RoleOwner, 403},
		{"viewer cannot access admin", models.RoleViewer, models.RoleAdmin, 403},
		{"viewer cannot access developer", models.RoleViewer, models.RoleDeveloper, 403},
		{"viewer can access viewer", models.RoleViewer, models.RoleViewer, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupAuthApp()
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

func TestRequireScopeWithAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		required string
		wantCode int
	}{
		{"has exact scope", []string{"read", "write"}, "read", 200},
		{"has wildcard scope", []string{"*"}, "deploy", 200},
		{"missing scope", []string{"read"}, "write", 403},
		{"empty scopes", []string{}, "read", 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupAuthApp()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("api_key", &models.APIKey{
					Scopes: tt.scopes,
				})
				return c.Next()
			})
			app.Use(RequireScope(tt.required))
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

func TestRequireScopeWithoutAPIKey(t *testing.T) {
	// When no API key is present, scope check should be skipped
	app := setupAuthApp()
	app.Use(RequireScope("admin"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (scope skipped), got %d", resp.StatusCode)
	}
}

func TestMiddlewareChain(t *testing.T) {
	user := &models.User{
		ID:    "user-chain",
		Email: "chain@test.com",
		Role:  models.RoleAdmin,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Use(RequireRole(models.RoleDeveloper))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (admin >= developer), got %d", resp.StatusCode)
	}
}

func TestMiddlewareChainInsufficientRole(t *testing.T) {
	user := &models.User{
		ID:    "user-chain",
		Email: "chain@test.com",
		Role:  models.RoleViewer,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Use(RequireRole(models.RoleAdmin))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 (viewer < admin), got %d", resp.StatusCode)
	}
}

func TestGenerateTokenFields(t *testing.T) {
	user := &models.User{
		ID:        "user-gen",
		Email:     "gen@test.com",
		Name:      "Gen User",
		Role:      models.RoleAdmin,
		ProjectID: "proj-gen",
	}

	tokenStr, err := GenerateToken(testSecret, user, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("Generated token is empty")
	}

	// Parse the token back and verify claims
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(testSecret), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Token is not valid")
	}

	if claims.Subject != "user-gen" {
		t.Errorf("Expected subject 'user-gen', got '%s'", claims.Subject)
	}
	if claims.Email != "gen@test.com" {
		t.Errorf("Expected email 'gen@test.com', got '%s'", claims.Email)
	}
	if claims.Name != "Gen User" {
		t.Errorf("Expected name 'Gen User', got '%s'", claims.Name)
	}
	if claims.Role != models.RoleAdmin {
		t.Errorf("Expected role admin, got '%s'", claims.Role)
	}
	if claims.ProjectID != "proj-gen" {
		t.Errorf("Expected project_id 'proj-gen', got '%s'", claims.ProjectID)
	}
	if claims.Issuer != "zenith" {
		t.Errorf("Expected issuer 'zenith', got '%s'", claims.Issuer)
	}
}

func TestConstantTimeCompareEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected bool
	}{
		{"empty strings", "", "", true},
		{"one empty", "abc", "", false},
		{"other empty", "", "abc", false},
		{"same long strings", "abcdefghijklmnop", "abcdefghijklmnop", true},
		{"different lengths", "abc", "abcd", false},
		{"unicode same", "\u00e9", "\u00e9", true},
		{"unicode different", "\u00e9", "\u00e8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstantTimeCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("ConstantTimeCompare(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestRequestContext(t *testing.T) {
	app := setupAuthApp()
	app.Use(RequestContext())
	app.Get("/test", func(c *fiber.Ctx) error {
		requestID, _ := c.Locals("request_id").(string)
		return c.SendString("rid:" + requestID)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestJWTAuthCaseInsensitiveBearer(t *testing.T) {
	user := &models.User{
		ID:    "user-ci",
		Email: "ci@test.com",
		Role:  models.RoleDeveloper,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	// Test "bearer" (lowercase)
	app := setupAuthApp()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 for lowercase 'bearer', got %d", resp.StatusCode)
	}
}
