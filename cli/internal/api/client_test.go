package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:8080", "test-token")
	if c.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected BaseURL 'http://localhost:8080', got '%s'", c.BaseURL)
	}
	if c.Token != "test-token" {
		t.Errorf("Expected Token 'test-token', got '%s'", c.Token)
	}
	if c.HTTPClient == nil {
		t.Error("Expected non-nil HTTPClient")
	}
}

func TestNewClient_EmptyToken(t *testing.T) {
	c := NewClient("http://localhost:8080", "")
	if c.Token != "" {
		t.Errorf("Expected empty Token, got '%s'", c.Token)
	}
	if c.HTTPClient == nil {
		t.Error("Expected non-nil HTTPClient even with empty token")
	}
}

func TestNewClient_Timeout(t *testing.T) {
	c := NewClient("http://localhost:8080", "token")
	if c.HTTPClient.Timeout.Seconds() != 30 {
		t.Errorf("Expected 30s timeout, got %v", c.HTTPClient.Timeout)
	}
}

func TestListProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/projects" {
			t.Errorf("Expected path '/api/v1/projects', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got '%s'", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"id": "1", "name": "project-1", "status": "active"},
				{"id": "2", "name": "project-2", "status": "active"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	projects, err := c.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	if projects[0].Name != "project-1" {
		t.Errorf("Expected project name 'project-1', got '%s'", projects[0].Name)
	}
}

func TestGetProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got '%s'", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/v1/projects/proj-1") {
			t.Errorf("Expected path ending with '/api/v1/projects/proj-1', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Project{
			ID:     "proj-1",
			Name:   "my-project",
			Status: "active",
			Region: "fsn1",
			Plan:   "pro",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	project, err := c.GetProject("proj-1")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if project.ID != "proj-1" {
		t.Errorf("Expected ID 'proj-1', got '%s'", project.ID)
	}
	if project.Name != "my-project" {
		t.Errorf("Expected Name 'my-project', got '%s'", project.Name)
	}
	if project.Region != "fsn1" {
		t.Errorf("Expected Region 'fsn1', got '%s'", project.Region)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "bad-token")
	_, err := c.ListProjects()
	if err == nil {
		t.Error("Expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "API error 401") {
		t.Errorf("Expected error to mention 'API error 401', got: %v", err)
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("Expected error body in message, got: %v", err)
	}
}

func TestAPIError_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"400 Bad Request", http.StatusBadRequest, "bad request"},
		{"403 Forbidden", http.StatusForbidden, "forbidden"},
		{"404 Not Found", http.StatusNotFound, "not found"},
		{"500 Server Error", http.StatusInternalServerError, "internal error"},
		{"502 Bad Gateway", http.StatusBadGateway, "bad gateway"},
		{"503 Unavailable", http.StatusServiceUnavailable, "service unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c := NewClient(server.URL, "test-token")
			_, err := c.ListProjects()
			if err == nil {
				t.Errorf("Expected error for %d response", tt.statusCode)
			}
			if !strings.Contains(err.Error(), tt.body) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.body, err)
			}
		})
	}
}

func TestListApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got '%s'", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/apps") {
			t.Errorf("Expected path to contain '/apps', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "web-app", "status": "Running", "replicas": 2, "image": "nginx:latest", "port": 8080},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	apps, err := c.ListApps("project-1")
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(apps))
	}
	if apps[0].Name != "web-app" {
		t.Errorf("Expected app name 'web-app', got '%s'", apps[0].Name)
	}
	if apps[0].Status != "Running" {
		t.Errorf("Expected status 'Running', got '%s'", apps[0].Status)
	}
}

func TestListApps_URLConstruction(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	c.ListApps("my-project")

	expectedPath := "/api/v1/projects/my-project/apps"
	if capturedPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, capturedPath)
	}
}

func TestCreateApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/apps") {
			t.Errorf("Expected path to contain '/apps', got '%s'", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Read and verify request body
		body, _ := io.ReadAll(r.Body)
		var app App
		json.Unmarshal(body, &app)
		if app.Name != "new-app" {
			t.Errorf("Expected app name 'new-app', got '%s'", app.Name)
		}
		if app.Image != "nginx:latest" {
			t.Errorf("Expected image 'nginx:latest', got '%s'", app.Image)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(App{
			Name:     "new-app",
			Image:    "nginx:latest",
			Replicas: 1,
			Port:     8080,
			Status:   "Creating",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	app, err := c.CreateApp("project-1", &App{
		Name:     "new-app",
		Image:    "nginx:latest",
		Replicas: 1,
		Port:     8080,
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	if app.Name != "new-app" {
		t.Errorf("Expected app name 'new-app', got '%s'", app.Name)
	}
	if app.Status != "Creating" {
		t.Errorf("Expected status 'Creating', got '%s'", app.Status)
	}
}

func TestCreateApp_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("app already exists"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	_, err := c.CreateApp("project-1", &App{Name: "existing-app"})
	if err == nil {
		t.Error("Expected error for 409 response")
	}
}

func TestRedeploy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}
		expectedPath := "/api/v1/projects/project-1/apps/web-app/redeploy"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	err := c.Redeploy("project-1", "web-app")
	if err != nil {
		t.Fatalf("Redeploy failed: %v", err)
	}
}

func TestRedeploy_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("app not found"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	err := c.Redeploy("project-1", "nonexistent-app")
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestListDatabases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got '%s'", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "my-db", "engine": "postgresql", "status": "Ready", "version": "16", "storage": "20Gi"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	dbs, err := c.ListDatabases("project-1")
	if err != nil {
		t.Fatalf("ListDatabases failed: %v", err)
	}
	if len(dbs) != 1 {
		t.Errorf("Expected 1 database, got %d", len(dbs))
	}
	if dbs[0].Engine != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%s'", dbs[0].Engine)
	}
}

func TestListDatabases_URLConstruction(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	c.ListDatabases("my-project")

	expectedPath := "/api/v1/projects/my-project/databases"
	if capturedPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, capturedPath)
	}
}

func TestCreateDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var db Database
		json.Unmarshal(body, &db)
		if db.Name != "new-db" {
			t.Errorf("Expected db name 'new-db', got '%s'", db.Name)
		}
		if db.Engine != "postgresql" {
			t.Errorf("Expected engine 'postgresql', got '%s'", db.Engine)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Database{
			Name:    "new-db",
			Engine:  "postgresql",
			Version: "16",
			Storage: "20Gi",
			Status:  "Creating",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	db, err := c.CreateDatabase("project-1", &Database{
		Name:    "new-db",
		Engine:  "postgresql",
		Version: "16",
		Storage: "20Gi",
	})
	if err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}
	if db.Name != "new-db" {
		t.Errorf("Expected db name 'new-db', got '%s'", db.Name)
	}
	if db.Status != "Creating" {
		t.Errorf("Expected status 'Creating', got '%s'", db.Status)
	}
}

func TestGetDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got '%s'", r.Method)
		}
		expectedPath := "/api/v1/projects/project-1/databases/my-db"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Database{
			Name:    "my-db",
			Engine:  "postgresql",
			Version: "16",
			Status:  "Ready",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	db, err := c.GetDatabase("project-1", "my-db")
	if err != nil {
		t.Fatalf("GetDatabase failed: %v", err)
	}
	if db.Name != "my-db" {
		t.Errorf("Expected db name 'my-db', got '%s'", db.Name)
	}
	if db.Engine != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%s'", db.Engine)
	}
}

func TestGetDatabase_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("database not found"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	_, err := c.GetDatabase("project-1", "nonexistent")
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestAuthorizationHeader(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer server.Close()

	c := NewClient(server.URL, "my-secret-token")
	c.ListProjects()

	expectedAuth := "Bearer my-secret-token"
	if capturedAuth != expectedAuth {
		t.Errorf("Expected Authorization '%s', got '%s'", expectedAuth, capturedAuth)
	}
}

func TestAuthorizationHeader_EmptyToken(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer server.Close()

	c := NewClient(server.URL, "")
	c.ListProjects()

	// When token is empty, Authorization header should not be set
	if capturedAuth != "" {
		t.Errorf("Expected empty Authorization header for empty token, got '%s'", capturedAuth)
	}
}

func TestContentTypeHeader(t *testing.T) {
	var capturedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(App{Name: "test"})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	c.CreateApp("project-1", &App{Name: "test"})

	if capturedContentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", capturedContentType)
	}
}

