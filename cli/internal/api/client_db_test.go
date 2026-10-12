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
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = w.Write([]byte(`{"name":"pg","engine":"postgresql","status":"provisioning","port":5432}`))
			return
		}
		_, _ = w.Write([]byte(`[{"name":"pg","engine":"postgresql","status":"ready","port":5432}]`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "tok")

	dbs, err := c.ListDatabases("proj-1")
	if err != nil || len(dbs) != 1 || gotPath != "/api/v1/databases" || gotQuery != "project_id=proj-1" {
		t.Fatalf("list: %v %v path=%s q=%s", dbs, err, gotPath, gotQuery)
	}
	if _, err := c.CreateDatabase("proj-1", &Database{Name: "pg", Engine: "postgresql"}); err != nil || gotPath != "/api/v1/databases" || body["project_id"] != "proj-1" {
		t.Fatalf("create: %v path=%s body=%v", err, gotPath, body)
	}
}
