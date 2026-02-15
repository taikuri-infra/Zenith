package install

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"valid long token", "valid-hetzner-token-1234567890", false},
		{"valid exact 10 chars", "1234567890", false},
		{"short token", "short", true},
		{"empty token", "", true},
		{"9 chars", "123456789", true},
		{"11 chars", "12345678901", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken(%q) error = %v, wantErr %v", tt.token, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "too short") {
				t.Errorf("Expected error to mention 'too short', got: %v", err)
			}
		})
	}
}

func TestGetInstallSteps(t *testing.T) {
	cfg := &Config{
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	steps := GetInstallSteps(cfg)
	if len(steps) != 7 {
		t.Errorf("Expected 7 install steps, got %d", len(steps))
	}

	// Verify steps execute without error
	for i, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("Step %d (%s) failed: %v", i, step.Name, err)
		}
	}
}

func TestGetInstallSteps_StepNames(t *testing.T) {
	cfg := &Config{
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	steps := GetInstallSteps(cfg)

	expectedNames := []string{
		"Validate Hetzner token",
		"Create management server",
		"Install k3s",
		"Install Cluster API",
		"Deploy Zenith platform",
		"Configure DNS & SSL",
		"Create admin account",
	}

	for i, expected := range expectedNames {
		if i >= len(steps) {
			t.Fatalf("Expected at least %d steps", i+1)
		}
		if steps[i].Name != expected {
			t.Errorf("Step %d: expected name '%s', got '%s'", i, expected, steps[i].Name)
		}
	}
}

func TestGetInstallSteps_StepDurations(t *testing.T) {
	cfg := &Config{
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	steps := GetInstallSteps(cfg)

	for _, step := range steps {
		if step.Duration <= 0 {
			t.Errorf("Step '%s' should have a positive duration", step.Name)
		}
	}
}

func TestGetInstallSteps_StepDescriptions(t *testing.T) {
	cfg := &Config{
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	steps := GetInstallSteps(cfg)

	for _, step := range steps {
		if step.Description == "" {
			t.Errorf("Step '%s' should have a description", step.Name)
		}
	}
}

func TestGetInstallSteps_DescriptionContainsConfig(t *testing.T) {
	cfg := &Config{
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	steps := GetInstallSteps(cfg)

	// The second step ("Create management server") should mention the server type and region
	if len(steps) >= 2 {
		desc := steps[1].Description
		if !strings.Contains(desc, cfg.ServerType) {
			t.Errorf("Expected step 2 description to contain server type '%s', got: '%s'", cfg.ServerType, desc)
		}
		if !strings.Contains(desc, cfg.Region) {
			t.Errorf("Expected step 2 description to contain region '%s', got: '%s'", cfg.Region, desc)
		}
	}
}

func TestGetInstallSteps_InvalidToken(t *testing.T) {
	cfg := &Config{
		HetznerToken: "short",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	steps := GetInstallSteps(cfg)
	// First step validates token
	if err := steps[0].Action(cfg); err == nil {
		t.Error("Expected first step to fail with invalid token")
	}
}

func TestRegionOptions(t *testing.T) {
	opts := RegionOptions()
	if len(opts) != len(Regions) {
		t.Errorf("Expected %d region options, got %d", len(Regions), len(opts))
	}

	for i, opt := range opts {
		if opt == "" {
			t.Error("Region option should not be empty")
		}
		// Each option should contain the region ID
		if !strings.Contains(opt, Regions[i].ID) {
			t.Errorf("Expected option to contain region ID '%s', got '%s'", Regions[i].ID, opt)
		}
		// Each option should contain the country
		if !strings.Contains(opt, Regions[i].Country) {
			t.Errorf("Expected option to contain country '%s', got '%s'", Regions[i].Country, opt)
		}
	}
}

func TestServerTypeOptions(t *testing.T) {
	opts := ServerTypeOptions()
	if len(opts) != len(ServerTypes) {
		t.Errorf("Expected %d server type options, got %d", len(ServerTypes), len(opts))
	}

	for i, opt := range opts {
		if opt == "" {
			t.Error("Server type option should not be empty")
		}
		// Each option should contain the server type ID
		if !strings.Contains(opt, ServerTypes[i].ID) {
			t.Errorf("Expected option to contain server type ID '%s', got '%s'", ServerTypes[i].ID, opt)
		}
		// Each option should contain the price
		priceStr := fmt.Sprintf("%.2f", ServerTypes[i].Price)
		if !strings.Contains(opt, priceStr) {
			t.Errorf("Expected option to contain price '%s', got '%s'", priceStr, opt)
		}
	}
}

func TestParseRegionSelection(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fsn1 - Falkenstein, Germany", "fsn1"},
		{"nbg1 - Nuremberg, Germany", "nbg1"},
		{"hel1 - Helsinki, Finland", "hel1"},
		{"ash - Ashburn, USA", "ash"},
		{"hil - Hillsboro, USA", "hil"},
		{"unknown", "fsn1"},
		{"", "fsn1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseRegionSelection(tt.input)
			if result != tt.expected {
				t.Errorf("ParseRegionSelection(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseServerTypeSelection(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cx22 - 2 vCPU", "cx22"},
		{"cx32 - 4 vCPU", "cx32"},
		{"cx42 - 8 vCPU", "cx42"},
		{"unknown", "cx22"},
		{"", "cx22"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseServerTypeSelection(tt.input)
			if result != tt.expected {
				t.Errorf("ParseServerTypeSelection(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRegions_Data(t *testing.T) {
	if len(Regions) == 0 {
		t.Fatal("Expected at least one region")
	}

	regionIDs := make(map[string]bool)
	for _, r := range Regions {
		if r.ID == "" {
			t.Error("Region ID should not be empty")
		}
		if r.Name == "" {
			t.Error("Region Name should not be empty")
		}
		if r.Country == "" {
			t.Error("Region Country should not be empty")
		}
		if regionIDs[r.ID] {
			t.Errorf("Duplicate region ID: %s", r.ID)
		}
		regionIDs[r.ID] = true
	}
}

func TestServerTypes_Data(t *testing.T) {
	if len(ServerTypes) == 0 {
		t.Fatal("Expected at least one server type")
	}

	serverTypeIDs := make(map[string]bool)
	for _, s := range ServerTypes {
		if s.ID == "" {
			t.Error("ServerType ID should not be empty")
		}
		if s.Name == "" {
			t.Error("ServerType Name should not be empty")
		}
		if s.CPUs <= 0 {
			t.Errorf("ServerType %s should have positive CPUs", s.ID)
		}
		if s.RAM <= 0 {
			t.Errorf("ServerType %s should have positive RAM", s.ID)
		}
		if s.Price <= 0 {
			t.Errorf("ServerType %s should have positive Price", s.ID)
		}
		if s.Description == "" {
			t.Errorf("ServerType %s should have a description", s.ID)
		}
		if serverTypeIDs[s.ID] {
			t.Errorf("Duplicate server type ID: %s", s.ID)
		}
		serverTypeIDs[s.ID] = true
	}
}

func TestRegionValidation(t *testing.T) {
	validIDs := make(map[string]bool)
	for _, r := range Regions {
		validIDs[r.ID] = true
	}

	// Test known valid regions
	validRegions := []string{"fsn1", "nbg1", "hel1", "ash", "hil"}
	for _, id := range validRegions {
		if !validIDs[id] {
			t.Errorf("Expected region '%s' to be valid", id)
		}
	}

	// Test unknown region
	if validIDs["invalid-region"] {
		t.Error("Expected 'invalid-region' to be invalid")
	}
}

func TestServerTypeValidation(t *testing.T) {
	validIDs := make(map[string]bool)
	for _, s := range ServerTypes {
		validIDs[s.ID] = true
	}

	validTypes := []string{"cx22", "cx32", "cx42"}
	for _, id := range validTypes {
		if !validIDs[id] {
			t.Errorf("Expected server type '%s' to be valid", id)
		}
	}

	if validIDs["cx11"] {
		t.Error("Expected 'cx11' to be invalid")
	}
}

func TestDefaultHelmConfig(t *testing.T) {
	cfg := &Config{
		HetznerToken: "test-token-1234567890",
		ServerType:   "cx22",
		Region:       "fsn1",
	}

	hcfg := DefaultHelmConfig(cfg)

	if hcfg.ReleaseName != "zenith" {
		t.Errorf("Expected ReleaseName 'zenith', got '%s'", hcfg.ReleaseName)
	}
	if hcfg.Namespace != "zenith-system" {
		t.Errorf("Expected Namespace 'zenith-system', got '%s'", hcfg.Namespace)
	}
	if hcfg.ChartPath == "" {
		t.Error("Expected non-empty ChartPath")
	}
	if hcfg.HetznerToken != cfg.HetznerToken {
		t.Errorf("Expected HetznerToken '%s', got '%s'", cfg.HetznerToken, hcfg.HetznerToken)
	}
	if hcfg.Domain != "" {
		t.Errorf("Expected empty Domain by default, got '%s'", hcfg.Domain)
	}
	if hcfg.ValuesFile != "" {
		t.Errorf("Expected empty ValuesFile by default, got '%s'", hcfg.ValuesFile)
	}
}

func TestGetHelmArgs_Basic(t *testing.T) {
	hcfg := &HelmConfig{
		ReleaseName:  "zenith",
		Namespace:    "zenith-system",
		ChartPath:    "oci://ghcr.io/dotechhq/zenith/charts/zenith",
		HetznerToken: "test-token",
	}

	args := GetHelmArgs(hcfg)

	// Should contain basic args
	expectedContains := []string{
		"helm",
		"upgrade",
		"--install",
		"zenith",
		"oci://ghcr.io/dotechhq/zenith/charts/zenith",
		"--namespace",
		"zenith-system",
		"--create-namespace",
	}

	for _, expected := range expectedContains {
		found := false
		for _, arg := range args {
			if arg == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected args to contain '%s', got: %v", expected, args)
		}
	}
}

func TestGetHelmArgs_WithDomain(t *testing.T) {
	hcfg := &HelmConfig{
		ReleaseName:  "zenith",
		Namespace:    "zenith-system",
		ChartPath:    "oci://ghcr.io/dotechhq/zenith/charts/zenith",
		HetznerToken: "test-token",
		Domain:       "example.com",
	}

	args := GetHelmArgs(hcfg)

	// Should contain --set global.domain=example.com
	foundDomainSet := false
	for i, arg := range args {
		if arg == "--set" && i+1 < len(args) && strings.Contains(args[i+1], "global.domain=example.com") {
			foundDomainSet = true
			break
		}
	}

	if !foundDomainSet {
		t.Errorf("Expected args to contain '--set global.domain=example.com', got: %v", args)
	}
}

func TestGetHelmArgs_WithoutDomain(t *testing.T) {
	hcfg := &HelmConfig{
		ReleaseName:  "zenith",
		Namespace:    "zenith-system",
		ChartPath:    "oci://ghcr.io/dotechhq/zenith/charts/zenith",
		HetznerToken: "test-token",
		Domain:       "",
	}

	args := GetHelmArgs(hcfg)

	// Should NOT contain domain set
	for _, arg := range args {
		if strings.Contains(arg, "global.domain") {
			t.Errorf("Expected args to NOT contain 'global.domain' when domain is empty, got: %v", args)
		}
	}
}

func TestGetHelmArgs_WithValuesFile(t *testing.T) {
	// Note: GetHelmArgs does not currently include ValuesFile, but we test it does not crash
	hcfg := &HelmConfig{
		ReleaseName:  "zenith",
		Namespace:    "zenith-system",
		ChartPath:    "oci://ghcr.io/dotechhq/zenith/charts/zenith",
		HetznerToken: "test-token",
		ValuesFile:   "/path/to/values.yaml",
	}

	args := GetHelmArgs(hcfg)
	if len(args) == 0 {
		t.Error("Expected non-empty args")
	}
}

func TestConfig_StructFields(t *testing.T) {
	cfg := Config{
		HetznerToken: "hc_token123",
		ServerType:   "cx22",
		Region:       "fsn1",
		SSHKeyPath:   "/home/user/.ssh/id_rsa",
	}

	if cfg.HetznerToken != "hc_token123" {
		t.Errorf("Expected HetznerToken 'hc_token123', got '%s'", cfg.HetznerToken)
	}
	if cfg.ServerType != "cx22" {
		t.Errorf("Expected ServerType 'cx22', got '%s'", cfg.ServerType)
	}
	if cfg.Region != "fsn1" {
		t.Errorf("Expected Region 'fsn1', got '%s'", cfg.Region)
	}
	if cfg.SSHKeyPath != "/home/user/.ssh/id_rsa" {
		t.Errorf("Expected SSHKeyPath '/home/user/.ssh/id_rsa', got '%s'", cfg.SSHKeyPath)
	}
}

func TestStep_StructFields(t *testing.T) {
	step := Step{
		Name:        "Test step",
		Description: "This is a test",
		Action:      func(cfg *Config) error { return nil },
	}

	if step.Name != "Test step" {
		t.Errorf("Expected Name 'Test step', got '%s'", step.Name)
	}
	if step.Description != "This is a test" {
		t.Errorf("Expected Description 'This is a test', got '%s'", step.Description)
	}
	if step.Action == nil {
		t.Error("Expected non-nil Action")
	}

	// Execute the action
	if err := step.Action(&Config{}); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestParseRegionSelection_FromRegionOptions(t *testing.T) {
	// Test that ParseRegionSelection works with the output from RegionOptions
	opts := RegionOptions()
	for i, opt := range opts {
		result := ParseRegionSelection(opt)
		if result != Regions[i].ID {
			t.Errorf("ParseRegionSelection(%q) = %q, want %q", opt, result, Regions[i].ID)
		}
	}
}

func TestParseServerTypeSelection_FromServerTypeOptions(t *testing.T) {
	opts := ServerTypeOptions()
	for i, opt := range opts {
		result := ParseServerTypeSelection(opt)
		if result != ServerTypes[i].ID {
			t.Errorf("ParseServerTypeSelection(%q) = %q, want %q", opt, result, ServerTypes[i].ID)
		}
	}
}
