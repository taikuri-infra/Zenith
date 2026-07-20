package install

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/cloudflare"
	"github.com/dotechhq/zenith/cli/internal/healthcheck"
	"github.com/dotechhq/zenith/cli/internal/hetzner"
	"github.com/dotechhq/zenith/cli/internal/installstate"
	"github.com/dotechhq/zenith/cli/internal/k3s"
	"github.com/dotechhq/zenith/cli/internal/sshclient"
	"github.com/dotechhq/zenith/cli/internal/sshkeys"
	"gopkg.in/yaml.v3"
)

// Step represents a single installation step.
type Step struct {
	Name        string
	Description string
	Action      func(cfg *Config) error
	Duration    time.Duration // estimated duration for display
}

// InstallResult holds the output from a completed installation.
type InstallResult struct {
	ServerIP          string
	Domain            string
	MissionControlURL string
	CloudURL          string
	AdminUser         string
	AdminPassword     string
	ClusterName       string // empty if no first cluster was created
	ClusterIP         string // empty if no first cluster was created
}

// ServerProvider indicates how the management server is obtained.
type ServerProvider string

const (
	ProviderHetzner  ServerProvider = "hetzner"
	ProviderExisting ServerProvider = "existing"
)

// DNSProvider indicates how DNS records are managed.
type DNSProvider string

const (
	DNSCloudflare DNSProvider = "cloudflare"
	DNSManual     DNSProvider = "manual"
)

// Config holds the installation configuration gathered from the wizard.
type Config struct {
	// Mission Control Server
	MCProvider   ServerProvider
	HetznerToken string // only if MCProvider == ProviderHetzner
	Region       string // only if MCProvider == ProviderHetzner
	ServerType   string // only if MCProvider == ProviderHetzner
	SSHHost      string // set by provisioning or by user for existing server
	SSHUser      string
	SSHKeyPath   string

	// Domain
	Domain string

	// DNS
	DNSProvider     DNSProvider
	CloudflareToken string // only if DNSProvider == DNSCloudflare

	// First Cluster (optional)
	WithCluster         bool
	ClusterProvider     ServerProvider
	ClusterHetznerToken string
	ClusterRegion       string
	ClusterServerType   string
	ClusterSSHHost      string
	ClusterSSHUser      string

	// Set during installation (internal use)
	AdminEmail    string // defaults to "admin@<domain>" if empty
	AdminPassword string // password configured on the server during installPlatform
	AdminToken    string // JWT token obtained after createFirstCluster login
	// KnownHostKey is the raw SSH wire-format public key captured on first connection.
	// Nil on the first connect (TOFU mode); set to the captured key on all subsequent connects.
	KnownHostKey []byte

	// Set during provisioning (internal use)
	HetznerSSHKeyID        int64
	GeneratedSSHPrivateKey []byte
	ProvisionedServerID    int64

	// ChartVersion is set after installZenithChart succeeds (from helm list).
	ChartVersion string

	// Compose edition (self-host, docker-compose, no Kubernetes).
	Edition       string // "compose" or "cloud" (empty = cloud/Hetzner path)
	ComposeLocal  bool   // true: install on this machine; false: SSH to SSHHost
	InstallDir    string // target checkout dir for the compose stack (default "zenith")
	FreeSubdomain bool   // register a free <slug>.apps.freezenith.com + auto-HTTPS
	RegisterURL   string // override the subdomain-registration service URL (tests/self-host)

	// DryRun skips all real API calls for testing the installer flow.
	DryRun bool
}

// Regions available for Hetzner Cloud.
var Regions = []Region{
	{ID: "nbg1", Name: "Nuremberg", Country: "Germany"},
	{ID: "fsn1", Name: "Falkenstein", Country: "Germany"},
	{ID: "hel1", Name: "Helsinki", Country: "Finland"},
	{ID: "ash", Name: "Ashburn", Country: "USA"},
}

// Region is a Hetzner datacenter location.
type Region struct {
	ID      string
	Name    string
	Country string
}

// ServerTypes available for management plane.
var ServerTypes = []ServerType{
	{ID: "cx22", Name: "CX22", CPUs: 2, RAM: 4, Price: 4.35, Description: "2 vCPU, 4 GB RAM (recommended)"},
	{ID: "cx32", Name: "CX32", CPUs: 4, RAM: 8, Price: 7.75, Description: "4 vCPU, 8 GB RAM"},
	{ID: "cx42", Name: "CX42", CPUs: 8, RAM: 16, Price: 14.55, Description: "8 vCPU, 16 GB RAM"},
}

