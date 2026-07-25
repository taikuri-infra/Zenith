package install

import (
	"strings"
	"testing"
)

func stepNames(steps []Step) []string {
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	return names
}

func hasStep(steps []Step, name string) bool {
	for _, s := range steps {
		if s.Name == name {
			return true
		}
	}
	return false
}

func TestComposeSteps_HTTP01OwnDomain(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, Domain: "app.example.com", TLSMode: TLSHTTP01}
	steps := GetComposeInstallSteps(cfg)
	if !hasStep(steps, "Configure DNS") {
		t.Errorf("http01 own-domain should configure DNS; got %v", stepNames(steps))
	}
	if !hasStep(steps, "Configure TLS") {
		t.Errorf("http01 should configure TLS; got %v", stepNames(steps))
	}
}

func TestComposeSteps_DNS01SkipsDNSStep(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, Domain: "app.example.com", TLSMode: TLSDNS01, CloudflareToken: "tok"}
	steps := GetComposeInstallSteps(cfg)
	if hasStep(steps, "Configure DNS") {
		t.Errorf("dns01 must NOT run the public A-record step; got %v", stepNames(steps))
	}
	if !hasStep(steps, "Configure TLS") {
		t.Errorf("dns01 should configure TLS; got %v", stepNames(steps))
	}
}

func TestComposeSteps_SelfSignedSkipsDNSStep(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, Domain: "zenith.lan", TLSMode: TLSSelfSigned}
	steps := GetComposeInstallSteps(cfg)
	if hasStep(steps, "Configure DNS") {
		t.Errorf("self-signed must NOT configure public DNS; got %v", stepNames(steps))
	}
	if !hasStep(steps, "Configure TLS") {
		t.Errorf("self-signed should configure TLS; got %v", stepNames(steps))
	}
}

func TestComposeSteps_LocalhostHasNoTLS(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, Domain: "localhost"}
	steps := GetComposeInstallSteps(cfg)
	if hasStep(steps, "Configure TLS") || hasStep(steps, "Configure DNS") {
		t.Errorf("localhost install should have neither TLS nor DNS steps; got %v", stepNames(steps))
	}
}

func TestComposeSteps_FreeSubdomainConfiguresTLS(t *testing.T) {
	cfg := &Config{Edition: "compose", ComposeLocal: true, FreeSubdomain: true}
	steps := GetComposeInstallSteps(cfg)
	if !hasStep(steps, "Configure TLS") {
		t.Errorf("free subdomain should configure TLS (HTTP-01); got %v", stepNames(steps))
	}
	if hasStep(steps, "Configure DNS") {
		t.Errorf("free subdomain manages its own DNS via the register service; got %v", stepNames(steps))
	}
}

func TestBuildComposeEnv_DNS01IncludesToken(t *testing.T) {
	cfg := &Config{Domain: "app.example.com", TLSMode: TLSDNS01, CloudflareToken: "cf-secret-token"}
	env := buildComposeEnv(cfg, "admin@example.com", "999")
	if !strings.Contains(env, "CF_DNS_API_TOKEN=cf-secret-token") {
		t.Errorf("dns01 env must carry the Cloudflare token:\n%s", env)
	}
}

func TestBuildComposeEnv_HTTP01OmitsToken(t *testing.T) {
	cfg := &Config{Domain: "app.example.com", TLSMode: TLSHTTP01, CloudflareToken: "cf-secret-token"}
	env := buildComposeEnv(cfg, "admin@example.com", "999")
	if strings.Contains(env, "CF_DNS_API_TOKEN") {
		t.Errorf("http01 env must not carry a DNS token:\n%s", env)
	}
}
