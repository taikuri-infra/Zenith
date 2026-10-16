package logout

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/cli/internal/config"
)

func logoutFixture(t *testing.T, token, refresh string) (path string, hits *int, srv *httptest.Server) {
	t.Helper()
	n := 0
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/logout" {
			n++
		}
	}))
	t.Cleanup(srv.Close)
	path = filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{APIEndpoint: srv.URL, Token: token, RefreshToken: refresh}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatal(err)
	}
	return path, &n, srv
}

func assertCleared(t *testing.T, path string) {
	t.Helper()
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "" || cfg.RefreshToken != "" || !cfg.TokenExpiresAt.IsZero() {
		t.Fatalf("session not cleared: %+v", cfg)
	}
}

func TestLogoutBareJWTCallsServer(t *testing.T) {
	path, hits, srv := logoutFixture(t, "eyJhbGciOiJIUzI1NiJ9.e30.sig", "")
	var out bytes.Buffer
	if err := runLogout(&out, path); err != nil {
		t.Fatal(err)
	}
	if *hits != 1 {
		t.Fatalf("logout endpoint hits = %d, want 1", *hits)
	}
	assertCleared(t, path)
	if !strings.Contains(out.String(), "Logged out of "+srv.URL) {
		t.Fatalf("out = %s", out.String())
	}
}

func TestLogoutSessionCallsServer(t *testing.T) {
	path, hits, _ := logoutFixture(t, "acc", "ref")
	if err := runLogout(&bytes.Buffer{}, path); err != nil {
		t.Fatal(err)
	}
	if *hits != 1 {
		t.Fatalf("logout endpoint hits = %d, want 1", *hits)
	}
	assertCleared(t, path)
}

func TestLogoutDeployTokenSkipsServer(t *testing.T) {
	path, hits, _ := logoutFixture(t, "znt_id_a:znt_sk_b", "")
	if err := runLogout(&bytes.Buffer{}, path); err != nil {
		t.Fatal(err)
	}
	if *hits != 0 {
		t.Fatalf("deploy token must not call logout, hits = %d", *hits)
	}
	assertCleared(t, path)
}
