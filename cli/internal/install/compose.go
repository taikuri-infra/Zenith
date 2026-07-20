package install

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dotechhq/zenith/cli/internal/cloudflare"
	"github.com/dotechhq/zenith/cli/internal/installstate"
)

// freeSubdomainBase is the domain FreeZenith operates for zero-DNS installs:
// every free install gets <slug>.apps.freezenith.com with an auto Let's Encrypt cert.
const freeSubdomainBase = "apps.freezenith.com"

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

// detectPublicIP asks the target for its own public IP (tries several providers).
func detectPublicIP(r runner) (string, error) {
	for _, u := range []string{"https://api.ipify.org", "https://ifconfig.me", "https://icanhazip.com"} {
		out, err := r.Run("curl -fsS --max-time 8 " + u)
		ip := strings.TrimSpace(out)
		if err == nil && net.ParseIP(ip) != nil {
			return ip, nil
		}
	}
	return "", fmt.Errorf("could not detect the target's public IP")
}

// defaultRegisterURL is the FreeZenith-operated subdomain registration service.
// IT holds the Cloudflare token; the customer's box only ever receives a hostname.
const defaultRegisterURL = "https://register.freezenith.com"

func registerURL(cfg *Config) string {
	if cfg.RegisterURL != "" {
		return cfg.RegisterURL
	}
	return defaultRegisterURL
}

func postJSON(url, token string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return (&http.Client{Timeout: 20 * time.Second}).Do(req)
}