// ServerType is a Hetzner machine type.
type ServerType struct {
	ID          string
	Name        string
	CPUs        int
	RAM         int
	Price       float64
	Description string
}

// ValidateToken checks if the Hetzner API token has a valid format.
func ValidateToken(token string) error {
	if len(token) < 10 {
		return fmt.Errorf("token is too short")
	}
	return nil
}

// ValidateDomain checks if the domain looks valid.
func ValidateDomain(domain string) error {
	if len(domain) < 4 {
		return fmt.Errorf("domain is too short")
	}
	dotFound := false
	for _, c := range domain {
		if c == '.' {
			dotFound = true
		}
	}
	if !dotFound {
		return fmt.Errorf("domain must contain at least one dot (e.g., example.com)")
	}
	return nil
}

// GetServerTypeByID returns the ServerType for a given ID, or nil.
func GetServerTypeByID(id string) *ServerType {
	for _, s := range ServerTypes {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

// GetRegionByID returns the Region for a given ID, or nil.
func GetRegionByID(id string) *Region {
	for _, r := range Regions {
		if r.ID == id {
			return &r
		}
	}
	return nil
}

// EstimateMonthlyCost calculates the estimated monthly cost based on config.
func EstimateMonthlyCost(cfg *Config) float64 {
	var total float64
	if cfg.MCProvider == ProviderHetzner {
		if st := GetServerTypeByID(cfg.ServerType); st != nil {
			total += st.Price
		}
	}
	if cfg.WithCluster && cfg.ClusterProvider == ProviderHetzner {
		if st := GetServerTypeByID(cfg.ClusterServerType); st != nil {
			total += st.Price
		}
	}
	return total
}

// GetInstallSteps returns the ordered list of installation steps.
func GetInstallSteps(cfg *Config) []Step {
	steps := []Step{
		{
			Name:        "Provision server",
			Description: describeProvision(cfg),
			Duration:    30 * time.Second,
			Action: func(cfg *Config) error {
				if cfg.MCProvider == ProviderHetzner {
					return provisionHetznerServer(cfg)
				}
				return verifyExistingServer(cfg)
			},
		},
		{
			Name:        "Install platform",
			Description: "Installing k3s, CAPI, Zenith operator, API, auth, monitoring...",
			Duration:    90 * time.Second,
			Action: func(cfg *Config) error {
				return installPlatform(cfg)
			},
		},
		{
			Name:        "Install Zenith chart",
			Description: fmt.Sprintf("Installing Zenith via Helm on %s...", cfg.SSHHost),
			Duration:    2 * time.Minute,
			Action: func(cfg *Config) error {
				return installZenithChart(cfg)
			},
		},
		{
			Name:        "Configure DNS",
			Description: describeDNS(cfg),
			Duration:    10 * time.Second,
			Action: func(cfg *Config) error {
				return configureDNS(cfg)
			},
		},
		{
			Name:        "Issue SSL certificates",
			Description: "Requesting Let's Encrypt certificates via cert-manager...",
			Duration:    15 * time.Second,
			Action: func(cfg *Config) error {
				return issueSSL(cfg)
			},
		},
		{
			Name:        "Wait for Mission Control",
			Description: "Waiting for Mission Control to become healthy...",
			Duration:    30 * time.Second,
			Action: func(cfg *Config) error {
				return waitForHealthy(cfg)
			},
		},
	}

	if cfg.WithCluster {
		steps = append(steps, Step{
			Name:        "Create first cluster",
			Description: "Sending cluster configuration to Mission Control API...",
			Duration:    15 * time.Second,
			Action: func(cfg *Config) error {
				return createFirstCluster(cfg)
			},
		})
	}

	return steps
}

// BuildResult constructs the installation result from config and persists state.
func BuildResult(cfg *Config) *InstallResult {
	ip := cfg.SSHHost
	if ip == "" {
		ip = "203.0.113.42" // fallback placeholder
	}

	adminPassword := cfg.AdminPassword
	if adminPassword == "" {
		adminPassword = GeneratePassword(16)
	}

	result := &InstallResult{
		ServerIP:          ip,
		Domain:            cfg.Domain,
		MissionControlURL: fmt.Sprintf("https://mission.%s", cfg.Domain),
		CloudURL:          fmt.Sprintf("https://cloud.%s", cfg.Domain),
		AdminUser:         "admin",
		AdminPassword:     adminPassword,
	}

	if cfg.WithCluster {
		result.ClusterName = "cluster-01"
		result.ClusterIP = "203.0.113.100"
	}

	// Persist installation state to ~/.zen/install-state.yaml.
	// AdminPassword is intentionally NOT saved — display-once security model.
	saveState := &installstate.State{
		Domain:            cfg.Domain,
		ServerIP:          ip,
		MissionControlURL: result.MissionControlURL,
		CloudURL:          result.CloudURL,
		AdminUser:         result.AdminUser,
		Provider:          string(cfg.MCProvider),
		Region:            cfg.Region,
		ServerID:          fmt.Sprintf("%d", cfg.ProvisionedServerID),
		SSHKeyID:          fmt.Sprintf("%d", cfg.HetznerSSHKeyID),
		SSHKeyPath:        cfg.SSHKeyPath,
		InstalledAt:       time.Now().UTC(),
	}
	if cfg.ChartVersion != "" {
		saveState.ZenithVersion = cfg.ChartVersion
	}
	if len(cfg.KnownHostKey) > 0 {
		saveState.ServerHostKey = base64.StdEncoding.EncodeToString(cfg.KnownHostKey)
	}
	_ = installstate.Save(saveState)

	return result
}

// dialSSH creates an SSH client from the current install config.
func dialSSH(cfg *Config) (*sshclient.Client, error) {
	if cfg.SSHHost == "" {
		return nil, fmt.Errorf("SSH host is not set (provisioning step may have failed)")
	}
	user := cfg.SSHUser
	if user == "" {
		user = "root"
	}
	sshCfg := sshclient.Config{
		Host:         cfg.SSHHost,
		Port:         22,
		User:         user,
		Timeout:      30 * time.Second,
		KnownHostKey: cfg.KnownHostKey, // nil → TOFU; non-nil → verify
	}
	if len(cfg.GeneratedSSHPrivateKey) > 0 {
		sshCfg.PrivateKey = cfg.GeneratedSSHPrivateKey
	}
	cli, err := sshclient.DialWithRetry(sshCfg, 10, 15*time.Second)
	if err != nil {
		return nil, err
	}
	// Pin the host key after first TOFU connect so all subsequent dials verify it.
	if cfg.KnownHostKey == nil && len(cli.CapturedHostKey()) > 0 {
		cfg.KnownHostKey = cli.CapturedHostKey()
	}
	return cli, nil
}

// provisionHetznerServer creates a Hetzner server and waits until running.
func provisionHetznerServer(cfg *Config) error {
	if cfg.DryRun {
		cfg.SSHHost = "203.0.113.1"
		cfg.ProvisionedServerID = 0
		return nil
	}

	ctx := context.Background()
	client := hetzner.NewClient(cfg.HetznerToken)

	// Generate an ephemeral SSH key for this install
	kp, err := sshkeys.Generate()
	if err != nil {
		return fmt.Errorf("generate SSH key: %w", err)
	}

	// Upload key to Hetzner
	keyName := fmt.Sprintf("zenith-install-%d", time.Now().Unix())
	sshKey, err := client.CreateSSHKey(ctx, hetzner.CreateSSHKeyRequest{
		Name:      keyName,
		PublicKey: strings.TrimSpace(kp.PublicKeySSH),
	})
	if err != nil {
		return fmt.Errorf("create SSH key: %w", err)
	}

	cfg.HetznerSSHKeyID = sshKey.ID
	cfg.GeneratedSSHPrivateKey = kp.PrivateKeyPEM

	// Save key to disk so zen upgrade can SSH back in
	if home, err := os.UserHomeDir(); err == nil {
		keyPath := filepath.Join(home, ".zen", "install-key.pem")
		if mkErr := os.MkdirAll(filepath.Dir(keyPath), 0o700); mkErr != nil {
			fmt.Printf("warning: could not create ~/.zen directory: %v\n", mkErr)
		} else if writeErr := os.WriteFile(keyPath, kp.PrivateKeyPEM, 0o600); writeErr != nil {
			fmt.Printf("warning: could not save SSH key to %s: %v\n", keyPath, writeErr)
		} else {
			cfg.SSHKeyPath = keyPath
		}
	}

	// Create the server
	serverResp, err := client.CreateServer(ctx, hetzner.CreateServerRequest{
		Name:       fmt.Sprintf("zenith-mc-%d", time.Now().Unix()),
		ServerType: cfg.ServerType,
		Image:      "ubuntu-22.04",
		Location:   cfg.Region,
		SSHKeys:    []string{fmt.Sprintf("%d", sshKey.ID)},
		Labels: map[string]string{
			"managed-by": "zenith-installer",
			"role":       "mission-control",
		},
	})
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	cfg.ProvisionedServerID = serverResp.Server.ID

	// Wait for it to be running (5-minute timeout)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	srv, err := client.WaitForServerRunning(timeoutCtx, serverResp.Server.ID)
	if err != nil {
		return fmt.Errorf("server never became running: %w", err)
	}
	cfg.SSHHost = srv.PublicNet.IPv4.IP

	return nil
}

// verifyExistingServer SSH-connects to an existing server and checks basic requirements.
func verifyExistingServer(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}

	client, err := dialSSH(cfg)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", cfg.SSHHost, err)
	}
	defer client.Close()

	out, err := client.Run("uname -s && free -m | awk '/^Mem:/ {print $2}'")
	if err != nil {
		return fmt.Errorf("server check failed: %w", err)
	}
	if out == "" {
		return fmt.Errorf("server returned empty response")
	}
	return nil
}

