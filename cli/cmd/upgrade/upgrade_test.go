package upgrade

import (
	"strings"
	"testing"
)

func TestHelmUpgradeCommand_ReleaseName(t *testing.T) {
	cmd := buildHelmUpgradeCmd("1.2.3")
	if !strings.Contains(cmd, "zenith ") {
		t.Errorf("Expected release name 'zenith' in command, got: %s", cmd)
	}
	if strings.Contains(cmd, "zenith-platform") {
		t.Errorf("Expected 'zenith-platform' to be gone, got: %s", cmd)
	}
	if !strings.Contains(cmd, "oci://ghcr.io/dotechhq/zenith/charts/zenith") {
		t.Errorf("Expected OCI chart ref in command, got: %s", cmd)
	}
}

func TestHelmUpgradeCommand_NoVersion(t *testing.T) {
	cmd := buildHelmUpgradeCmd("")
	if strings.Contains(cmd, "--version") {
		t.Errorf("Expected no --version flag when version is empty, got: %s", cmd)
	}
}

func TestHelmUpgradeCommand_WithVersion(t *testing.T) {
	cmd := buildHelmUpgradeCmd("2.0.0")
	if !strings.Contains(cmd, "--version 2.0.0") {
		t.Errorf("Expected '--version 2.0.0' in command, got: %s", cmd)
	}
}

func TestHelmRollbackCommand_ReleaseName(t *testing.T) {
	cmd := buildHelmRollbackCmd()
	if !strings.Contains(cmd, "zenith ") && !strings.HasSuffix(strings.TrimSuffix(cmd, " 2>&1"), "zenith") {
		t.Errorf("Expected release name 'zenith' in rollback command, got: %s", cmd)
	}
	if strings.Contains(cmd, "zenith-platform") {
		t.Errorf("Expected 'zenith-platform' to be gone from rollback, got: %s", cmd)
	}
}
