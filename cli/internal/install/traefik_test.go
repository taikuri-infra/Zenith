package install

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func assertValidYAML(t *testing.T, s string) {
	t.Helper()
	var out any
	if err := yaml.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("generated config is not valid YAML: %v\n%s", err, s)
	}
}

func TestGeneratedConfigsAreValidYAML(t *testing.T) {
	for _, mode := range []string{TLSHTTP01, TLSDNS01, TLSSelfSigned} {
		assertValidYAML(t, generateTraefikStaticConfig(mode, "admin@example.com"))
	}
	assertValidYAML(t, generateTraefikCertsConfig())
}

func TestGenerateTraefikStaticConfig_HTTP01(t *testing.T) {
	cfg := generateTraefikStaticConfig(TLSHTTP01, "admin@example.com")
	for _, want := range []string{"httpChallenge", "certResolver: le", "acme:", "admin@example.com", "/letsencrypt/acme.json"} {
		if !contains(cfg, want) {
			t.Errorf("http01 config missing %q\n%s", want, cfg)
		}
	}
	if contains(cfg, "dnsChallenge") {
		t.Error("http01 config should not contain dnsChallenge")
	}
}

func TestGenerateTraefikStaticConfig_DNS01(t *testing.T) {
	cfg := generateTraefikStaticConfig(TLSDNS01, "admin@example.com")
	for _, want := range []string{"dnsChallenge", "provider: cloudflare", "certResolver: le"} {
		if !contains(cfg, want) {
			t.Errorf("dns01 config missing %q\n%s", want, cfg)
		}
	}
	if contains(cfg, "httpChallenge") {
		t.Error("dns01 config should not contain httpChallenge")
	}
}

func TestGenerateTraefikStaticConfig_SelfSigned(t *testing.T) {
	cfg := generateTraefikStaticConfig(TLSSelfSigned, "admin@example.com")
	// No ACME at all for self-signed.
	if contains(cfg, "acme") || contains(cfg, "certResolver") || contains(cfg, "certificatesResolvers") {
		t.Errorf("self-signed config must not reference ACME/resolvers\n%s", cfg)
	}
	// Still needs the file provider (for the default cert) and the entrypoints.
	for _, want := range []string{"file:", "/etc/traefik/dynamic", "websecure", ":443"} {
		if !contains(cfg, want) {
			t.Errorf("self-signed config missing %q\n%s", want, cfg)
		}
	}
}

func TestGenerateTraefikCertsConfig(t *testing.T) {
	c := generateTraefikCertsConfig()
	for _, want := range []string{"defaultCertificate", "zenith.crt", "zenith.key"} {
		if !contains(c, want) {
			t.Errorf("certs config missing %q\n%s", want, c)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
