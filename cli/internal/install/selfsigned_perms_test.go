package install

import (
	"os"
	"path/filepath"
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

// Self-signed installs write the git-ignored traefik/traefik.selfsigned.yml and
// leave the tracked traefik/traefik.yml alone (zen upgrade's git checkout must
// stay clean); ACME modes keep writing traefik.yml.
func TestWriteTraefikConfigSelfSignedLeavesTrackedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "traefik"), 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := filepath.Join(dir, "traefik", "traefik.yml")
	const shipped = "# shipped\ncertificatesResolvers:\n  le: {}\n"
	if err := os.WriteFile(tracked, []byte(shipped), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Domain: "zenith.lan", TLSMode: TLSSelfSigned}
	if err := composeWriteTLS(cfg, localRunner{}, dir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(tracked); string(b) != shipped {
		t.Errorf("traefik.yml was rewritten:\n%s", b)
	}
	b, err := os.ReadFile(filepath.Join(dir, "traefik", "traefik.selfsigned.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != generateTraefikStaticConfig(TLSSelfSigned, acmeEmail(cfg)) {
		t.Errorf("traefik.selfsigned.yml content:\n%s", b)
	}

	http := &Config{Domain: "zenith.example.com", TLSMode: TLSHTTP01}
	dir2 := t.TempDir()
	if err := composeWriteTLS(http, localRunner{}, dir2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir2, "traefik", "traefik.yml")); err != nil {
		t.Errorf("http01: traefik.yml not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir2, "traefik", "traefik.selfsigned.yml")); err == nil {
		t.Error("http01 must not write traefik.selfsigned.yml")
	}
}
