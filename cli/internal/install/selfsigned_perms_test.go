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
