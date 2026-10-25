package k8sops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// UpgradeOptions configures Upgrade.
type UpgradeOptions struct {
	Target     string // version without the leading v
	Force      bool   // allow a downgrade or re-install of the same version
	SkipBackup bool
	DryRun     bool
	BackupDir  string // default: ./zenith-backups
	Now        func() time.Time
}

// UpgradeResult reports what Upgrade did.
type UpgradeResult struct {
	From, To       string
	Backup         string
	RolledBack     bool
	RollbackFailed bool
}

type helmValues struct {
	Domain string `json:"domain"`
	Images struct {
		API        string `json:"api"`
		PullPolicy string `json:"pullPolicy"`
	} `json:"images"`
	TLS struct {
		IssuerName string `json:"issuerName"`
		Issuer     struct {
			Create *bool  `json:"create"`
			Email  string `json:"email"`
		} `json:"issuer"`
	} `json:"tls"`
}

func (i Install) releaseValues(ctx context.Context, r Runner) (*helmValues, error) {
	out, err := r.Run(ctx, "", nil, "helm", "get", "values", "zenith", "-n", i.Namespace, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("read the zenith Helm release: %w", err)
	}
	var v helmValues
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("helm values: %w", err)
	}
	if v.Domain == "" {
		return nil, fmt.Errorf("the zenith Helm release has no domain value")
	}
	return &v, nil
}

// scriptEnv is the installer environment that reproduces the current release at a new version.
func (v *helmValues) scriptEnv(version string) map[string]string {
	env := map[string]string{"ZENITH_DOMAIN": v.Domain, "ZENITH_VERSION": version}
	if v.Images.API != "" {
		repo := v.Images.API
		if idx := strings.LastIndex(repo, "/zenith-api"); idx >= 0 {
			env["ZENITH_IMAGE_REPO"] = repo[:idx]
		}
	}
	if v.Images.PullPolicy != "" {
		env["ZENITH_PULL_POLICY"] = v.Images.PullPolicy
	}
	if v.TLS.Issuer.Create != nil && !*v.TLS.Issuer.Create && v.TLS.IssuerName != "" {
		env["ZENITH_TLS_ISSUER"] = v.TLS.IssuerName
	}
	if v.TLS.Issuer.Email != "" {
		env["ZENITH_EMAIL"] = v.TLS.Issuer.Email
	}
	if version != "latest" {
		env["ZENITH_BRANCH"] = "v" + strings.TrimPrefix(version, "v")
	}
	return env
}

// runScript runs the embedded installer with env, streaming its output; tests replace it.
var runScript = func(ctx context.Context, env map[string]string, kubeconfig string, out io.Writer) error {
	dir, err := os.MkdirTemp("", "zenith-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "install-k3s.sh")
	if err := os.WriteFile(path, installScript, 0o700); err != nil {
		return err
	}
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command -- the script is embedded in this binary and written to a private temp dir; env values are flag-validated strings.
	cmd := exec.CommandContext(ctx, "bash", path)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if kubeconfig != "" {
		cmd.Env = append(cmd.Env, "KUBECONFIG="+kubeconfig)
	}
	cmd.Stdout, cmd.Stderr = out, out
	return cmd.Run()
}

// InstallPlatform runs the embedded k3s installer (k3s, operators, Zenith chart).
func InstallPlatform(ctx context.Context, env map[string]string, out io.Writer) error {
	return runScript(ctx, env, "", out)
}

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`)

// Upgrade moves the installation to opt.Target by re-running the installer
// against the existing release's settings, after a backup. A failed upgrade
// is rolled back with helm.
func Upgrade(ctx context.Context, r Runner, inst Install, opt UpgradeOptions, log io.Writer) (*UpgradeResult, error) {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if !versionPattern.MatchString(opt.Target) {
		return nil, fmt.Errorf("invalid version %q", opt.Target)
	}
	res := &UpgradeResult{From: inst.Version(ctx, r), To: opt.Target}
	if !opt.Force && res.From != "" && res.From != "latest" && CompareVersions(opt.Target, res.From) <= 0 {
		return res, fmt.Errorf("already on %s (target %s is not newer; use --force)", res.From, opt.Target)
	}
	vals, err := inst.releaseValues(ctx, r)
	if err != nil {
		return res, err
	}
	if opt.DryRun {
		fmt.Fprintf(log, "would upgrade %s -> %s (domain %s)\n", res.From, opt.Target, vals.Domain)
		return res, nil
	}
	if !opt.SkipBackup {
		dir := opt.BackupDir
		if dir == "" {
			dir = "zenith-backups"
		}
		fmt.Fprintln(log, "backing up before the upgrade")
		if res.Backup, err = Backup(ctx, r, inst, dir, opt.Now()); err != nil {
			return res, fmt.Errorf("pre-upgrade backup (use --skip-backup to skip): %w", err)
		}
	}
	fmt.Fprintf(log, "upgrading %s -> %s\n", res.From, opt.Target)
	if err := runScript(ctx, vals.scriptEnv(opt.Target), inst.Kubeconfig, log); err != nil {
		fmt.Fprintln(log, "upgrade failed; rolling back")
		if _, rbErr := r.Run(ctx, "", nil, "helm", "rollback", "zenith", "-n", inst.Namespace, "--wait", "--timeout", "10m"); rbErr != nil {
			res.RollbackFailed = true
			return res, fmt.Errorf("upgrade failed (%v) and rollback failed: %w", err, rbErr)
		}
		res.RolledBack = true
		return res, fmt.Errorf("upgrade failed and was rolled back: %w", err)
	}
	return res, nil
}

// githubAPI is the GitHub API base URL; tests point it at an httptest server.
var githubAPI = "https://api.github.com"

var platformTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// LatestRelease returns the newest non-draft platform release tag (vX.Y.Z, prereleases included),
// without the leading v. The repo also publishes zen-v*, zenith-api-v* and zenith-web-v* releases.
func LatestRelease(ctx context.Context, repo string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPI+"/repos/"+repo+"/releases?per_page=30", nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
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
