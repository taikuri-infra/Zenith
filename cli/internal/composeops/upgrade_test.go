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
