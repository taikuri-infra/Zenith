package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
	"github.com/gofiber/fiber/v2"
)

// --- RequireInternalSecret ---

func TestRequireInternalSecretValid(t *testing.T) {
	app := fiber.New()
	app.Use(RequireInternalSecret("my-secret"))
	app.Get("/internal", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/internal", nil)
	req.Header.Set("X-Internal-Secret", "my-secret")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireInternalSecretInvalid(t *testing.T) {
	app := fiber.New()
	app.Use(RequireInternalSecret("my-secret"))
	app.Get("/internal", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/internal", nil)
	req.Header.Set("X-Internal-Secret", "wrong-secret")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireInternalSecretMissing(t *testing.T) {
	app := fiber.New()
	app.Use(RequireInternalSecret("my-secret"))
	app.Get("/internal", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/internal", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

// --- DeployTokenAuthFunc ---

func TestDeployTokenAuthFuncValid(t *testing.T) {
	dt := &entities.DeployToken{
		ID:        "dt-1",
		UserID:    "user-deploy",
		ProjectID: "proj-deploy",
		Scopes:    []string{"deploy:app"},
	}

	validate := func(tokenID, secret string) (*entities.DeployToken, error) {
		if tokenID == "znt_id_abc" && secret == "znt_sk_xyz" {
			return dt, nil
		}
		return nil, errors.New("invalid")
	}

	var capturedUserID, capturedProjectID string
	var capturedRole entities.Role

	app := fiber.New()
	app.Use(DeployTokenAuthFunc(validate))
	app.Get("/deploy", func(c *fiber.Ctx) error {
		capturedUserID, _ = c.Locals("user_id").(string)
		capturedProjectID, _ = c.Locals("project_id").(string)
		capturedRole, _ = c.Locals("role").(entities.Role)
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/deploy", nil)
	req.Header.Set("Authorization", "DeployToken znt_id_abc:znt_sk_xyz")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	if capturedUserID != "user-deploy" {
		t.Errorf("Expected user_id 'user-deploy', got '%s'", capturedUserID)
	}
	if capturedProjectID != "proj-deploy" {
		t.Errorf("Expected project_id 'proj-deploy', got '%s'", capturedProjectID)
	}
	if capturedRole != entities.RoleDeveloper {
		t.Errorf("Expected role 'developer', got '%s'", capturedRole)
	}
}

func TestDeployTokenAuthFuncInvalidCredentials(t *testing.T) {
	validate := func(tokenID, secret string) (*entities.DeployToken, error) {
		return nil, errors.New("invalid or expired")
	}

	app := fiber.New()
	app.Use(DeployTokenAuthFunc(validate))
	app.Get("/deploy", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/deploy", nil)
	req.Header.Set("Authorization", "DeployToken bad_id:bad_secret")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestDeployTokenAuthFuncInvalidFormat(t *testing.T) {
	validate := func(tokenID, secret string) (*entities.DeployToken, error) {
		return nil, errors.New("should not be called")
	}

	app := fiber.New()
	app.Use(DeployTokenAuthFunc(validate))
	app.Get("/deploy", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Missing colon separator
	req := httptest.NewRequest("GET", "/deploy", nil)
	req.Header.Set("Authorization", "DeployToken no-colon-here")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestDeployTokenAuthFuncSkipsNonDeployTokenAuth(t *testing.T) {
	validate := func(tokenID, secret string) (*entities.DeployToken, error) {
		return nil, errors.New("should not be called")
	}

	app := fiber.New()
	app.Use(DeployTokenAuthFunc(validate))
	app.Get("/deploy", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Bearer token should pass through (not DeployToken)
	req := httptest.NewRequest("GET", "/deploy", nil)
	req.Header.Set("Authorization", "Bearer some-jwt")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (pass through), got %d", resp.StatusCode)
	}
}

func TestDeployTokenAuthFuncSkipsNoAuthHeader(t *testing.T) {
	validate := func(tokenID, secret string) (*entities.DeployToken, error) {
		return nil, errors.New("should not be called")
	}

	app := fiber.New()
	app.Use(DeployTokenAuthFunc(validate))
	app.Get("/deploy", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/deploy", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (pass through), got %d", resp.StatusCode)
	}
}

func TestDeployTokenAuthFuncSkipsAlreadyAuthenticated(t *testing.T) {
	validate := func(tokenID, secret string) (*entities.DeployToken, error) {
		return nil, errors.New("should not be called")
	}

	app := fiber.New()
	// Simulate prior authentication
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "already-authed")
		return c.Next()
	})
	app.Use(DeployTokenAuthFunc(validate))
	app.Get("/deploy", func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		return c.SendString("user:" + userID)
	})

	req := httptest.NewRequest("GET", "/deploy", nil)
	req.Header.Set("Authorization", "DeployToken id:secret")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// --- RequireDeployScope ---

func TestRequireDeployScopeHasScope(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("deploy_token", &entities.DeployToken{
			Scopes: []string{"deploy:app", "deploy:env"},
		})
		return c.Next()
	})
	app.Use(RequireDeployScope("deploy:app"))
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

func TestRequireDeployScopeMissingScope(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("deploy_token", &entities.DeployToken{
			Scopes: []string{"deploy:env"},
		})
		return c.Next()
	})
	app.Use(RequireDeployScope("deploy:app"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireDeployScopeWildcard(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("deploy_token", &entities.DeployToken{
			Scopes: []string{"infra:*"},
		})
		return c.Next()
	})
	app.Use(RequireDeployScope("deploy:app"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	// infra:* is the wildcard for deploy tokens (ScopeInfraAll)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (infra:* wildcard), got %d", resp.StatusCode)
	}
}

func TestRequireDeployScopeNotDeployToken(t *testing.T) {
	// When no deploy_token is in Locals, scope check should be skipped
	app := fiber.New()
	app.Use(RequireDeployScope("deploy:app"))
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

func TestRequireDeployScopeEmptyScopes(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("deploy_token", &entities.DeployToken{
			Scopes: []string{},
		})
		return c.Next()
	})
	app.Use(RequireDeployScope("deploy:app"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for empty scopes, got %d", resp.StatusCode)
	}
}

// --- JWTAuth with TokenBlacklist ---

func TestJWTAuthWithBlacklistedToken(t *testing.T) {
	user := &entities.User{
		ID:    "user-bl",
		Email: "bl@test.com",
		Role:  entities.RoleDeveloper,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	bl := NewTokenBlacklist()
	defer bl.Stop()
	bl.Revoke(token, time.Now().Add(1*time.Hour))

	app := fiber.New()
	app.Use(JWTAuth(testSecret, bl))
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
		t.Errorf("Expected 401 for blacklisted token, got %d", resp.StatusCode)
	}
}

func TestJWTAuthWithNonBlacklistedToken(t *testing.T) {
	user := &entities.User{
		ID:    "user-ok",
		Email: "ok@test.com",
		Role:  entities.RoleDeveloper,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	bl := NewTokenBlacklist()
	defer bl.Stop()
	// Blacklist a different token
	bl.Revoke("some-other-token", time.Now().Add(1*time.Hour))

	app := fiber.New()
	app.Use(JWTAuth(testSecret, bl))
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
		t.Errorf("Expected 200 for non-blacklisted token, got %d", resp.StatusCode)
	}
}

// --- JWTAuth with team member token (AccountID/MemberID) ---

func TestJWTAuthTeamMemberToken(t *testing.T) {
	user := &entities.User{
		ID:        "member-user-1",
		Email:     "member@test.com",
		Name:      "Team Member",
		Role:      entities.RoleDeveloper,
		ProjectID: "proj-team",
	}

	overrides := zenithJWT.TeamMemberOverrides{
		AccountID: "owner-user-1",
		MemberID:  "tm-001",
		Role:      entities.RoleDeveloper,
	}

	token, err := zenithJWT.GenerateTeamMemberToken(testSecret, user, 1*time.Hour, overrides)
	if err != nil {
		t.Fatalf("Failed to generate team member token: %v", err)
	}

	var capturedUserID, capturedMemberID string

	app := fiber.New()
	app.Use(JWTAuth(testSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedUserID, _ = c.Locals("user_id").(string)
		capturedMemberID, _ = c.Locals("member_id").(string)
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

	// When AccountID is set, user_id should be the owner's ID (AccountID trick)
	if capturedUserID != "owner-user-1" {
		t.Errorf("Expected user_id 'owner-user-1' (account_id), got '%s'", capturedUserID)
	}
	if capturedMemberID != "tm-001" {
		t.Errorf("Expected member_id 'tm-001', got '%s'", capturedMemberID)
	}
}

// --- RequireAuth with blacklist ---

func TestRequireAuthWithBlacklist(t *testing.T) {
	user := &entities.User{
		ID:    "user-rbl",
		Email: "rbl@test.com",
		Role:  entities.RoleDeveloper,
	}

	token, _ := GenerateToken(testSecret, user, 1*time.Hour)

	bl := NewTokenBlacklist()
	defer bl.Stop()
	bl.Revoke(token, time.Now().Add(1*time.Hour))

	app := fiber.New()
	app.Use(RequireAuth(testSecret, bl))
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
		t.Errorf("Expected 401 for blacklisted token via RequireAuth, got %d", resp.StatusCode)
	}
}

// --- RequestContext with requestid set ---

func TestRequestContextWithRequestID(t *testing.T) {
	app := fiber.New()
	// Simulate requestid middleware setting Locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "rid-abc-123")
		return c.Next()
	})
	app.Use(RequestContext())

	var capturedRequestID string

	app.Get("/test", func(c *fiber.Ctx) error {
		capturedRequestID, _ = c.Locals("request_id").(string)
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

	if capturedRequestID != "rid-abc-123" {
		t.Errorf("Expected request_id 'rid-abc-123', got '%s'", capturedRequestID)
	}
}

func TestRequestContextWithoutRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestContext())

	var capturedRequestID string

	app.Get("/test", func(c *fiber.Ctx) error {
		capturedRequestID, _ = c.Locals("request_id").(string)
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

	if capturedRequestID != "" {
		t.Errorf("Expected empty request_id, got '%s'", capturedRequestID)
	}
}

// --- ParseToken ---

func TestParseTokenValid(t *testing.T) {
	user := &entities.User{
		ID:    "user-parse",
		Email: "parse@test.com",
		Name:  "Parse User",
		Role:  entities.RoleAdmin,
	}

	tokenStr, err := GenerateToken(testSecret, user, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ParseToken(testSecret, tokenStr)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.Subject != "user-parse" {
		t.Errorf("Expected subject 'user-parse', got '%s'", claims.Subject)
	}
	if claims.Email != "parse@test.com" {
		t.Errorf("Expected email 'parse@test.com', got '%s'", claims.Email)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	_, err := ParseToken(testSecret, "not-a-valid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestParseTokenWrongSecret(t *testing.T) {
	user := &entities.User{
		ID:    "user-ws",
		Email: "ws@test.com",
		Role:  entities.RoleDeveloper,
	}

	tokenStr, _ := GenerateToken("secret-a", user, 1*time.Hour)
	_, err := ParseToken("secret-b", tokenStr)
	if err == nil {
		t.Error("Expected error for wrong secret")
	}
}

// --- RequireRole with customer role ---

func TestRequireRoleCustomerLowest(t *testing.T) {
	tests := []struct {
		name     string
		minRole  entities.Role
		wantCode int
	}{
		{"customer can access customer", entities.RoleCustomer, 200},
		{"customer cannot access viewer", entities.RoleViewer, 403},
		{"customer cannot access developer", entities.RoleDeveloper, 403},
		{"customer cannot access admin", entities.RoleAdmin, 403},
		{"customer cannot access owner", entities.RoleOwner, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("role", entities.RoleCustomer)
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

// --- RequireRole with invalid role type ---

func TestRequireRoleWithStringInsteadOfRole(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		// Set role as plain string instead of entities.Role
		c.Locals("role", "admin")
		return c.Next()
	})
	app.Use(RequireRole(entities.RoleViewer))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	// Type assertion to entities.Role will fail for a plain string
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for wrong type, got %d", resp.StatusCode)
	}
}
