package db

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
)

type fakeAPI struct {
	created   map[string]any
	listQuery string
}

func (f *fakeAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/projects":
			_, _ = w.Write([]byte(`{"items":[{"id":"p-1","name":"shop"},{"id":"p-2","name":"blog"}]}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/databases":
			f.listQuery = r.URL.Query().Get("project_id")
			_, _ = w.Write([]byte(`[{"id":"db-1","name":"main","engine":"postgresql","status":"ready","port":5432},
				{"id":"db-2","name":"cache","engine":"redis","status":"provisioning","port":6379}]`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/databases":
			_ = json.NewDecoder(r.Body).Decode(&f.created)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"db-3","name":"orders","engine":"mysql","status":"provisioning"}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/databases/db-1":
			_, _ = w.Write([]byte(`{"id":"db-1","name":"main","engine":"postgresql","status":"ready","host_port":49153,"connection_string":"postgresql://u:p@zenith-db-db1:5432/main"}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestListDatabases(t *testing.T) {
	f := &fakeAPI{}
	srv := f.server(t)
	defer srv.Close()

	var out bytes.Buffer
	if err := listDatabases(&out, cliapi.NewClient(srv.URL, "tok"), "blog"); err != nil {
		t.Fatal(err)
	}
	if f.listQuery != "p-2" {
		t.Errorf("listed project %q, want p-2", f.listQuery)
	}
	s := out.String()
	for _, want := range []string{"main", "postgresql", "ready", "cache", "redis", "provisioning"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "No databases found") {
		t.Errorf("output claims no databases:\n%s", s)
	}
}

func TestListDatabasesDefaultsToFirstProject(t *testing.T) {
	f := &fakeAPI{}
	srv := f.server(t)
	defer srv.Close()
	var out bytes.Buffer
	if err := listDatabases(&out, cliapi.NewClient(srv.URL, "tok"), ""); err != nil {
		t.Fatal(err)
	}
	if f.listQuery != "p-1" {
		t.Errorf("listed project %q, want the first project p-1", f.listQuery)
	}
}

func TestCreateDatabase(t *testing.T) {
	f := &fakeAPI{}
	srv := f.server(t)
	defer srv.Close()

	var out bytes.Buffer
	if err := createDatabase(&out, cliapi.NewClient(srv.URL, "tok"), "shop", "orders", "mysql"); err != nil {
		t.Fatal(err)
	}
	if f.created["name"] != "orders" || f.created["engine"] != "mysql" || f.created["project_id"] != "p-1" {
		t.Errorf("create body = %v", f.created)
	}
	if !strings.Contains(out.String(), "orders") {
		t.Errorf("output = %q", out.String())
	}
}

func TestCreateDatabaseRequiresName(t *testing.T) {
	f := &fakeAPI{}
	srv := f.server(t)
	defer srv.Close()
	if err := createDatabase(&bytes.Buffer{}, cliapi.NewClient(srv.URL, "tok"), "shop", "", "postgresql"); err == nil {
		t.Error("want an error without --name")
	}
	if f.created != nil {
		t.Error("API must not be called without a name")
	}
}

func TestConnectPrintsConnectionString(t *testing.T) {
	f := &fakeAPI{}
	srv := f.server(t)
	defer srv.Close()

	var out bytes.Buffer
	if err := connectDatabase(&out, cliapi.NewClient(srv.URL, "tok"), "shop", "main", connectOpts{apiEndpoint: "https://zenith.example.com"}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "postgresql://u:p@zenith-db-db1:5432/main") {
		t.Errorf("output missing connection string:\n%s", s)
	}
	for _, want := range []string{"Inside the app network:", "From your machine:",
		"ssh -N -L 5432:127.0.0.1:49153 root@zenith.example.com", "postgresql://u:p@127.0.0.1:5432/main"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
	if err := connectDatabase(&bytes.Buffer{}, cliapi.NewClient(srv.URL, "tok"), "shop", "nope", connectOpts{}); err == nil {
		t.Error("want an error for an unknown database")
	}
}
