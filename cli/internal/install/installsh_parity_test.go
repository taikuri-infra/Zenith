package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stripLeadingComments drops the leading `#` comment lines (each installer names
// itself there) so the remaining config can be compared byte for byte.
func stripLeadingComments(s string) string {
	for strings.HasPrefix(s, "#") {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			return ""
		}
		s = s[i+1:]
	}
	return s
}

// install.sh (ZENITH_TLS_MODE=selfsigned) and `zen install --tls-mode selfsigned`
// must hand Traefik the same configuration. The Go generators are the reference.
func TestInstallShSelfSignedTraefikConfigMatchesGo(t *testing.T) {
	for _, bin := range []string{"bash", "openssl"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not available: %v", bin, err)
		}
	}
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	repo := filepath.Join(filepath.Dir(self), "..", "..", "..")
	script := filepath.Join(repo, "infra", "scripts", "install.sh")

	// Same dry-run setup as infra/scripts/install_test.sh's self-signed case.
	dir := t.TempDir()
	for _, f := range []string{".env.example", "VERSION"} {
		b, err := os.ReadFile(filepath.Join(repo, f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"ZENITH_DRY_RUN=1",
		"ZENITH_PUBLIC_IP=203.0.113.7",
		"ZENITH_TLS_MODE=selfsigned",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install.sh dry run: %v\n%s", err, out)
	}

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got, want := read("traefik/dynamic/certs.yml"), generateTraefikCertsConfig(); got != want {
		t.Errorf("certs.yml differs from the Go installer's\n--- install.sh\n%s\n--- zen install\n%s", got, want)
	}
	got := stripLeadingComments(read("traefik/traefik.yml"))
	want := stripLeadingComments(generateTraefikStaticConfig(TLSSelfSigned, ""))
	if got != want {
		t.Errorf("traefik.yml differs from the Go installer's (leading comments stripped)\n--- install.sh\n%s\n--- zen install\n%s", got, want)
	}
}
