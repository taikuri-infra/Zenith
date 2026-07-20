package install

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dotechhq/zenith/cli/internal/installstate"
)

// generateSecret returns an alphanumeric secret. Unlike GeneratePassword (which
// includes !@#$%), this is safe to write into a docker-compose .env file, where
// Docker Compose interpolates '$' — a symbol password there reaches the container
// mangled and no longer matches what we display to the user.
func generateSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = 'x'
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// The public self-host stack the compose edition installs.
const (
	composeRepoURL = "https://github.com/taikuri-infra/Zenith.git"
	composeBranch  = "main"
)

// runner executes shell commands on the install target — either the local
// machine (localRunner) or a remote host over SSH (*sshclient.Client). Both
// satisfy this interface via Run + Close.
type runner interface {
	Run(cmd string) (string, error)
	Close() error
}

// localRunner runs commands on the machine zen is invoked from.
type localRunner struct{}

func (localRunner) Run(cmd string) (string, error) {
	out, err := exec.Command("bash", "-lc", cmd).CombinedOutput()
	return string(out), err
}

func (localRunner) Close() error { return nil }

// newRunner returns a runner for the configured compose target.
func newRunner(cfg *Config) (runner, error) {
	if cfg.ComposeLocal {
		return localRunner{}, nil
	}
	return dialSSH(cfg)
}

func composeInstallDir(cfg *Config) string {
	if cfg.InstallDir != "" {
		return cfg.InstallDir
	}
	return "zenith"
}

// installDirRe restricts the checkout dir to shell-safe characters, since it is
// interpolated into commands run on the target (the runner needs a real shell
// for SSH parity, so metacharacters could otherwise inject).
var installDirRe = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

func validateInstallDir(dir string) error {
	if !installDirRe.MatchString(dir) {
		return fmt.Errorf("invalid install dir %q: only letters, digits and . _ - / are allowed", dir)
	}
	return nil
}

// GetComposeInstallSteps returns the ordered steps for the Compose (self-host)
// edition — the docker-compose stack on any Linux box, no Kubernetes. Mirrors
// GetInstallSteps; runSteps executes it identically (resume + dry-run reuse).
func GetComposeInstallSteps(cfg *Config) []Step {
	return []Step{
		{
			Name:        "Connect",
			Description: describeComposeTarget(cfg),
			Duration:    5 * time.Second,
			Action:      composeConnect,
		},
		{
			Name:        "Ensure Docker",
			Description: "Checking for Docker and Docker Compose...",
			Duration:    30 * time.Second,
			Action:      composeEnsureDocker,
		},
		{
			Name:        "Fetch stack",
			Description: "Fetching FreeZenith and generating secrets...",
			Duration:    20 * time.Second,
			Action:      composeFetchStack,
		},
		{
			Name:        "Start services",
			Description: "Pulling prebuilt images and starting the stack...",
			Duration:    90 * time.Second,
			Action:      composeUp,
		},
		{
			Name:        "Wait for health",
			Description: "Waiting for the API to become healthy...",
			Duration:    60 * time.Second,
			Action:      composeWaitHealth,
		},
		{
			Name:        "Save credentials",
			Description: "Saving login credentials...",
			Duration:    2 * time.Second,
			Action:      composeSaveCredentials,
		},
	}
}

func describeComposeTarget(cfg *Config) string {
	if cfg.ComposeLocal {
		return "Installing on this machine..."
	}
	return fmt.Sprintf("Connecting to %s...", cfg.SSHHost)
}

func composeConnect(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer r.Close()
	if out, err := r.Run("echo zenith-connect-ok"); err != nil {
		return fmt.Errorf("target not reachable: %w\n%s", err, out)
	}
	return nil
}

// dockerPrefix returns "sudo " if docker needs it on the target (a freshly
// installed docker group isn't active in the current shell), else "".
func dockerPrefix(r runner) string {
	if _, err := r.Run("docker info >/dev/null 2>&1"); err == nil {
		return ""
	}
	if _, err := r.Run("sudo -n docker info >/dev/null 2>&1"); err == nil {
		return "sudo "
	}
	return "sudo "
}

func composeEnsureDocker(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	defer r.Close()

	// Already present?
	if _, err := r.Run("command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1"); err == nil {
		return nil
	}
	sudo := ""
	if out, _ := r.Run("id -u"); strings.TrimSpace(out) != "0" {
		sudo = "sudo "
	}
	if out, err := r.Run("curl -fsSL https://get.docker.com | " + sudo + "sh"); err != nil {
		return fmt.Errorf("install docker: %w\n%s", err, out)
	}
	// Add the invoking user to the docker group for future logins (compose up
	// this run still uses the sudo fallback via dockerPrefix).
	_, _ = r.Run(sudo + `usermod -aG docker "$USER" 2>/dev/null || true`)
	return nil
}