// registerSubdomain asks the registration service for a free subdomain pointing at
// ip and returns the assigned hostname. The Cloudflare token stays on the service.
func registerSubdomain(serviceURL, token, ip string) (string, error) {
	body, _ := json.Marshal(map[string]string{"ip": ip})
	resp, err := postJSON(strings.TrimRight(serviceURL, "/")+"/register", token, body)
	if err != nil {
		return "", fmt.Errorf("contact registration service: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Hostname string `json:"hostname"`
		Error    string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK || out.Hostname == "" {
		msg := out.Error
		if msg == "" {
			msg = resp.Status
		}
		return "", fmt.Errorf("registration failed: %s", msg)
	}
	return out.Hostname, nil
}

// releaseSubdomain asks the registration service to remove a subdomain (uninstall).
func releaseSubdomain(serviceURL, token, hostname string) error {
	body, _ := json.Marshal(map[string]string{"hostname": hostname})
	resp, err := postJSON(strings.TrimRight(serviceURL, "/")+"/release", token, body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// composeRegisterSubdomain reserves a free <slug>.apps.freezenith.com via the
// registration service and sets cfg.Domain so the rest of the flow serves HTTPS
// there (Traefik + Let's Encrypt) — no DNS knowledge, and no token on this box.
func composeRegisterSubdomain(cfg *Config) error {
	if cfg.DryRun {
		if cfg.Domain == "" {
			cfg.Domain = "swift-otter-demo." + freeSubdomainBase
		}
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	ip, err := detectPublicIP(r)
	if err != nil {
		return err
	}
	host, err := registerSubdomain(registerURL(cfg), cfg.RegisterToken, ip)
	if err != nil {
		return err
	}
	cfg.Domain = host
	return nil
}

// GetComposeInstallSteps returns the ordered steps for the Compose (self-host)
// edition — the docker-compose stack on any Linux box, no Kubernetes. Mirrors
// GetInstallSteps; runSteps executes it identically (resume + dry-run reuse).
func GetComposeInstallSteps(cfg *Config) []Step {
	steps := []Step{
		{Name: "Connect", Description: describeComposeTarget(cfg), Duration: 5 * time.Second, Action: composeConnect},
	}
	if cfg.FreeSubdomain {
		steps = append(steps, Step{
			Name:        "Register subdomain",
			Description: "Reserving a free <slug>.apps.freezenith.com with HTTPS...",
			Duration:    10 * time.Second,
			Action:      composeRegisterSubdomain,
		})
	}
	steps = append(steps,
		Step{Name: "Ensure Docker", Description: "Checking for Docker and Docker Compose...", Duration: 30 * time.Second, Action: composeEnsureDocker},
		Step{Name: "Fetch stack", Description: "Fetching FreeZenith and generating secrets...", Duration: 20 * time.Second, Action: composeFetchStack},
	)
	if needsCustomDNS(cfg) {
		steps = append(steps, Step{
			Name:        "Configure DNS",
			Description: fmt.Sprintf("Pointing %s at this server...", cfg.Domain),
			Duration:    30 * time.Second,
			Action:      composeConfigureDNS,
		})
	}
	steps = append(steps,
		Step{Name: "Start services", Description: "Pulling prebuilt images and starting the stack...", Duration: 90 * time.Second, Action: composeUp},
		Step{Name: "Wait for health", Description: "Waiting for the API to become healthy...", Duration: 60 * time.Second, Action: composeWaitHealth},
		Step{Name: "Save credentials", Description: "Saving login credentials...", Duration: 2 * time.Second, Action: composeSaveCredentials},
	)
	return steps
}

// needsCustomDNS is true for a bring-your-own public domain (not localhost, not a
// free FreeZenith subdomain) — the customer's own domain must be pointed at the box.
func needsCustomDNS(cfg *Config) bool {
	return !cfg.FreeSubdomain && cfg.Domain != "" && cfg.Domain != "localhost"
}

// composeConfigureDNS makes sure the customer's domain points at the box before
// Traefik requests a Let's Encrypt cert. FreeZenith never holds the customer's DNS
// credentials, so by default it guides them to add the A record and waits for it;
// if they supplied a Cloudflare token for their own zone, it creates the record.
func composeConfigureDNS(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	r, err := newRunner(cfg)
	if err != nil {
		return err
	}
	ip, err := detectPublicIP(r)
	r.Close()
	if err != nil {
		return err
	}

	// Automated path: create the A record in the customer's own Cloudflare zone.
	if cfg.DNSProvider == DNSCloudflare && cfg.CloudflareToken != "" {
		cf := cloudflare.NewClient(cfg.CloudflareToken)
		zone, zerr := cf.FindZone(cfg.Domain)
		if zerr != nil {
			return fmt.Errorf("find your Cloudflare zone for %s: %w", cfg.Domain, zerr)
		}
		if uerr := cf.UpsertRecord(zone.ID, cfg.Domain, ip); uerr != nil {
			return fmt.Errorf("create A record %s -> %s: %w", cfg.Domain, ip, uerr)
		}
	} else if !domainResolvesTo(cfg.Domain, ip) {
		// Manual path: the record must already point at this box.
		return fmt.Errorf(
			"%s is not pointed at this server yet.\n"+
				"    Add this DNS record at your provider, then re-run with --resume:\n\n"+
				"        A   %s   ->   %s\n",
			cfg.Domain, cfg.Domain, ip)
	}

	// Wait for the record to resolve before HTTPS issuance (bounded).
	deadline := time.Now().Add(5 * time.Minute)
	delay := 5 * time.Second
	for {
		if domainResolvesTo(cfg.Domain, ip) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not resolve to %s in time (DNS may still be propagating — re-run with --resume)", cfg.Domain, ip)
		}
		time.Sleep(delay)
	}
}

// domainResolvesTo reports whether domain currently resolves to ip.
func domainResolvesTo(domain, ip string) bool {
	addrs, err := net.LookupHost(domain)
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if a == ip {
			return true
		}
	}
	return false
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

	// Release the free subdomain via the registration service if one was used.
	if strings.HasSuffix(cfg.Domain, "."+freeSubdomainBase) {
		_ = releaseSubdomain(registerURL(cfg), cfg.RegisterToken, cfg.Domain)
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