func TestProject_StructFields(t *testing.T) {
	p := Project{
		ID:          "proj-1",
		Name:        "my-project",
		DisplayName: "My Project",
		Owner:       "user-1",
		Plan:        "pro",
		Region:      "fsn1",
		Status:      "active",
		CreatedAt:   "2026-01-15T00:00:00Z",
	}

	if p.ID != "proj-1" {
		t.Errorf("Expected ID 'proj-1', got '%s'", p.ID)
	}
	if p.DisplayName != "My Project" {
		t.Errorf("Expected DisplayName 'My Project', got '%s'", p.DisplayName)
	}
	if p.Owner != "user-1" {
		t.Errorf("Expected Owner 'user-1', got '%s'", p.Owner)
	}
}

func TestApp_StructFields(t *testing.T) {
	app := App{
		Name:     "web-app",
		Image:    "nginx:1.25",
		Replicas: 3,
		Port:     8080,
		Status:   "Running",
		CPU:      "500m",
		Memory:   "256Mi",
	}

	if app.Name != "web-app" {
		t.Errorf("Expected Name 'web-app', got '%s'", app.Name)
	}
	if app.Replicas != 3 {
		t.Errorf("Expected Replicas 3, got %d", app.Replicas)
	}
	if app.CPU != "500m" {
		t.Errorf("Expected CPU '500m', got '%s'", app.CPU)
	}
}

func TestDatabase_StructFields(t *testing.T) {
	db := Database{
		Name:             "my-db",
		Engine:           "postgresql",
		Version:          "16",
		Storage:          "20Gi",
		Status:           "Ready",
		ConnectionString: "postgresql://localhost:5432/mydb",
		Port:             5432,
	}

	if db.Name != "my-db" {
		t.Errorf("Expected Name 'my-db', got '%s'", db.Name)
	}
	if db.ConnectionString != "postgresql://localhost:5432/mydb" {
		t.Errorf("Expected ConnectionString, got '%s'", db.ConnectionString)
	}
	if db.Port != 5432 {
		t.Errorf("Expected Port 5432, got %d", db.Port)
	}
}

func TestListApps_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []interface{}{},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	apps, err := c.ListApps("project-1")
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("Expected 0 apps, got %d", len(apps))
	}
}

func TestListDatabases_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []interface{}{},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	dbs, err := c.ListDatabases("project-1")
	if err != nil {
		t.Fatalf("ListDatabases failed: %v", err)
	}
	if len(dbs) != 0 {
		t.Errorf("Expected 0 databases, got %d", len(dbs))
	}
}

func TestClient_Login_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth/login" {
			http.Error(w, "unexpected request", 404)
			return
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["email"] == "" || body["password"] == "" {
			http.Error(w, "missing fields", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "jwt.token.here",
			"refresh_token": "refresh.token.here",
			"token_type":    "bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	token, err := c.Login("admin@example.com", "secret")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if token != "jwt.token.here" {
		t.Errorf("Expected access_token 'jwt.token.here', got %q", token)
	}
}

func TestClient_Login_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"invalid credentials"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Login("bad@example.com", "wrong")
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}

func TestListApps_MultipleApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "app-1", "image": "nginx:latest", "replicas": 2, "port": 8080, "status": "Running"},
				{"name": "app-2", "image": "redis:latest", "replicas": 1, "port": 6379, "status": "Running"},
				{"name": "app-3", "image": "postgres:16", "replicas": 1, "port": 5432, "status": "Pending"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	apps, err := c.ListApps("project-1")
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(apps) != 3 {
		t.Errorf("Expected 3 apps, got %d", len(apps))
	}
}