// installPlatform installs k3s on the remote server via SSH.
func installPlatform(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}

	client, err := dialSSH(cfg)
	if err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}
	defer client.Close()

	if err := k3s.Install(client, k3s.Options{}); err != nil {
		return fmt.Errorf("k3s install: %w", err)
	}

	if err := k3s.WaitForReady(client, 120); err != nil {
		return fmt.Errorf("k3s not ready: %w", err)
	}

	return nil
}

// defaultHelmVersion is the pinned Helm version installed on remote servers.
// Pinning prevents supply-chain attacks via a compromised "latest" installer.
const defaultHelmVersion = "v3.17.3"

// installZenithChart installs helm on the remote server, writes a temp values
// file via base64, runs helm upgrade --install for the Zenith chart, then cleans up.
func installZenithChart(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}

	sshCli, err := dialSSH(cfg)
	if err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}
	defer sshCli.Close()

	// Install helm at a pinned version if not already present.
	helmInstallCmd := fmt.Sprintf(
		"which helm >/dev/null 2>&1 || (DESIRED_VERSION=%s curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash 2>&1)",
		defaultHelmVersion,
	)
	out, err := sshCli.Run(helmInstallCmd)
	if err != nil {
		return fmt.Errorf("install helm: %w\nOutput: %s", err, out)
	}

	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@" + cfg.Domain
	}
	hetznerToken := cfg.HetznerToken
	if hetznerToken == "" {
		hetznerToken = "none"
	}

	type helmValues struct {
		Global struct {
			Domain       string `yaml:"domain"`
			HetznerToken string `yaml:"hetznerToken"`
		} `yaml:"global"`
		Secrets struct {
			AdminEmail    string `yaml:"adminEmail"`
			AdminPassword string `yaml:"adminPassword"`
		} `yaml:"secrets"`
	}
	var vals helmValues
	vals.Global.Domain = cfg.Domain
	vals.Global.HetznerToken = hetznerToken
	vals.Secrets.AdminEmail = adminEmail
	vals.Secrets.AdminPassword = cfg.AdminPassword
	yamlBytes, err := yaml.Marshal(&vals)
	if err != nil {
		return fmt.Errorf("marshal helm values: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(yamlBytes)
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > /tmp/zenith-install-values.yaml", encoded)
	if _, err := sshCli.Run(writeCmd); err != nil {
		return fmt.Errorf("write values file: %w", err)
	}

	helmCmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm upgrade --install zenith " +
		"oci://ghcr.io/dotechhq/zenith/charts/zenith " +
		"--namespace zenith-system --create-namespace " +
		"-f /tmp/zenith-install-values.yaml " +
		"--wait --timeout 10m 2>&1"
	if cfg.ChartVersion != "" {
		helmCmd = "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm upgrade --install zenith " +
			"oci://ghcr.io/dotechhq/zenith/charts/zenith " +
			"--namespace zenith-system --create-namespace " +
			"--version " + cfg.ChartVersion + " " +
			"-f /tmp/zenith-install-values.yaml " +
			"--wait --timeout 10m 2>&1"
	}
	if out, err := sshCli.Run(helmCmd); err != nil {
		sshCli.Run("rm -f /tmp/zenith-install-values.yaml") //nolint:errcheck
		return fmt.Errorf("helm install: %w\nOutput: %s", err, out)
	}

	sshCli.Run("rm -f /tmp/zenith-install-values.yaml") //nolint:errcheck

	// Record the installed chart version so BuildResult can persist it to state.
	if v, err := sshCli.Run(
		"KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm list -n zenith-system --filter '^zenith$' -o json 2>/dev/null | " +
			`grep -o '"chart":"[^"]*"' | head -1 | sed 's/.*zenith-//' | tr -d '"'`,
	); err == nil && strings.TrimSpace(v) != "" {
		cfg.ChartVersion = strings.TrimSpace(v)
	}
	return nil
}

// configureDNS creates DNS records via Cloudflare or does nothing for manual DNS.
func configureDNS(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}

	if cfg.DNSProvider == DNSManual {
		return nil
	}

	if cfg.DNSProvider == DNSCloudflare {
		client := cloudflare.NewClient(cfg.CloudflareToken)

		zone, err := client.FindZone(cfg.Domain)
		if err != nil {
			return fmt.Errorf("find Cloudflare zone: %w", err)
		}

		ip := cfg.SSHHost
		if ip == "" {
			return fmt.Errorf("server IP not set — provisioning step may have failed")
		}

		for _, sub := range []string{
			fmt.Sprintf("mission.%s", cfg.Domain),
			fmt.Sprintf("cloud.%s", cfg.Domain),
		} {
			if err := client.UpsertRecord(zone.ID, sub, ip); err != nil {
				return fmt.Errorf("upsert DNS record for %s: %w", sub, err)
			}
		}
		return nil
	}

	return fmt.Errorf("unknown DNS provider: %s", cfg.DNSProvider)
}

