package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A re-run on an install whose traefik.yml an older `zen install` overwrote
// restores the tracked file before `git pull --ff-only`, or the pull is refused.
func TestFetchStackCommandRestoresTraefikYMLBeforePull(t *testing.T) {
	cmd := fetchStackCommand("zenith")
	restore := strings.Index(cmd, "checkout -- traefik/traefik.yml")
	pull := strings.Index(cmd, "pull --ff-only")
	if restore < 0 || pull < 0 || restore > pull {
		t.Fatalf("restore must come before the pull:\n%s", cmd)
	}
	if !strings.Contains(cmd, "git clone --depth 1 --branch "+composeBranch) {
		t.Fatalf("clone branch missing:\n%s", cmd)
	}
}

func gitRepoWithTraefik(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "traefik"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "traefik", "traefik.yml"), []byte("shipped\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "traefik/traefik.yml"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "init"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestRestoreTrackedTraefikCommand(t *testing.T) {
	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }
	t.Run("modified, no local file: copied then restored", func(t *testing.T) {
		dir := gitRepoWithTraefik(t)
		yml, local := filepath.Join(dir, "traefik", "traefik.yml"), filepath.Join(dir, "traefik", "traefik.local.yml")
		os.WriteFile(yml, []byte("generated\n"), 0o600)
		if out, err := (localRunner{}).Run(restoreTrackedTraefikCommand(dir)); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		if read(yml) != "shipped\n" || read(local) != "generated\n" {
			t.Fatalf("traefik.yml=%q traefik.local.yml=%q", read(yml), read(local))
		}
	})
	t.Run("modified, local file exists: local kept, tracked restored", func(t *testing.T) {
		dir := gitRepoWithTraefik(t)
		yml, local := filepath.Join(dir, "traefik", "traefik.yml"), filepath.Join(dir, "traefik", "traefik.local.yml")
		os.WriteFile(yml, []byte("generated\n"), 0o600)
		os.WriteFile(local, []byte("live\n"), 0o600)
		if out, err := (localRunner{}).Run(restoreTrackedTraefikCommand(dir)); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		if read(yml) != "shipped\n" || read(local) != "live\n" {
			t.Fatalf("traefik.yml=%q traefik.local.yml=%q", read(yml), read(local))
		}
	})
	t.Run("unmodified: no-op", func(t *testing.T) {
		dir := gitRepoWithTraefik(t)
		if out, err := (localRunner{}).Run(restoreTrackedTraefikCommand(dir)); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "traefik", "traefik.local.yml")); err == nil {
			t.Fatal("traefik.local.yml written for a clean checkout")
		}
	})
}
