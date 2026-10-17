//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

// TestInstall: the fresh install is up, standalone, the candidate version, closed to sign-ups, and zen can log in.
func TestInstall(t *testing.T) {
	if cfg.InstallSeconds <= 0 || cfg.InstallSeconds > 900 {
		t.Fatalf("install took %ds (limit 900)", cfg.InstallSeconds)
	}
	logf(t, "install took %ds", cfg.InstallSeconds)
	code, body := getURL(t, cfg.Endpoint+"/health")
	if code != 200 {
		t.Fatalf("/health %d %s", code, body)
	}
	code, body = getURL(t, cfg.Endpoint+"/")
	if code != 200 {
		t.Fatalf("dashboard %d %s", code, tail(body, 500))
	}
	info, err := api(t).ServerInfo()
	logf(t, "GET %s/api/v1/version → %+v %v", cfg.Endpoint, info, err)
	if err != nil || info.Mode != "standalone" {
		t.Fatalf("version: %+v %v", info, err)
	}
	if cfg.CandidateVersion != "" && info.Version != cfg.CandidateVersion {
		t.Fatalf("version %q, want %q", info.Version, cfg.CandidateVersion)
	}
	code, body = postJSON(t, cfg.Endpoint+"/api/v1/auth/register", `{"email":"x@y.z","password":"longenough","name":"x"}`)
	if code != 403 {
		t.Fatalf("register %d %s", code, body)
	}
	out, err := zen(t, "login", "--endpoint", cfg.Endpoint, "--email", cfg.AdminEmail)
	if err != nil || !strings.Contains(out, "Logged in") {
		t.Fatalf("zen login: %v\n%s", err, out)
	}
}