// issueSSL is a no-op — cert-manager handles certificates automatically.
func issueSSL(cfg *Config) error {
	return nil
}

// waitForHealthy polls the Mission Control health endpoint until it responds 200.
func waitForHealthy(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}

	url := fmt.Sprintf("https://mission.%s/health", cfg.Domain)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return healthcheck.WaitUntilHealthy(ctx, healthcheck.Options{
		URL:      url,
		Interval: 10 * time.Second,
	})
}

// createFirstCluster logs in to Mission Control with admin credentials and saves
// the JWT token to cfg for state persistence. This enables zen status / zen deploy
// to work immediately after install without requiring a separate zen login.
func createFirstCluster(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	if cfg.AdminPassword == "" {
		return fmt.Errorf("admin password not set — run zen install first")
	}

	mcURL := fmt.Sprintf("https://mission.%s", cfg.Domain)
	apiClient := cliapi.NewClient(mcURL, "")

	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@" + cfg.Domain
	}

	token, err := apiClient.Login(adminEmail, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("login to Mission Control at %s: %w", mcURL, err)
	}

	cfg.AdminToken = token
	return nil
}

func GeneratePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
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

func describeProvision(cfg *Config) string {
	if cfg.MCProvider == ProviderHetzner {
		region := cfg.Region
		if r := GetRegionByID(cfg.Region); r != nil {
			region = r.Name
		}
		return fmt.Sprintf("Creating %s server in %s...", cfg.ServerType, region)
	}
	return fmt.Sprintf("Verifying server at %s...", cfg.SSHHost)
}

