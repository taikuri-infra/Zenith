package install

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
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
	ServerIP        string
	Domain          string
	MissionControlURL string
	CloudURL        string
	AdminUser       string
	AdminPassword   string
	ClusterName     string // empty if no first cluster was created
	ClusterIP       string // empty if no first cluster was created
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
	MCProvider    ServerProvider
	HetznerToken  string // only if MCProvider == ProviderHetzner
	Region        string // only if MCProvider == ProviderHetzner
	ServerType    string // only if MCProvider == ProviderHetzner
	SSHHost       string // only if MCProvider == ProviderExisting
	SSHUser       string // only if MCProvider == ProviderExisting
	SSHKeyPath    string

	// Domain
	Domain string

	// DNS
	DNSProvider    DNSProvider
	CloudflareToken string // only if DNSProvider == DNSCloudflare

	// First Cluster (optional)
	WithCluster       bool
	ClusterProvider   ServerProvider // same options as MCProvider
	ClusterHetznerToken string      // reuses MCProvider token if same provider
	ClusterRegion     string
	ClusterServerType string
	ClusterSSHHost    string // only if ClusterProvider == ProviderExisting
	ClusterSSHUser    string // only if ClusterProvider == ProviderExisting
}

// Regions available for Hetzner Cloud.
var Regions = []Region{
	{ID: "nbg1", Name: "Nuremberg", Country: "Germany"},
	{ID: "fsn1", Name: "Falkenstein", Country: "Germany"},
	{ID: "hel1", Name: "Helsinki", Country: "Finland"},
	{ID: "ash", Name: "Ashburn", Country: "USA"},
}

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

type ServerType struct {
	ID          string
	Name        string
	CPUs        int
	RAM         int
	Price       float64
	Description string
}

// ValidateToken checks if the Hetzner API token is valid format.
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

// BuildResult constructs the installation result from config.
// In production this would query the actual provisioned resources.
func BuildResult(cfg *Config) *InstallResult {
	ip := "203.0.113.42" // placeholder
	if cfg.MCProvider == ProviderExisting {
		ip = cfg.SSHHost
	}

	result := &InstallResult{
		ServerIP:          ip,
		Domain:            cfg.Domain,
		MissionControlURL: fmt.Sprintf("https://mission.%s", cfg.Domain),
		CloudURL:          fmt.Sprintf("https://cloud.%s", cfg.Domain),
		AdminUser:         "admin",
		AdminPassword:     generatePassword(16),
	}

	if cfg.WithCluster {
		result.ClusterName = "cluster-01"
		result.ClusterIP = "203.0.113.100"
	}

	return result
}

// --- Placeholder provisioning functions ---
// These will be replaced with real implementations.

func provisionHetznerServer(cfg *Config) error {
	// TODO: Use hcloud API to create server
	// hcloud server create --name zenith-mc --type cx22 --image ubuntu-22.04 --location fsn1
	time.Sleep(100 * time.Millisecond) // simulate work
	return nil
}

func verifyExistingServer(cfg *Config) error {
	// TODO: SSH into server and verify it meets requirements
	time.Sleep(50 * time.Millisecond)
	return nil
}

func installPlatform(cfg *Config) error {
	// TODO: SSH + install k3s, CAPI, Helm charts
	time.Sleep(100 * time.Millisecond)
	return nil
}

func configureDNS(cfg *Config) error {
	// TODO: If Cloudflare, use API to create DNS records
	// Otherwise, print manual DNS instructions
	time.Sleep(50 * time.Millisecond)
	return nil
}

func issueSSL(cfg *Config) error {
	// TODO: cert-manager will handle this, just wait for certificate readiness
	time.Sleep(50 * time.Millisecond)
	return nil
}

func waitForHealthy(cfg *Config) error {
	// TODO: Poll Mission Control health endpoint
	time.Sleep(100 * time.Millisecond)
	return nil
}

func createFirstCluster(cfg *Config) error {
	// TODO: POST to Mission Control API with cluster config
	time.Sleep(100 * time.Millisecond)
	return nil
}

func generatePassword(length int) string {
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
