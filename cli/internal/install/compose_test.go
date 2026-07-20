package install

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterAndReleaseSubdomain_Client(t *testing.T) {
	var sawRegister, sawRelease bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/register":
			sawRegister = true
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["ip"] == "" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"no ip"}`))
				return
			}
			_, _ = w.Write([]byte(`{"hostname":"swift-otter-abcd.apps.freezenith.com"}`))
		case "/release":
			sawRelease = true
			_, _ = w.Write([]byte(`{"status":"released"}`))
		}
	}))
	defer srv.Close()

	host, err := registerSubdomain(srv.URL, "203.0.113.5")
	if err != nil {
		t.Fatalf("registerSubdomain: %v", err)
	}
	if host != "swift-otter-abcd.apps.freezenith.com" {
		t.Errorf("hostname = %q", host)
	}
	if !sawRegister {
		t.Error("service /register was not called")
	}
	if err := releaseSubdomain(srv.URL, host); err != nil {
		t.Fatalf("releaseSubdomain: %v", err)
	}
	if !sawRelease {
		t.Error("service /release was not called")
	}
}

func TestGetComposeInstallSteps_Order(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true}
	steps := GetComposeInstallSteps(cfg)

	want := []string{
		"Connect",
		"Ensure Docker",
		"Fetch stack",
		"Start services",
		"Wait for health",
		"Save credentials",
	}
	if len(steps) != len(want) {
		t.Fatalf("got %d steps, want %d", len(steps), len(want))
	}
	for i, name := range want {
		if steps[i].Name != name {
			t.Errorf("step %d: got %q, want %q", i, steps[i].Name, name)
		}
	}
}

func TestGetComposeInstallSteps_FreeSubdomainAddsStep(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, FreeSubdomain: true}
	steps := GetComposeInstallSteps(cfg)
	if len(steps) != 7 {
		t.Fatalf("got %d steps, want 7 with --free-domain", len(steps))
	}
	if steps[1].Name != "Register subdomain" {
		t.Errorf("step 1 = %q, want %q", steps[1].Name, "Register subdomain")
	}
}

func TestComposeSteps_FreeSubdomainDryRun(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, FreeSubdomain: true, DryRun: true}
	for _, step := range GetComposeInstallSteps(cfg) {
		if err := step.Action(cfg); err != nil {
			t.Errorf("dry-run step %q failed: %v", step.Name, err)
		}
	}
	if !strings.HasSuffix(cfg.Domain, freeSubdomainBase) {
		t.Errorf("expected a free subdomain in dry-run, got %q", cfg.Domain)
	}
}

func TestComposeSteps_DryRunAllSucceed(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, DryRun: true}
	for _, step := range GetComposeInstallSteps(cfg) {
		if err := step.Action(cfg); err != nil {
			t.Errorf("dry-run step %q failed: %v", step.Name, err)
		}
	}
	// Fetch step must have generated an admin password even in dry-run.
	if cfg.AdminPassword == "" {
		t.Error("expected AdminPassword to be generated during dry-run")
	}
}

func TestBuildComposeEnv(t *testing.T) {
	cfg := &Config{AdminPassword: "secretpw"}
	env := buildComposeEnv(cfg, "admin@example.com", "991")

	mustContain := []string{
		"JWT_SECRET=", "ADMIN_EMAIL=admin@example.com",
		"ADMIN_PASSWORD=secretpw", "DB_PASSWORD=", "DOCKER_GID=991",
	}
	for _, s := range mustContain {
		if !strings.Contains(env, s) {
			t.Errorf("env missing %q\n%s", s, env)
		}
	}
	// No domain => no TLS/Caddy/Traefik vars.
	if strings.Contains(env, "ZENITH_DOMAIN=") {
		t.Error("expected no ZENITH_DOMAIN when domain is empty")
	}

	cfg.Domain = "app.example.com"
	env = buildComposeEnv(cfg, "admin@example.com", "991")
	for _, s := range []string{"ZENITH_DOMAIN=app.example.com", "ACME_EMAIL=admin@example.com", "https://app.example.com"} {
		if !strings.Contains(env, s) {
			t.Errorf("env with domain missing %q\n%s", s, env)
		}
	}
}

func TestGenerateSecret_NoShellOrComposeHostileChars(t *testing.T) {
	// Compose interpolates '$' in .env, and shells treat !@#$% specially — a
	// generated secret must contain none of them or it reaches the container
	// mangled and login breaks.
	for i := 0; i < 200; i++ {
		s := generateSecret(24)
		if len(s) != 24 {
			t.Fatalf("length = %d, want 24", len(s))
		}
		if strings.ContainsAny(s, "!@#$%^&*()`'\" ") {
			t.Fatalf("secret contains hostile char: %q", s)
		}
	}
}

func TestValidateInstallDir(t *testing.T) {
	ok := []string{"zenith", "opt/zenith", "my-app_1", "./zenith"}
	for _, d := range ok {
		if err := validateInstallDir(d); err != nil {
			t.Errorf("expected %q valid, got %v", d, err)
		}
	}
	bad := []string{"zen; rm -rf /", "a b", "$(whoami)", "a`b`", "a|b"}
	for _, d := range bad {
		if err := validateInstallDir(d); err == nil {
			t.Errorf("expected %q rejected", d)
		}
	}
}
