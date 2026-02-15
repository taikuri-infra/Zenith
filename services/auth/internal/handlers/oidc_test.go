package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/auth/internal/crypto"
	"github.com/dotechhq/zenith/services/auth/internal/handlers"
	"github.com/dotechhq/zenith/services/auth/internal/models"
	"github.com/dotechhq/zenith/services/auth/internal/storage"
	"github.com/gofiber/fiber/v2"
)

const testSecret = "test-jwt-secret-key"
const testIssuer = "https://auth.zenith.dev"

func setupAuthApp() (*fiber.App, storage.Store) {
	store := storage.NewMemoryStore()
	app := fiber.New()

	oidcHandler := handlers.NewOIDCHandler(store, testSecret, testIssuer)
	realmHandler := handlers.NewRealmHandler(store)

	admin := app.Group("/admin")
	admin.Post("/realms", realmHandler.Create)
	admin.Get("/realms", realmHandler.List)
	admin.Get("/realms/:realm", realmHandler.Get)
	admin.Delete("/realms/:realm", realmHandler.Delete)
	admin.Post("/realms/:realm/clients", realmHandler.CreateClient)
	admin.Get("/realms/:realm/clients", realmHandler.ListClients)

	realms := app.Group("/realms/:realm")
	realms.Get("/.well-known/openid-configuration", oidcHandler.Discovery)
	realms.Post("/protocol/openid-connect/token", oidcHandler.Token)
	realms.Post("/register", oidcHandler.Register)

	return app, store
}

func createTestRealm(app *fiber.App) {
	body := `{"name":"test-realm","display_name":"Test Realm"}`
	req := httptest.NewRequest("POST", "/admin/realms", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)
}

func TestDiscovery(t *testing.T) {
	app, _ := setupAuthApp()
	createTestRealm(app)

	req := httptest.NewRequest("GET", "/realms/test-realm/.well-known/openid-configuration", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var doc map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&doc)

	issuer, _ := doc["issuer"].(string)
	if !strings.Contains(issuer, "test-realm") {
		t.Errorf("Expected issuer to contain realm, got '%s'", issuer)
	}
}

func TestRegister(t *testing.T) {
	app, _ := setupAuthApp()
	createTestRealm(app)

	body := `{"email":"user@test.com","password":"password123","name":"Test User"}`
	req := httptest.NewRequest("POST", "/realms/test-realm/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["email"] != "user@test.com" {
		t.Errorf("Expected email 'user@test.com', got '%v'", result["email"])
	}
}

func TestRegisterShortPassword(t *testing.T) {
	app, _ := setupAuthApp()
	createTestRealm(app)

	body := `{"email":"user@test.com","password":"short"}`
	req := httptest.NewRequest("POST", "/realms/test-realm/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for short password, got %d", resp.StatusCode)
	}
}

func TestPasswordLogin(t *testing.T) {
	app, store := setupAuthApp()
	createTestRealm(app)

	// Create user directly in store
	hash, _ := crypto.HashPassword("password123")
	store.CreateUser(&models.User{
		ID:           "user-1",
		RealmID:      "test-realm",
		Email:        "user@test.com",
		Name:         "Test",
		PasswordHash: hash,
		Roles:        []string{"user"},
	})

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", "user@test.com")
	form.Set("password", "password123")

	req := httptest.NewRequest("POST", "/realms/test-realm/protocol/openid-connect/token",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var token models.TokenResponse
	json.NewDecoder(resp.Body).Decode(&token)

	if token.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}
	if token.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}
	if token.IDToken == "" {
		t.Error("Expected non-empty ID token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("Expected token type 'Bearer', got '%s'", token.TokenType)
	}
}

func TestPasswordLoginInvalidCreds(t *testing.T) {
	app, store := setupAuthApp()
	createTestRealm(app)

	hash, _ := crypto.HashPassword("password123")
	store.CreateUser(&models.User{
		ID: "user-1", RealmID: "test-realm", Email: "user@test.com",
		PasswordHash: hash, Roles: []string{"user"},
	})

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", "user@test.com")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest("POST", "/realms/test-realm/protocol/openid-connect/token",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestUnsupportedGrantType(t *testing.T) {
	app, _ := setupAuthApp()
	createTestRealm(app)

	form := url.Values{}
	form.Set("grant_type", "invalid")

	req := httptest.NewRequest("POST", "/realms/test-realm/protocol/openid-connect/token",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateRealm(t *testing.T) {
	app, _ := setupAuthApp()

	body := `{"name":"my-realm","display_name":"My Realm"}`
	req := httptest.NewRequest("POST", "/admin/realms", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}

func TestListRealms(t *testing.T) {
	app, _ := setupAuthApp()
	createTestRealm(app)

	req := httptest.NewRequest("GET", "/admin/realms", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCreateClient(t *testing.T) {
	app, _ := setupAuthApp()
	createTestRealm(app)

	body := `{"name":"web-app","type":"public","redirect_uris":["http://localhost:3000/callback"]}`
	req := httptest.NewRequest("POST", "/admin/realms/test-realm/clients", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}