func composeFetchStack(cfg *Config) error {
	if cfg.DryRun {
		if cfg.AdminPassword == "" {
			cfg.AdminPassword = generateSecret(20)
		}
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	defer r.Close()

	dir := composeInstallDir(cfg)
	if err := validateInstallDir(dir); err != nil {
		return err
	}
	clone := fmt.Sprintf(
		"if [ -d %s/.git ]; then git -C %s pull --ff-only; else git clone --depth 1 --branch %s %s %s; fi",
		dir, dir, composeBranch, composeRepoURL, dir)
	if out, err := r.Run(clone); err != nil {
		return fmt.Errorf("fetch stack: %w\n%s", err, out)
	}

	if cfg.AdminPassword == "" {
		cfg.AdminPassword = generateSecret(20)
	}
	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@localhost"
	}
	gidOut, _ := r.Run("getent group docker 2>/dev/null | cut -d: -f3 || stat -c '%g' /var/run/docker.sock 2>/dev/null || echo 999")
	gid := strings.TrimSpace(gidOut)
	if gid == "" {
		gid = "999"
	}

	env := buildComposeEnv(cfg, adminEmail, gid)
	enc := base64.StdEncoding.EncodeToString([]byte(env))
	// .env holds JWT/admin/DB secrets — create it owner-only (umask 077) so it
	// is never world-readable.
	if out, err := r.Run(fmt.Sprintf("umask 077 && (echo %s | base64 -d) > %s/.env", enc, dir)); err != nil {
		return fmt.Errorf("write .env: %w\n%s", err, out)
	}
	return nil
}

// buildComposeEnv renders the .env for the compose stack. The compose file uses
// ${VAR:-default} for everything, so only the security-relevant keys are set.
func buildComposeEnv(cfg *Config, adminEmail, dockerGID string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "JWT_SECRET=%s\n", generateSecret(48))
	fmt.Fprintf(&b, "ADMIN_EMAIL=%s\n", adminEmail)
	fmt.Fprintf(&b, "ADMIN_PASSWORD=%s\n", cfg.AdminPassword)
	fmt.Fprintf(&b, "DB_PASSWORD=%s\n", generateSecret(24))
	fmt.Fprintf(&b, "S3_ACCESS_KEY=zenith\n")
	fmt.Fprintf(&b, "S3_SECRET_KEY=%s\n", generateSecret(24))
	fmt.Fprintf(&b, "DOCKER_GID=%s\n", dockerGID)
	if d := cfg.Domain; d != "" && d != "localhost" {
		fmt.Fprintf(&b, "ZENITH_DOMAIN=%s\n", d)
		fmt.Fprintf(&b, "ACME_EMAIL=%s\n", adminEmail)
		fmt.Fprintf(&b, "NEXT_PUBLIC_API_URL=https://%s\n", d)
		fmt.Fprintf(&b, "CORS_ORIGINS=https://%s\n", d)
	}
	return b.String()
}

func composeUp(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	dir := composeInstallDir(cfg)
	if err := validateInstallDir(dir); err != nil {
		return err
	}
	prefix := dockerPrefix(r)
	profile := ""
	if d := cfg.Domain; d != "" && d != "localhost" {
		profile = "--profile tls "
	}
	cmd := fmt.Sprintf("cd %s && %sdocker compose %sup -d", dir, prefix, profile)
	if out, err := r.Run(cmd); err != nil {
		return fmt.Errorf("start services: %w\n%s", err, out)
	}
	return nil
}

func composeWaitHealth(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	deadline := time.Now().Add(3 * time.Minute)
	delay := 3 * time.Second
	for time.Now().Before(deadline) {
		if out, err := r.Run("curl -fsS http://localhost:8080/health"); err == nil && strings.Contains(out, "healthy") {
			return nil
		}
		time.Sleep(delay)
		if delay < 15*time.Second {
			delay += 2 * time.Second
		}
	}
	return fmt.Errorf("the API did not become healthy within 3 minutes (check: docker compose logs api)")
}

// ComposeUninstall stops and removes the compose stack on the target.
func ComposeUninstall(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	dir := composeInstallDir(cfg)
	if err := validateInstallDir(dir); err != nil {
		return err
	}
	prefix := dockerPrefix(r)
	cmd := fmt.Sprintf("cd %s && %sdocker compose --profile tls down -v", dir, prefix)
	if out, err := r.Run(cmd); err != nil {
		return fmt.Errorf("uninstall: %w\n%s", err, out)
	}
	return nil
}

func composeSaveCredentials(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	url := "http://localhost:3000"
	if d := cfg.Domain; d != "" && d != "localhost" {
		url = "https://" + d
	}
	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@localhost"
	}
	state := &installstate.State{
		Domain:            cfg.Domain,
		ServerIP:          cfg.SSHHost,
		MissionControlURL: url,
		CloudURL:          url,
		AdminUser:         adminEmail,
		Provider:          "compose",
		InstalledAt:       time.Now().UTC(),
	}
	_ = installstate.Save(state)
	return nil
}
