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
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(json.RawMessage(`{"error":"unauthorized"}`))
			return
		}
		switch r.URL.Path {
		case "/register":
			sawRegister = true
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["ip"] == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(json.RawMessage(`{"error":"no ip"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(json.RawMessage(`{"hostname":"swift-otter-abcd.apps.freezenith.com"}`))
		case "/release":
			sawRelease = true
			_ = json.NewEncoder(w).Encode(json.RawMessage(`{"status":"released"}`))
		}
	}))
	defer srv.Close()

	host, err := registerSubdomain(srv.URL, "test-token", "203.0.113.5")
	if err != nil {
		t.Fatalf("registerSubdomain: %v", err)
	}
	if host != "swift-otter-abcd.apps.freezenith.com" {
		t.Errorf("hostname = %q", host)
	}
	if !sawRegister {
		t.Error("service /register was not called")
	}
	if err := releaseSubdomain(srv.URL, "test-token", host); err != nil {
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
	// Base 6 + "Register subdomain" + "Configure TLS" (free subdomain is HTTP-01).
	if len(steps) != 8 {
		t.Fatalf("got %d steps, want 8 with --free-domain", len(steps))
	}
	if steps[1].Name != "Register subdomain" {
		t.Errorf("step 1 = %q, want %q", steps[1].Name, "Register subdomain")
	}
}

func TestGetComposeInstallSteps_CustomDomainAddsDNSStep(t *testing.T) {
	has := func(cfg *Config, name string) bool {
		for _, s := range GetComposeInstallSteps(cfg) {
			if s.Name == name {
				return true
			}
		}
		return false
	}
	if !has(&Config{Edition: "compose", ComposeLocal: true, Domain: "app.example.com"}, "Configure DNS") {
		t.Error("a custom domain should add the 'Configure DNS' step")
	}
	if has(&Config{Edition: "compose", ComposeLocal: true, Domain: "localhost"}, "Configure DNS") {
		t.Error("localhost must not add Configure DNS")
	}
	if has(&Config{Edition: "compose", ComposeLocal: true, FreeSubdomain: true}, "Configure DNS") {
		t.Error("a free subdomain must not add Configure DNS")
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
	// ACME_EMAIL is no longer an env var — Traefik reads it from the generated
	// traefik.local.yml. The domain-derived URL vars remain.
	for _, s := range []string{"ZENITH_DOMAIN=app.example.com", "https://app.example.com"} {
		if !strings.Contains(env, s) {
			t.Errorf("env with domain missing %q\n%s", s, env)
		}
	}
	if strings.Contains(env, "ACME_EMAIL=") {
		t.Error("ACME_EMAIL should no longer be written to .env (it lives in traefik.local.yml)")
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

func TestBuildComposeEnv_SelfSignedWritesTLSKeys(t *testing.T) {
	cfg := &Config{Domain: "zenith.lan", AdminPassword: "pw", TLSMode: TLSSelfSigned}
	env := buildComposeEnv(cfg, "admin@zenith.lan", "999")
	for _, want := range []string{"ZENITH_TLS_MODE=selfsigned\n", "ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt\n", "ZENITH_CERT_RESOLVER=\n", "ZENITH_TRAEFIK_CONFIG=/etc/traefik/traefik.local.yml\n"} {
		if !strings.Contains(env, want) {
			t.Errorf("missing %q in:\n%s", want, env)
		}
	}
	plain := buildComposeEnv(&Config{Domain: "zenith.example.com", AdminPassword: "pw"}, "a@b.c", "999")
	for _, no := range []string{"ZENITH_TLS_MODE", "ZENITH_TLS_CA_FILE", "ZENITH_CERT_RESOLVER"} {
		if strings.Contains(plain, no) {
			t.Errorf("http01 env must not contain %s", no)
		}
	}
}

// Every install that configures TLS points Traefik at the generated
// traefik.local.yml; a localhost install (no TLS step) keeps the shipped file.
func TestBuildComposeEnv_TraefikConfigKey(t *testing.T) {
	const key = "ZENITH_TRAEFIK_CONFIG=/etc/traefik/traefik.local.yml\n"
	for _, cfg := range []*Config{
		{Domain: "zenith.example.com", TLSMode: TLSHTTP01},
		{Domain: "zenith.example.com", TLSMode: TLSDNS01},
		{Domain: "zenith.lan", TLSMode: TLSSelfSigned},
		{Domain: "x.apps.freezenith.com", FreeSubdomain: true},
	} {
		if env := buildComposeEnv(cfg, "a@b.c", "999"); strings.Count(env, key) != 1 {
			t.Errorf("%+v: want %q once in:\n%s", cfg, key, env)
		}
	}
	for _, d := range []string{"", "localhost"} {
		if env := buildComposeEnv(&Config{Domain: d}, "a@b.c", "999"); strings.Contains(env, "ZENITH_TRAEFIK_CONFIG") {
			t.Errorf("domain %q must not set ZENITH_TRAEFIK_CONFIG:\n%s", d, env)
		}
	}
}
