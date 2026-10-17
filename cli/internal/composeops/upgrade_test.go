package composeops

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func upgradeRunner() *fakeRunner {
	return &fakeRunner{answers: map[string]string{
		"git rev-parse HEAD":                   "oldsha\n",
		"docker compose exec -T postgres psql": "",
		"docker compose ps -q storage":         "cid\n",
		"docker inspect":                       "vol\n",
	}}
}

func TestUpgrade_HappyPath(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", Health: func(context.Context, string) error { return nil },
		Timeout: time.Second, Now: time.Now,
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.From != "0.1.0-alpha.2" || res.To != "0.1.0-alpha.3" || res.Backup == "" || res.RolledBack {
		t.Fatalf("res = %+v", res)
	}
	joined := strings.Join(r.cmds(), "\n")
	order := []string{
		"git fetch --depth 1 origin tag v0.1.0-alpha.3",
		"git checkout v0.1.0-alpha.3",
		"env ZENITH_DRY_RUN=1 bash infra/scripts/install.sh",
		"docker compose pull",
		"docker compose up -d --remove-orphans",
	}
	last := -1
	for _, step := range order {
		i := strings.Index(joined, step)
		if i < 0 || i < last {
			t.Fatalf("step %q missing or out of order:\n%s", step, joined)
		}
		last = i
	}
}

func TestUpgrade_RollsBackOnHealthFailure(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	calls := 0
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(_ context.Context, want string) error {
			calls++
			if want == "0.1.0-alpha.3" {
				return errors.New("never healthy")
			}
			return nil
		},
	}, &bytes.Buffer{})
	if err == nil || res == nil || !res.RolledBack {
		t.Fatalf("want rollback, got res=%+v err=%v", res, err)
	}
	joined := strings.Join(r.cmds(), "\n")
	if !strings.Contains(joined, "git checkout oldsha") {
		t.Fatalf("old ref not restored:\n%s", joined)
	}
	if inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" {
		t.Fatalf("ZENITH_VERSION = %q after rollback", inst.EnvGet("ZENITH_VERSION"))
	}
	if calls != 2 {
		t.Fatalf("health checks = %d, want new then old", calls)
	}
}

