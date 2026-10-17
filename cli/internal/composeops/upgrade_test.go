package composeops

import (
	"bytes"
	"context"
	"errors"
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
		if !strings.HasPrefix(c, "git rev-parse") {
			t.Fatalf("dry run executed %q", c)
		}
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

// selfSignedAlpha1 makes inst look like a v0.1.0-alpha.1 self-signed install:
// ZENITH_TLS_MODE=selfsigned, no ZENITH_TRAEFIK_CONFIG, traefik.yml overwritten.
func selfSignedAlpha1(t *testing.T, inst Install) string {
	t.Helper()
	if err := inst.EnvSet("ZENITH_TLS_MODE", "selfsigned"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(inst.Dir, "traefik"), 0o755); err != nil {
		t.Fatal(err)
	}
	const cfg = "# Generated by install.sh (ZENITH_TLS_MODE=selfsigned).\nentryPoints: {}\n"
	if err := os.WriteFile(filepath.Join(inst.Dir, "traefik", "traefik.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestUpgrade_MigratesAlpha1SelfSignedTraefikConfig(t *testing.T) {
	inst := writeInstall(t)
	cfg := selfSignedAlpha1(t, inst)
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true} // traefik.yml modified
	var log bytes.Buffer
	if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, &log); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(inst.Dir, "traefik", "traefik.selfsigned.yml"))
	if err != nil || string(b) != cfg {
		t.Fatalf("traefik.selfsigned.yml = %q, %v; want the old traefik.yml", b, err)
	}
	if got := inst.EnvGet("ZENITH_TRAEFIK_CONFIG"); got != "/etc/traefik/traefik.selfsigned.yml" {
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
	if n := strings.Count(log.String(), "traefik.selfsigned.yml"); n != 1 {
		t.Fatalf("want one migration log line, log:\n%s", log.String())
	}
}

func TestUpgrade_SelfSignedMigrationNoopOnHTTP01(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	r.fail = map[string]bool{"git diff --quiet": true}
	if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(context.Context, string) error { return nil },
	}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, c := range r.cmds() {
		if strings.HasPrefix(c, "git diff") || c == "git checkout -- traefik/traefik.yml" {
			t.Fatalf("http01 install ran %q", c)
		}
	}
	if inst.EnvGet("ZENITH_TRAEFIK_CONFIG") != "" {
		t.Fatal("http01 install got ZENITH_TRAEFIK_CONFIG")
	}
	if _, err := os.Stat(filepath.Join(inst.Dir, "traefik", "traefik.selfsigned.yml")); err == nil {
		t.Fatal("http01 install got traefik.selfsigned.yml")
	}
}

func TestUpgrade_SelfSignedMigrationNoopWhenUnmodifiedOrAlreadyMigrated(t *testing.T) {
	for name, setup := range map[string]func(Install, *fakeRunner){
		"unmodified": func(Install, *fakeRunner) {},
		"already migrated": func(i Install, r *fakeRunner) {
			r.fail = map[string]bool{"git diff --quiet": true}
			_ = i.EnvSet("ZENITH_TRAEFIK_CONFIG", "/x.yml")
		},
	} {
		t.Run(name, func(t *testing.T) {
			inst := writeInstall(t)
			selfSignedAlpha1(t, inst)
			r := upgradeRunner()
			setup(inst, r)
			if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
				Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
				Health: func(context.Context, string) error { return nil },
			}, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(inst.Dir, "traefik", "traefik.selfsigned.yml")); err == nil {
				t.Fatal("traefik.selfsigned.yml written")
			}
		})
	}
}

// Rolling back to alpha.1 (whose compose ignores ZENITH_TRAEFIK_CONFIG) puts the
// self-signed config back in traefik.yml.
func TestUpgrade_RollbackRevertsSelfSignedMigration(t *testing.T) {
	inst := writeInstall(t)
	cfg := selfSignedAlpha1(t, inst)
	// the fake git checkout does not restore the file; mimic it
	if err := os.WriteFile(filepath.Join(inst.Dir, "traefik", "traefik.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
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
	if _, err := os.Stat(filepath.Join(inst.Dir, "traefik", "traefik.selfsigned.yml")); err == nil {
		t.Fatal("traefik.selfsigned.yml left behind after rollback")
	}
	if inst.EnvGet("ZENITH_TRAEFIK_CONFIG") != "" {
		t.Fatal("ZENITH_TRAEFIK_CONFIG left in .env after rollback")
	}
}

func TestUpgrade_CheckoutFailureRevertsSelfSignedMigration(t *testing.T) {
	inst := writeInstall(t)
	cfg := selfSignedAlpha1(t, inst)
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
