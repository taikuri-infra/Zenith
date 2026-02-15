package install

import (
	"fmt"
	"time"
)

// Step represents a single installation step.
type Step struct {
	Name        string
	Description string
	Action      func(cfg *Config) error
	Duration    time.Duration // estimated duration for display
}

// Config holds the installation configuration gathered from the wizard.
type Config struct {
	HetznerToken string
	ServerType   string
	Region       string
	SSHKeyPath   string
}

// Regions available for Hetzner Cloud.
var Regions = []Region{
	{ID: "fsn1", Name: "Falkenstein", Country: "Germany"},
	{ID: "nbg1", Name: "Nuremberg", Country: "Germany"},
	{ID: "hel1", Name: "Helsinki", Country: "Finland"},
	{ID: "ash", Name: "Ashburn", Country: "USA"},
	{ID: "hil", Name: "Hillsboro", Country: "USA"},
}

type Region struct {
	ID      string
	Name    string
	Country string
}

// ServerTypes available for management plane.
var ServerTypes = []ServerType{
	{ID: "cx22", Name: "CX22", CPUs: 2, RAM: 4, Price: 4.49, Description: "2 vCPU, 4 GB RAM (recommended)"},
	{ID: "cx32", Name: "CX32", CPUs: 4, RAM: 8, Price: 7.49, Description: "4 vCPU, 8 GB RAM"},
	{ID: "cx42", Name: "CX42", CPUs: 8, RAM: 16, Price: 15.49, Description: "8 vCPU, 16 GB RAM"},
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

// GetInstallSteps returns the ordered list of installation steps.
func GetInstallSteps(cfg *Config) []Step {
	return []Step{
		{
			Name:        "Validate Hetzner token",
			Description: "Checking API access...",
			Duration:    2 * time.Second,
			Action: func(cfg *Config) error {
				return ValidateToken(cfg.HetznerToken)
			},
		},
		{
			Name:        "Create management server",
			Description: fmt.Sprintf("Provisioning %s in %s...", cfg.ServerType, cfg.Region),
			Duration:    30 * time.Second,
			Action: func(cfg *Config) error {
				// In production: hcloud server create
				return nil
			},
		},
		{
			Name:        "Install k3s",
			Description: "Setting up lightweight Kubernetes...",
			Duration:    45 * time.Second,
			Action: func(cfg *Config) error {
				// In production: SSH + k3s install
				return nil
			},
		},
		{
			Name:        "Install Cluster API",
			Description: "Setting up CAPI + CAPH...",
			Duration:    30 * time.Second,
			Action: func(cfg *Config) error {
				// In production: clusterctl init
				return nil
			},
		},
		{
			Name:        "Deploy Zenith platform",
			Description: "Installing operator, API, auth, monitoring...",
			Duration:    45 * time.Second,
			Action: func(cfg *Config) error {
				// In production: helm install zenith
				return nil
			},
		},
		{
			Name:        "Configure DNS & SSL",
			Description: "Setting up domains and certificates...",
			Duration:    15 * time.Second,
			Action: func(cfg *Config) error {
				// In production: cert-manager + DNS
				return nil
			},
		},
		{
			Name:        "Create admin account",
			Description: "Generating admin credentials...",
			Duration:    5 * time.Second,
			Action: func(cfg *Config) error {
				// In production: create admin user
				return nil
			},
		},
	}
}

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
