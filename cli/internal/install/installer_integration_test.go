package install

import (
	"testing"
)

func TestInstallDryRun_FullFlow(t *testing.T) {
	cfg := &Config{
		MCProvider:   ProviderHetzner,
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
		Domain:       "example.com",
		DNSProvider:  DNSManual,
		DryRun:       true,
	}

	steps := GetInstallSteps(cfg)
	if len(steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(steps))
	}

	for i, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("step %d (%s) failed in dry-run: %v", i, step.Name, err)
		}
	}

	// After dry-run provisioning, SSHHost should be set
	if cfg.SSHHost == "" {
		t.Error("expected SSHHost to be set after dry-run provisioning")
	}
}

func TestInstallDryRun_WithCluster(t *testing.T) {
	cfg := &Config{
		MCProvider:        ProviderHetzner,
		HetznerToken:      "test-token-1234567890",
		ServerType:        "cx22",
		Region:            "fsn1",
		Domain:            "example.com",
		DNSProvider:       DNSManual,
		WithCluster:       true,
		ClusterProvider:   ProviderHetzner,
		ClusterServerType: "cx22",
		ClusterRegion:     "fsn1",
		DryRun:            true,
	}

	steps := GetInstallSteps(cfg)
	if len(steps) != 6 {
		t.Fatalf("expected 6 steps with cluster, got %d", len(steps))
	}

	for i, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("step %d (%s) failed in dry-run: %v", i, step.Name, err)
		}
	}
}

func TestInstallDryRun_ExistingServer(t *testing.T) {
	cfg := &Config{
		MCProvider:  ProviderExisting,
		SSHHost:     "10.0.0.1",
		SSHUser:     "root",
		Domain:      "example.com",
		DNSProvider: DNSManual,
		DryRun:      true,
	}

	steps := GetInstallSteps(cfg)
	for i, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("step %d (%s) failed in dry-run: %v", i, step.Name, err)
		}
	}
}

func TestInstallDryRun_CloudflareDNS(t *testing.T) {
	cfg := &Config{
		MCProvider:      ProviderHetzner,
		HetznerToken:    "test-token-1234567890",
		ServerType:      "cx22",
		Region:          "fsn1",
		Domain:          "example.com",
		DNSProvider:     DNSCloudflare,
		CloudflareToken: "cf_test_token",
		DryRun:          true,
	}

	steps := GetInstallSteps(cfg)
	for i, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("step %d (%s) failed in dry-run: %v", i, step.Name, err)
		}
	}
}

func TestInstallDryRun_BuildResult(t *testing.T) {
	cfg := &Config{
		MCProvider: ProviderHetzner,
		Domain:     "dryrun.example.com",
		SSHHost:    "1.2.3.4",
		DryRun:     true,
	}

	result := BuildResult(cfg)
	if result.ServerIP != "1.2.3.4" {
		t.Errorf("expected ServerIP '1.2.3.4', got %q", result.ServerIP)
	}
	if result.MissionControlURL != "https://mission.dryrun.example.com" {
		t.Errorf("expected MissionControlURL mismatch, got %q", result.MissionControlURL)
	}
}
