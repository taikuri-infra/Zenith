package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatabasesUseStandaloneEndpoints(t *testing.T) {
	var gotPath, gotQuery string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects" {
			_ = json.NewEncoder(w).Encode(json.RawMessage(`{"items":[{"id":"proj-uuid-1","name":"default"}]}`))
			return
		}
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(json.RawMessage(`{"name":"pg","engine":"postgresql","status":"provisioning","port":5432}`))
			return
		}
		_ = json.NewEncoder(w).Encode(json.RawMessage(`[{"name":"pg","engine":"postgresql","status":"ready","port":5432}]`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "tok")

	dbs, err := c.ListDatabases("default")
	if err != nil || len(dbs) != 1 || gotPath != "/api/v1/databases" || gotQuery != "project_id=proj-uuid-1" {
		t.Fatalf("list: %v %v path=%s q=%s", dbs, err, gotPath, gotQuery)
	}
	if _, err := c.CreateDatabase("default", &Database{Name: "pg", Engine: "postgresql"}); err != nil || gotPath != "/api/v1/databases" || body["project_id"] != "proj-uuid-1" {
		t.Fatalf("create: %v path=%s body=%v", err, gotPath, body)
	}
}

func TestDatabasesUnresolvableProject(t *testing.T) {
	var body map[string]any
	dbCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/projects" {
			_ = json.NewEncoder(w).Encode(json.RawMessage(`{"items":[]}`))
			return
		}
		dbCalls++
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		_ = json.NewEncoder(w).Encode(json.RawMessage(`{"name":"pg"}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "tok")
	dbs, err := c.ListDatabases("nope")
	if err != nil || len(dbs) != 0 || dbCalls != 0 {
		t.Fatalf("list: %v %v calls=%d", dbs, err, dbCalls)
	}
	if _, err := c.CreateDatabase("nope", &Database{Name: "pg", Engine: "postgresql"}); err != nil {
		t.Fatal(err)
	}
	if _, has := body["project_id"]; has {
		t.Errorf("project_id must be omitted, got %v", body)
	}
}
