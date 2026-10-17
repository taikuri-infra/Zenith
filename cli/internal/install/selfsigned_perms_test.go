package install

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// The api container runs as a non-root user and must read zenith-ca.crt, so the
// certificates are world-readable even under a strict umask; the keys are not.
func TestWriteSelfSignedCertsModes(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "traefik", "certs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "traefik", "dynamic"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Domain: "zenith.lan"}
	if err := writeSelfSignedCerts(cfg, localRunner{}, dir); err != nil {
		t.Fatal(err)
	}
	want := map[string]os.FileMode{
		"zenith.crt":    0o644,
		"zenith-ca.crt": 0o644,
		"zenith.key":    0o600,
		"zenith-ca.key": 0o600,
	}
	for name, mode := range want {
		fi, err := os.Stat(filepath.Join(dir, "traefik", "certs", name))
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm(); got != mode {
			t.Errorf("%s mode = %o, want %o", name, got, mode)
		}
	}
}

// Every TLS mode writes the generated static config to the git-ignored
// traefik/traefik.local.yml and leaves the tracked traefik/traefik.yml alone
// (zen upgrade's git checkout must stay clean).
func TestWriteTraefikConfigLeavesTrackedFile(t *testing.T) {
	for _, mode := range []string{TLSHTTP01, TLSDNS01, TLSSelfSigned} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "traefik"), 0o755); err != nil {
				t.Fatal(err)
			}
			tracked := filepath.Join(dir, "traefik", "traefik.yml")
			const shipped = "# shipped\ncertificatesResolvers:\n  le: {}\n"
			if err := os.WriteFile(tracked, []byte(shipped), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg := &Config{Domain: "zenith.example.com", AdminEmail: "ops@example.com", TLSMode: mode}
			if err := composeWriteTLS(cfg, localRunner{}, dir); err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(tracked); string(b) != shipped {
				t.Errorf("traefik.yml was rewritten:\n%s", b)
			}
			b, err := os.ReadFile(filepath.Join(dir, "traefik", "traefik.local.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != generateTraefikStaticConfig(mode, "ops@example.com") {
				t.Errorf("traefik.local.yml content:\n%s", b)
			}
			acme := strings.Contains(string(b), "certificatesResolvers:")
			if acme != (mode != TLSSelfSigned) {
				t.Errorf("%s: certificatesResolvers present = %v\n%s", mode, acme, b)
			}
			if mode != TLSSelfSigned && !strings.Contains(string(b), `email: "ops@example.com"`) {
				t.Errorf("%s: ACME email missing\n%s", mode, b)
			}
		})
	}
}
