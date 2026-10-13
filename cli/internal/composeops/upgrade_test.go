package composeops

import (
	"bytes"
	"context"
	"errors"
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
		"bash infra/scripts/install.sh",
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
