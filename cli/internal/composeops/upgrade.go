package composeops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// UpgradeOptions configures Upgrade.
type UpgradeOptions struct {
	Target     string // required, no leading v
	Force      bool   // allow same/older target
	SkipBackup bool
	DryRun     bool
	BackupDir  string // default <dir>/backups
	// Health checks the API; want is the expected version, or "" meaning
	// "only require /health to return 200" (rolling back a pre-2b install
	// whose version is unknown). nil = HTTP check on 127.0.0.1:8080.
	Health  func(ctx context.Context, want string) error
	Timeout time.Duration // default 180s
	Now     func() time.Time
}

// UpgradeResult reports what Upgrade did.
type UpgradeResult struct {
	From, To, Backup string
	RolledBack       bool // rollback ran and succeeded
	RollbackFailed   bool // rollback ran and failed
}

var targetRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]*[0-9A-Za-z])?$`)

// Upgrade moves a compose install to opt.Target, rolling back on failure.
func Upgrade(ctx context.Context, r Runner, inst Install, opt UpgradeOptions, log io.Writer) (*UpgradeResult, error) {
	target := strings.TrimPrefix(opt.Target, "v")
	if !targetRe.MatchString(target) {
		return nil, fmt.Errorf("invalid upgrade target %q: want X.Y.Z or X.Y.Z-prerelease", opt.Target)
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 180 * time.Second
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	health := opt.Health
	if health == nil {
		health = func(ctx context.Context, want string) error { return httpHealth(ctx, want, opt.Timeout) }
	}
	from := inst.EnvGet("ZENITH_VERSION")
	if !opt.Force && CompareVersions(target, from) <= 0 {
		return nil, fmt.Errorf("target %s is not newer than %s (use --force)", target, from)
	}
	refOut, err := r.Run(ctx, inst.Dir, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("read current git ref: %w", err)
	}
	oldRef := strings.TrimSpace(string(refOut))
	if oldRef == "" {
		return nil, fmt.Errorf("could not determine current git ref")
	}
	backupDir := opt.BackupDir
	if backupDir == "" {
		backupDir = filepath.Join(inst.Dir, "backups")
	}
	tag := "v" + target
	res := &UpgradeResult{From: from, To: target}

	if opt.DryRun {
		fmt.Fprintf(log, "Upgrade plan %s -> %s (dry run, nothing changed):\n", from, target)
		n := 1
		if opt.SkipBackup {
			fmt.Fprintf(log, "  (backup skipped)\n")
		} else {
			fmt.Fprintf(log, "  %d. back up to %s\n", n, backupDir)
			n++
		}
		switch traefikMigrationNeeded(ctx, r, inst) {
		case traefikMove:
			fmt.Fprintf(log, "  %d. move the modified %s to %s and set %s\n", n, trackedTraefikConfig, localTraefikConfig, traefikConfigKey)
			n++
		case traefikRestore:
			fmt.Fprintf(log, "  %d. restore the shipped %s (the live config is %s)\n", n, trackedTraefikConfig, localTraefikConfig)
			n++
		}
		for _, s := range []string{
			"git fetch --depth 1 origin tag " + tag + " and check out " + tag,
			"refresh .env (installer env-only mode)",
			"docker compose pull",
			"docker compose up -d --remove-orphans",
			"wait for health and version " + target,
		} {
			fmt.Fprintf(log, "  %d. %s\n", n, s)
			n++
		}
		return res, nil
	}

	if !opt.SkipBackup {
		fmt.Fprintf(log, "Backing up to %s\n", backupDir)
		p, err := Backup(ctx, r, inst, backupDir, opt.Now())
		if err != nil {
			return nil, fmt.Errorf("pre-upgrade backup failed (nothing changed): %w", err)
		}
		res.Backup = p
		fmt.Fprintf(log, "Backup written: %s\n", p)
	}

	var undoTraefik func() error // set when the traefik migration changed files
	steps := []struct {
		name string
		fn   func() error
	}{
		{"fetch " + tag, func() error {
			_, err := r.Run(ctx, inst.Dir, nil, "git", "fetch", "--depth", "1", "origin", "tag", tag)
			return err
		}},
		{"checkout " + tag, func() error {
			var err error
			if undoTraefik, err = migrateTraefikConfig(ctx, r, inst, log); err != nil {
				return err
			}
			if _, err = r.Run(ctx, inst.Dir, nil, "git", "checkout", tag); err != nil && undoTraefik != nil {
				if rerr := undoTraefik(); rerr != nil {
					return fmt.Errorf("%w (and restoring %s failed: %v)", err, trackedTraefikConfig, rerr)
				}
			}
			return err
		}},
		{"refresh .env", func() error {
			if _, err := r.Run(ctx, inst.Dir, nil, "env", "ZENITH_DRY_RUN=1", "bash", "infra/scripts/install.sh"); err != nil {
				return err
			}
			if inst.EnvGet("ZENITH_VERSION") != target {
				return inst.EnvSet("ZENITH_VERSION", target)
			}
			return nil
		}},
		{"docker compose pull", func() error {
			_, err := r.Run(ctx, inst.Dir, nil, "docker", "compose", "pull")
			return err
		}},
		{"docker compose up", func() error {
			_, err := r.Run(ctx, inst.Dir, nil, "docker", "compose", "up", "-d", "--remove-orphans")
			return err
		}},
		{"health check", func() error { return health(ctx, target) }},
	}
	for i, s := range steps {
		fmt.Fprintf(log, "[%d/%d] %s\n", i+1, len(steps), s.name)
		if err := s.fn(); err != nil {
			if i <= 1 { // fetch or checkout: nothing has changed yet
				return res, fmt.Errorf("%s failed: %v — nothing changed", s.name, err)
			}
			fmt.Fprintf(log, "step %q failed: %v; rolling back to %s\n", s.name, err, from)
			rbErr := rollback(context.WithoutCancel(ctx), r, inst, oldRef, from, undoTraefik, health)
			msg := fmt.Sprintf("upgrade failed at %q: %v", s.name, err)
			res.RolledBack = rbErr == nil
			res.RollbackFailed = rbErr != nil
			if rbErr != nil {
				msg += fmt.Sprintf("; rollback to %s also failed: %v; the database may already be migrated", from, rbErr)
				if res.Backup != "" {
					msg += " — restore with: zen backup restore " + res.Backup
				} else {
					msg += " — take a backup next time"
				}
			} else {
				msg += fmt.Sprintf("; rolled back to %s", from)
			}
			return res, fmt.Errorf("%s", msg)
		}
	}
	fmt.Fprintf(log, "Upgraded %s -> %s\n", from, target)
	return res, nil
}

func rollback(ctx context.Context, r Runner, inst Install, oldRef, from string, undoTraefik func() error, health func(context.Context, string) error) error {
	if _, err := r.Run(ctx, inst.Dir, nil, "git", "checkout", oldRef); err != nil {
		return fmt.Errorf("git checkout %s: %w", oldRef, err)
	}
	if undoTraefik != nil {
		if err := undoTraefik(); err != nil {
			return err
		}
	}
	if from == "" {
		if err := inst.EnvUnset("ZENITH_VERSION"); err != nil {
			return err
		}
	} else if err := inst.EnvSet("ZENITH_VERSION", from); err != nil {
		return err
	}
	if _, err := r.Run(ctx, inst.Dir, nil, "docker", "compose", "up", "-d", "--remove-orphans"); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	if err := health(ctx, from); err != nil {
		return fmt.Errorf("old version unhealthy: %w", err)
	}
	return nil
}

const (
	trackedTraefikConfig = "traefik/traefik.yml"
	localTraefikConfig   = "traefik/traefik.local.yml"
	traefikConfigKey     = "ZENITH_TRAEFIK_CONFIG"
)

type traefikAction int

const (
	traefikNone    traefikAction = iota
	traefikMove                  // no key yet: move the modified traefik.yml to traefik.local.yml
	traefikRestore               // traefik.local.yml is already live: just restore traefik.yml
)

// traefikMigrationNeeded reports what migrateTraefikConfig would do. Older
// installers overwrote the tracked traefik/traefik.yml (install.sh self-signed
// in v0.1.0-alpha.1, `zen install` in every TLS mode), which makes the release
// checkout refuse once upstream changes the file. Read-only: runs `git diff`.
func traefikMigrationNeeded(ctx context.Context, r Runner, inst Install) traefikAction {
	act := traefikNone
	switch key := inst.EnvGet(traefikConfigKey); {
	case key == "":
		act = traefikMove
	case key == "/etc/"+localTraefikConfig:
		if _, err := os.Stat(filepath.Join(inst.Dir, localTraefikConfig)); err == nil {
			act = traefikRestore
		}
	}
	if act == traefikNone {
		return act
	}
	if _, err := os.Stat(filepath.Join(inst.Dir, trackedTraefikConfig)); err != nil {
		return traefikNone
	}
	if _, err := r.Run(ctx, inst.Dir, nil, "git", "diff", "--quiet", "--", trackedTraefikConfig); err == nil {
		return traefikNone // unmodified
	}
	return act
}

// migrateTraefikConfig restores a modified tracked traefik/traefik.yml so the
// release checkout is not refused. Without ZENITH_TRAEFIK_CONFIG it first
// keeps the content as traefik/traefik.local.yml and points the key at it;
// with traefik.local.yml already live it only restores the tracked file. It
// returns an undo func when it changed anything (nil otherwise); on error,
// nothing is changed.
func migrateTraefikConfig(ctx context.Context, r Runner, inst Install, log io.Writer) (func() error, error) {
	act := traefikMigrationNeeded(ctx, r, inst)
	if act == traefikNone {
		return nil, nil
	}
	src := filepath.Join(inst.Dir, trackedTraefikConfig)
	fi, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	checkout := func() error {
		if _, err := r.Run(ctx, inst.Dir, nil, "git", "checkout", "--", trackedTraefikConfig); err != nil {
			return fmt.Errorf("restore %s: %w", trackedTraefikConfig, err)
		}
		return nil
	}
	const hint = "edit " + localTraefikConfig + " from now on"
	if act == traefikRestore {
		if err := checkout(); err != nil {
			return nil, err
		}
		fmt.Fprintf(log, "Restored the shipped %s (the live config is %s); %s\n", trackedTraefikConfig, localTraefikConfig, hint)
		return func() error { return os.WriteFile(src, b, fi.Mode().Perm()) }, nil
	}
	dst := filepath.Join(inst.Dir, localTraefikConfig)
	if err := os.WriteFile(dst, b, fi.Mode().Perm()); err != nil {
		return nil, err
	}
	// Key before checkout: if .env cannot be written, traefik.yml is untouched.
	if err := inst.EnvSet(traefikConfigKey, "/etc/"+localTraefikConfig); err != nil {
		os.Remove(dst)
		return nil, err
	}
	if err := checkout(); err != nil {
		os.Remove(dst)
		_ = inst.EnvUnset(traefikConfigKey)
		return nil, err
	}
	fmt.Fprintf(log, "Moved the modified %s to %s (%s set in .env); %s\n", trackedTraefikConfig, localTraefikConfig, traefikConfigKey, hint)
	return func() error { return revertTraefikConfig(inst) }, nil
}

// revertTraefikConfig undoes migrateTraefikConfig for the old release,
// whose compose file reads traefik.yml only.
func revertTraefikConfig(inst Install) error {
	src := filepath.Join(inst.Dir, localTraefikConfig)
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(inst.Dir, trackedTraefikConfig), b, 0o644); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return err
	}
	return inst.EnvUnset(traefikConfigKey)
}

func httpHealth(ctx context.Context, want string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var last error
	for {
		last = checkOnce(ctx, want)
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("not healthy within %s: %w", timeout, last)
		case <-time.After(3 * time.Second):
		}
	}
}

func get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return (&http.Client{Timeout: 10 * time.Second}).Do(req)
}

func checkOnce(ctx context.Context, want string) error {
	resp, err := get(ctx, "http://127.0.0.1:8080/health")
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/health returned %d", resp.StatusCode)
	}
	if want == "" {
		return nil
	}
	resp, err = get(ctx, "http://127.0.0.1:8080/api/v1/version")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/api/v1/version returned %d", resp.StatusCode)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return err
	}
	if want != "" && strings.TrimPrefix(v.Version, "v") != want {
		return fmt.Errorf("api reports version %q, want %q", v.Version, want)
	}
	return nil
}

// githubAPI is the GitHub API base URL; tests point it at an httptest server.
var githubAPI = "https://api.github.com"

var platformTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// LatestRelease returns the newest non-draft platform release tag (vX.Y.Z, prereleases included),
// without the leading v. The repo also publishes zen-v*, zenith-api-v* and zenith-web-v* releases.
func LatestRelease(ctx context.Context, repo string) (string, error) {
	resp, err := get(ctx, githubAPI+"/repos/"+repo+"/releases?per_page=30")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}
	var rels []struct {
		Tag   string `json:"tag_name"`
		Draft bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return "", err
	}
	for _, rel := range rels {
		if !rel.Draft && platformTag.MatchString(rel.Tag) {
			return strings.TrimPrefix(rel.Tag, "v"), nil
		}
	}
	return "", fmt.Errorf("no platform release found for %s", repo)
}
