package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dotechhq/zenith/services/cli/internal/config"
)

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/apps" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("Unexpected auth header: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := New(&config.Config{
		APIBaseURL:  server.URL,
		AccessToken: "test-token",
	})

	var result map[string]string
	if err := c.Get("/api/v1/apps", &result); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", result["status"])
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected JSON content type, got %s", ct)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test-app" {
			t.Errorf("Expected name test-app, got %s", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "app-123"})
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})

	var result map[string]string
	if err := c.Post("/api/v1/apps", map[string]string{"name": "test-app"}, &result); err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	if result["id"] != "app-123" {
		t.Errorf("Expected id app-123, got %s", result["id"])
	}
}

func TestClientPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})
	if err := c.Put("/api/v1/apps/123/scale", map[string]int{"replicas": 3}, nil); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/apps/app-123" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})
	if err := c.Delete("/api/v1/apps/app-123"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})
	err := c.Get("/api/v1/admin/stats", nil)
	if err == nil {
		t.Fatal("Expected error for 403 response")
	}
	if got := err.Error(); got != "API error (403): forbidden" {
		t.Errorf("Unexpected error message: %s", got)
	}
}

func TestClientAPIErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "app not found"})
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})
	err := c.Get("/api/v1/apps/nope", nil)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}
	if got := err.Error(); got != "API error (404): app not found" {
		t.Errorf("Unexpected error message: %s", got)
	}
}

func TestClientAPIErrorRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})
	err := c.Get("/api/v1/bad", nil)
	if err == nil {
		t.Fatal("Expected error for 500 response")
	}
	if got := err.Error(); got != "API error (500): internal server error" {
		t.Errorf("Unexpected error message: %s", got)
	}
}

func TestClientNoAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Expected no auth header, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL})
	if err := c.Get("/health", nil); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
}

func TestClientNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	c := New(&config.Config{APIBaseURL: server.URL, AccessToken: "tok"})
	if err := c.Post("/api/v1/deploy", nil, nil); err != nil {
		t.Fatalf("Post with nil body failed: %v", err)
	}
}

func TestLogin(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", orig)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "user@test.com" || body["password"] != "secret" {
			t.Errorf("Unexpected login body: %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token":         "jwt-token-abc",
			"refresh_token": "refresh-xyz",
		})
	}))
	defer server.Close()

	cfg := &config.Config{APIBaseURL: server.URL}
	c := New(cfg)

	if err := c.Login("user@test.com", "secret"); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if cfg.AccessToken != "jwt-token-abc" {
		t.Errorf("Expected access token jwt-token-abc, got %s", cfg.AccessToken)
	}
	if cfg.RefreshToken != "refresh-xyz" {
		t.Errorf("Expected refresh token refresh-xyz, got %s", cfg.RefreshToken)
	}
}
