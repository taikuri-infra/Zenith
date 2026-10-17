package composeops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	steps := []struct {
		name string
		fn   func() error
	}{
		{"fetch " + tag, func() error {
			_, err := r.Run(ctx, inst.Dir, nil, "git", "fetch", "--depth", "1", "origin", "tag", tag)
			return err
		}},
		{"checkout " + tag, func() error {
			_, err := r.Run(ctx, inst.Dir, nil, "git", "checkout", tag)
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
			rbErr := rollback(context.WithoutCancel(ctx), r, inst, oldRef, from, health)
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

func rollback(ctx context.Context, r Runner, inst Install, oldRef, from string, health func(context.Context, string) error) error {
	if _, err := r.Run(ctx, inst.Dir, nil, "git", "checkout", oldRef); err != nil {
		return fmt.Errorf("git checkout %s: %w", oldRef, err)
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