func TestUpgrade_RefusesDowngradeUnlessForced(t *testing.T) {
	inst := writeInstall(t)
	_, err := Upgrade(context.Background(), upgradeRunner(), inst, UpgradeOptions{Target: "0.1.0-alpha.1", SkipBackup: true, Now: time.Now, Health: func(context.Context, string) error { return nil }}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpgrade_DryRunChangesNothing(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	var log bytes.Buffer
	if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{Target: "0.1.0-alpha.3", DryRun: true, Now: time.Now}, &log); err != nil {
		t.Fatal(err)
	}
	for _, c := range r.cmds() {
		if !strings.HasPrefix(c, "git rev-parse") && !strings.HasPrefix(c, "git diff --quiet") {
			t.Fatalf("dry run executed %q", c)
		}
	}
	if strings.Contains(log.String(), "traefik.local.yml") {
		t.Fatalf("plan names the traefik migration for an unmodified traefik.yml:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "0.1.0-alpha.3") {
		t.Fatalf("plan not printed: %s", log.String())
	}
}

func bothUnhealthy(context.Context, string) error { return errors.New("down") }

func TestUpgrade_FailedRollbackPointsAtBackup(t *testing.T) {
	inst := writeInstall(t)
	res, err := Upgrade(context.Background(), upgradeRunner(), inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", Timeout: time.Second, Now: time.Now, BackupDir: t.TempDir(), Health: bothUnhealthy,
	}, &bytes.Buffer{})
	if err == nil || res == nil || res.RolledBack || !res.RollbackFailed || res.Backup == "" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !strings.Contains(err.Error(), "zen backup restore "+res.Backup) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpgrade_FailedRollbackWithoutBackup(t *testing.T) {
	inst := writeInstall(t)
	res, err := Upgrade(context.Background(), upgradeRunner(), inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now, Health: bothUnhealthy,
	}, &bytes.Buffer{})
	if err == nil || res.RolledBack || !strings.Contains(err.Error(), "take a backup next time") {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestUpgrade_RollbackUpFailure(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	r.fail = map[string]bool{"docker compose up": true}
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, &bytes.Buffer{})
	if err == nil || res.RolledBack || !res.RollbackFailed || !strings.Contains(err.Error(), "docker compose up") {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestUpgrade_FetchFailureDoesNotRollBack(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	r.fail = map[string]bool{"git fetch": true}
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, &bytes.Buffer{})
	if err == nil || res.RolledBack || res.RollbackFailed || !strings.Contains(err.Error(), "nothing changed") {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	for _, c := range r.cmds() {
		if strings.HasPrefix(c, "git checkout") || strings.HasPrefix(c, "docker compose") {
			t.Fatalf("ran %q after fetch failure", c)
		}
	}
}

func TestUpgrade_RejectsInvalidTarget(t *testing.T) {
	r := upgradeRunner()
	_, err := Upgrade(context.Background(), r, writeInstall(t), UpgradeOptions{Target: "1.0.0; rm -rf /", Now: time.Now}, &bytes.Buffer{})
	if err == nil || len(r.cmds()) != 0 {
		t.Fatalf("err=%v cmds=%v", err, r.cmds())
	}
}

func TestUpgrade_CheckoutFailureDoesNotRollBack(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	r.fail = map[string]bool{"git checkout": true}
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, &bytes.Buffer{})
	if err == nil || res.RolledBack || res.RollbackFailed || !strings.Contains(err.Error(), "nothing changed") {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	for _, c := range r.cmds() {
		if strings.HasPrefix(c, "docker compose") {
			t.Fatalf("ran %q after checkout failure", c)
		}
	}
	if n := strings.Count(strings.Join(r.cmds(), "\n"), "git checkout"); n != 1 {
		t.Fatalf("git checkout ran %d times, want 1 (no rollback)", n)
	}
}

func TestUpgrade_RollbackWithoutKnownVersionUnsetsEnvAndChecksHealthOnly(t *testing.T) {
	inst := writeInstall(t)
	if err := inst.EnvUnset("ZENITH_VERSION"); err != nil {
		t.Fatal(err)
	}
	r := upgradeRunner()
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(_ context.Context, want string) error {
			if want != "" {
				return errors.New("new version unhealthy")
			}
			return nil
		},
	}, &bytes.Buffer{})
	if err == nil || res == nil || !res.RolledBack || res.RollbackFailed {
		t.Fatalf("want successful rollback, got res=%+v err=%v", res, err)
	}
	b, _ := os.ReadFile(filepath.Join(inst.Dir, ".env"))
	if strings.Contains(string(b), "ZENITH_VERSION") {
		t.Fatalf(".env still has ZENITH_VERSION:\n%s", b)
	}
	if !strings.Contains(string(b), "DB_PASSWORD=old") {
		t.Fatalf("other keys lost:\n%s", b)
	}
}

func TestTargetReAcceptsDashInPrerelease(t *testing.T) {
	for _, ok := range []string{"1.5.0", "0.1.0-alpha.3", "0.1.0-alpha.3-test", "0.1.0-rc-1.2"} {
		if !targetRe.MatchString(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"v1.5.0", "1.5", "1.5.0-", "1.5.0 ", "latest"} {
		if targetRe.MatchString(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}

func latestFrom(t *testing.T, status int, body string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/releases" || r.URL.Query().Get("per_page") != "30" {
			t.Errorf("unexpected request %s", r.URL.String())
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	old := githubAPI
	githubAPI = srv.URL
	defer func() { githubAPI = old }()
	return LatestRelease(context.Background(), "o/r")
}

func TestLatestReleaseSkipsNonPlatformTags(t *testing.T) {
	got, err := latestFrom(t, 200, `[{"tag_name":"zen-v0.1.2"},{"tag_name":"v0.1.0"},{"tag_name":"v0.1.0-alpha.4"}]`)
	if err != nil || got != "0.1.0" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestLatestReleaseKeepsPrereleases(t *testing.T) {
	got, err := latestFrom(t, 200, `[{"tag_name":"zenith-api-v0.2.0"},{"tag_name":"v0.1.0-alpha.4"}]`)
	if err != nil || got != "0.1.0-alpha.4" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestLatestReleaseSkipsDrafts(t *testing.T) {
	got, err := latestFrom(t, 200, `[{"tag_name":"v0.2.0","draft":true},{"tag_name":"v0.1.0"}]`)
	if err != nil || got != "0.1.0" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestLatestReleaseNoPlatformRelease(t *testing.T) {
	_, err := latestFrom(t, 200, `[{"tag_name":"zen-v0.1.2"},{"tag_name":"zenith-web-v0.1.0"}]`)
	if err == nil || !strings.Contains(err.Error(), "no platform release found for o/r") {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestReleaseHTTPError(t *testing.T) {
	if _, err := latestFrom(t, 500, `oops`); err == nil {
		t.Fatal("want error")
	}
}

// modifiedTraefikYML makes inst look like an install whose installer overwrote
// the tracked traefik.yml (alpha.1 install.sh self-signed, or any older zen
// install): mode in .env (unless empty), no ZENITH_TRAEFIK_CONFIG.
func modifiedTraefikYML(t *testing.T, inst Install, mode string) string {
	t.Helper()
	if mode != "" {
		if err := inst.EnvSet("ZENITH_TLS_MODE", mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(inst.Dir, "traefik"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "# Generated (" + mode + ").\nentryPoints: {}\n"
	if err := os.WriteFile(filepath.Join(inst.Dir, "traefik", "traefik.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func upgradeOK(t *testing.T, r *fakeRunner, inst Install, log *bytes.Buffer) {
	t.Helper()
	if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, log); err != nil {
		t.Fatal(err)
	}
}

// selfsigned: alpha.1 install.sh; "" (http01): an older `zen install`, which
// wrote traefik.yml in every mode.
func TestUpgrade_MigratesModifiedTraefikYML(t *testing.T) {
	for _, mode := range []string{"selfsigned", ""} {
		t.Run("mode="+mode, func(t *testing.T) {
			inst := writeInstall(t)
			cfg := modifiedTraefikYML(t, inst, mode)
			r := upgradeRunner()
			r.fail = map[string]bool{"git diff --quiet": true} // traefik.yml modified
			var log bytes.Buffer
			upgradeOK(t, r, inst, &log)
			b, err := os.ReadFile(filepath.Join(inst.Dir, "traefik", "traefik.local.yml"))
			if err != nil || string(b) != cfg {
				t.Fatalf("traefik.local.yml = %q, %v; want the old traefik.yml", b, err)
			}
			if got := inst.EnvGet("ZENITH_TRAEFIK_CONFIG"); got != "/etc/traefik/traefik.local.yml" {
				t.Fatalf("ZENITH_TRAEFIK_CONFIG = %q", got)
			}
			joined := strings.Join(r.cmds(), "\n")
			last := -1
			for _, step := range []string{"git diff --quiet -- traefik/traefik.yml", "git checkout -- traefik/traefik.yml", "git checkout v0.1.0-alpha.3"} {
				i := strings.Index(joined, step)
				if i < 0 || i < last {
					t.Fatalf("step %q missing or out of order:\n%s", step, joined)
				}
				last = i
			}
			if n := strings.Count(log.String(), "edit traefik/traefik.local.yml from now on\n"); n != 1 {
				t.Fatalf("want one migration log line ending in the edit hint, log:\n%s", log.String())
			}
		})
	}
}

func TestUpgrade_TraefikMigrationNoopWhenUnmodifiedOrAlreadyMigrated(t *testing.T) {
	for name, setup := range map[string]func(Install, *fakeRunner){
		"unmodified http01":     func(Install, *fakeRunner) {},
		"unmodified selfsigned": func(i Install, _ *fakeRunner) { _ = i.EnvSet("ZENITH_TLS_MODE", "selfsigned") },
		"already migrated": func(i Install, r *fakeRunner) {
			r.fail = map[string]bool{"git diff --quiet": true}
			_ = i.EnvSet("ZENITH_TRAEFIK_CONFIG", "/x.yml")
		},
	} {
		t.Run(name, func(t *testing.T) {
			inst := writeInstall(t)
			r := upgradeRunner()
			setup(inst, r)
			before := inst.EnvGet("ZENITH_TRAEFIK_CONFIG")
			upgradeOK(t, r, inst, &bytes.Buffer{})
			for _, c := range r.cmds() {
				if c == "git checkout -- traefik/traefik.yml" {
					t.Fatalf("ran %q", c)
				}
			}
			if _, err := os.Stat(filepath.Join(inst.Dir, "traefik", "traefik.local.yml")); err == nil {
				t.Fatal("traefik.local.yml written")
			}
			if inst.EnvGet("ZENITH_TRAEFIK_CONFIG") != before {
				t.Fatal("ZENITH_TRAEFIK_CONFIG changed")
			}
		})
	}
}

// Rolling back to a release whose compose ignores ZENITH_TRAEFIK_CONFIG puts
// the old config back in traefik.yml.
func TestUpgrade_RollbackRevertsTraefikMigration(t *testing.T) {
	inst := writeInstall(t)
	cfg := modifiedTraefikYML(t, inst, "selfsigned")
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true}
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(_ context.Context, want string) error {
			if want == "0.1.0-alpha.3" {
				return errors.New("never healthy")
			}
			return nil
		},
	}, &bytes.Buffer{})
	if err == nil || res == nil || !res.RolledBack {
		t.Fatalf("want rollback, res=%+v err=%v", res, err)
	}
	if b, _ := os.ReadFile(filepath.Join(inst.Dir, "traefik", "traefik.yml")); string(b) != cfg {
		t.Fatalf("traefik.yml after rollback = %q", b)
	}
	if _, err := os.Stat(filepath.Join(inst.Dir, "traefik", "traefik.local.yml")); err == nil {
		t.Fatal("traefik.local.yml left behind after rollback")
	}
	if inst.EnvGet("ZENITH_TRAEFIK_CONFIG") != "" {
		t.Fatal("ZENITH_TRAEFIK_CONFIG left in .env after rollback")
	}
}

func TestUpgrade_CheckoutFailureRevertsTraefikMigration(t *testing.T) {
	inst := writeInstall(t)
	cfg := modifiedTraefikYML(t, inst, "")
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true, "git checkout v": true}
	_, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "nothing changed") {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(inst.Dir, "traefik", "traefik.yml")); string(b) != cfg {
		t.Fatalf("traefik.yml = %q", b)
	}
	if inst.EnvGet("ZENITH_TRAEFIK_CONFIG") != "" {
		t.Fatal("ZENITH_TRAEFIK_CONFIG left in .env")
	}
}

func TestUpgrade_DryRunPlansTraefikMigrationWhenModified(t *testing.T) {
	inst := writeInstall(t)
	modifiedTraefikYML(t, inst, "")
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true}
	var log bytes.Buffer
	if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{Target: "0.1.0-alpha.3", DryRun: true, Now: time.Now}, &log); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log.String(), "traefik.local.yml") {
		t.Fatalf("plan lacks the traefik migration:\n%s", log.String())
	}
	if _, err := os.Stat(filepath.Join(inst.Dir, "traefik", "traefik.local.yml")); err == nil {
		t.Fatal("dry run wrote traefik.local.yml")
	}
}

// Key already set and traefik.local.yml is the live config, but the tracked file
// is still modified (an older zen install re-run): only restore the tracked file.
func TestUpgrade_RestoresModifiedTraefikYMLWhenLocalIsLive(t *testing.T) {
	inst := writeInstall(t)
	modifiedTraefikYML(t, inst, "")
	local := filepath.Join(inst.Dir, "traefik", "traefik.local.yml")
	if err := os.WriteFile(local, []byte("live\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := inst.EnvSet("ZENITH_TRAEFIK_CONFIG", "/etc/traefik/traefik.local.yml"); err != nil {
		t.Fatal(err)
	}
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true}
	var log bytes.Buffer
	upgradeOK(t, r, inst, &log)
	joined := strings.Join(r.cmds(), "\n")
	i, j := strings.Index(joined, "git checkout -- traefik/traefik.yml"), strings.Index(joined, "git checkout v0.1.0-alpha.3")
	if i < 0 || j < 0 || i > j {
		t.Fatalf("tracked file not restored before the release checkout:\n%s", joined)
	}
	if b, _ := os.ReadFile(local); string(b) != "live\n" {
		t.Fatalf("traefik.local.yml changed: %q", b)
	}
	if inst.EnvGet("ZENITH_TRAEFIK_CONFIG") != "/etc/traefik/traefik.local.yml" {
		t.Fatal("key changed")
	}
	if !strings.Contains(log.String(), "edit traefik/traefik.local.yml from now on\n") {
		t.Fatalf("log:\n%s", log.String())
	}
}

func TestUpgrade_KeySetButLocalMissingIsNoop(t *testing.T) {
	inst := writeInstall(t)
	modifiedTraefikYML(t, inst, "")
	if err := inst.EnvSet("ZENITH_TRAEFIK_CONFIG", "/etc/traefik/traefik.local.yml"); err != nil {
		t.Fatal(err)
	}
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true}
	upgradeOK(t, r, inst, &bytes.Buffer{})
	for _, c := range r.cmds() {
		if c == "git checkout -- traefik/traefik.yml" {
			t.Fatalf("ran %q without a live traefik.local.yml", c)
		}
	}
}

// If .env cannot be written, the tracked traefik.yml (the live config of the
// old release) must be left exactly as it was.
func TestMigrateTraefikConfig_EnvSetFailureLeavesTraefikYML(t *testing.T) {
	dir := t.TempDir()
	inst := Install{Dir: dir}
	cfg := modifiedTraefikYML(t, inst, "")
	if err := os.Mkdir(filepath.Join(dir, ".env"), 0o755); err != nil { // EnvSet fails
		t.Fatal(err)
	}
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true}
	if _, err := migrateTraefikConfig(context.Background(), r, inst, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "traefik", "traefik.yml")); string(b) != cfg {
		t.Fatalf("traefik.yml = %q", b)
	}
	for _, c := range r.cmds() {
		if c == "git checkout -- traefik/traefik.yml" {
			t.Fatalf("ran %q although .env could not be updated", c)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "traefik", "traefik.local.yml")); err == nil {
		t.Fatal("traefik.local.yml left behind")
	}
}
