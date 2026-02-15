package install

import "testing"

func TestValidateToken(t *testing.T) {
	if err := ValidateToken("valid-hetzner-token-1234567890"); err != nil {
		t.Errorf("Expected valid token, got error: %v", err)
	}

	if err := ValidateToken("short"); err == nil {
		t.Error("Expected error for short token")
	}

	if err := ValidateToken(""); err == nil {
		t.Error("Expected error for empty token")
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

	for _, opt := range opts {
		if opt == "" {
			t.Error("Region option should not be empty")
		}
	}
}

func TestServerTypeOptions(t *testing.T) {
	opts := ServerTypeOptions()
	if len(opts) != len(ServerTypes) {
		t.Errorf("Expected %d server type options, got %d", len(ServerTypes), len(opts))
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
		{"unknown", "fsn1"},
	}

	for _, tt := range tests {
		result := ParseRegionSelection(tt.input)
		if result != tt.expected {
			t.Errorf("ParseRegionSelection(%q) = %q, want %q", tt.input, result, tt.expected)
		}
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
	}

	for _, tt := range tests {
		result := ParseServerTypeSelection(tt.input)
		if result != tt.expected {
			t.Errorf("ParseServerTypeSelection(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
