package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestListProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/projects" {
			t.Errorf("Expected path '/api/v1/projects', got '%s'", r.URL.Path)
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
}

func TestListApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "web-app", "status": "Running", "replicas": 2},
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
}

func TestListDatabases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "my-db", "engine": "postgresql", "status": "Ready"},
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
}
