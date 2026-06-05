package upgrade

import (
	"strings"
	"testing"

	"github.com/dotechhq/zenith/cli/internal/installstate"
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

func TestParseDiskSpaceFreeGB(t *testing.T) {
	tests := []struct {
		input  string
		wantGB float64
		wantOK bool
	}{
		{"10G", 10.0, true},
		{"5.5G", 5.5, true},
		{"512M", 0.5, true},
		{"2048M", 2.0, true},
		{"100K", 0.0001, true},
		{"", 0, false},
		{"garbage", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gb, ok := parseDiskSpaceFreeGB(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseDiskSpaceFreeGB(%q) ok=%v, want %v", tt.input, ok, tt.wantOK)
				return
			}
			if ok && (gb < tt.wantGB*0.99 || gb > tt.wantGB*1.01) {
				t.Errorf("parseDiskSpaceFreeGB(%q) = %.4f, want %.4f", tt.input, gb, tt.wantGB)
			}
		})
	}
}

func TestBuildSteps_SkipBackupRemovesOneStep(t *testing.T) {
	stepsWithBackup := len(buildStepsForTest(false))
	stepsNoBackup := len(buildStepsForTest(true))
	if stepsNoBackup != stepsWithBackup-1 {
		t.Errorf("--skip-backup should remove exactly 1 step: got %d vs %d", stepsNoBackup, stepsWithBackup)
	}
}

func buildStepsForTest(skipBackup bool) []stepFunc {
	return buildSteps(nil, &installstate.State{Domain: "test.example.com"}, "", skipBackup)
}

func TestBuildHelmDiffCmd(t *testing.T) {
	cmd := buildHelmDiffCmd("1.2.3")
	if !strings.Contains(cmd, "helm diff upgrade") {
		t.Errorf("Expected 'helm diff upgrade' in cmd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "zenith") {
		t.Errorf("Expected release name 'zenith' in cmd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "oci://ghcr.io/dotechhq/zenith/charts/zenith") {
		t.Errorf("Expected OCI chart ref in diff cmd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "1.2.3") {
		t.Errorf("Expected version 1.2.3 in diff cmd, got: %s", cmd)
	}
}

func TestBuildHelmDiffCmd_NoVersion(t *testing.T) {
	cmd := buildHelmDiffCmd("")
	if strings.Contains(cmd, "--version") {
		t.Errorf("Expected no --version flag when version is empty, got: %s", cmd)
	}
}