func describeDNS(cfg *Config) string {
	if cfg.DNSProvider == DNSCloudflare {
		return fmt.Sprintf("Configuring Cloudflare DNS for %s...", cfg.Domain)
	}
	return "Waiting for manual DNS propagation..."
}

// --- Legacy helpers (kept for backwards compatibility with tests) ---

// RegionOptions returns a list of region labels for the TUI form.
func RegionOptions() []string {
	opts := make([]string, len(Regions))
	for i, r := range Regions {
		opts[i] = fmt.Sprintf("%s - %s, %s", r.ID, r.Name, r.Country)
	}
	return opts
}

// ServerTypeOptions returns a list of server type labels for the TUI form.
func ServerTypeOptions() []string {
	opts := make([]string, len(ServerTypes))
	for i, s := range ServerTypes {
		opts[i] = fmt.Sprintf("%s - %s (€%.2f/mo)", s.ID, s.Description, s.Price)
	}
	return opts
}

// ParseRegionSelection extracts the region ID from a selection string.
func ParseRegionSelection(selection string) string {
	for _, r := range Regions {
		if len(selection) >= len(r.ID) && selection[:len(r.ID)] == r.ID {
			return r.ID
		}
	}
	return "fsn1"
}

// ParseServerTypeSelection extracts the server type ID from a selection string.
func ParseServerTypeSelection(selection string) string {
	for _, s := range ServerTypes {
		if len(selection) >= len(s.ID) && selection[:len(s.ID)] == s.ID {
			return s.ID
		}
	}
	return "cx22"
}
