package install

import (
	"fmt"
	"os/exec"
)

// HelmConfig contains configuration for Helm-based platform installation.
type HelmConfig struct {
	ReleaseName string
	Namespace   string
	ChartPath   string
	Domain      string
	HetznerToken string
	ValuesFile  string
}

// DefaultHelmConfig returns sensible defaults for Helm installation.
func DefaultHelmConfig(cfg *Config) *HelmConfig {
	return &HelmConfig{
		ReleaseName:  "zenith",
		Namespace:    "zenith-system",
		ChartPath:    "oci://ghcr.io/dotechhq/zenith/charts/zenith",
		HetznerToken: cfg.HetznerToken,
	}
}

// InstallViaHelm installs the Zenith platform using Helm.
func InstallViaHelm(hcfg *HelmConfig) error {
	args := []string{
		"upgrade", "--install",
		hcfg.ReleaseName,
		hcfg.ChartPath,
		"--namespace", hcfg.Namespace,
		"--create-namespace",
		"--set", fmt.Sprintf("global.hetznerToken=%s", hcfg.HetznerToken),
	}

	if hcfg.Domain != "" {
		args = append(args, "--set", fmt.Sprintf("global.domain=%s", hcfg.Domain))
	}

	if hcfg.ValuesFile != "" {
		args = append(args, "-f", hcfg.ValuesFile)
	}

	args = append(args, "--wait", "--timeout", "10m")

	cmd := exec.Command("helm", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

// VerifyInstallation checks that all Zenith pods are running.
func VerifyInstallation(namespace string) error {
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace,
		"--field-selector=status.phase!=Running",
		"-o", "name")

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kubectl check failed: %w", err)
	}

	if len(output) > 0 {
		return fmt.Errorf("some pods are not running in namespace %s", namespace)
	}

	return nil
}

// GetHelmArgs returns the Helm install arguments for display/debugging.
func GetHelmArgs(hcfg *HelmConfig) []string {
	args := []string{
		"helm", "upgrade", "--install",
		hcfg.ReleaseName,
		hcfg.ChartPath,
		"--namespace", hcfg.Namespace,
		"--create-namespace",
	}

	if hcfg.Domain != "" {
		args = append(args, "--set", fmt.Sprintf("global.domain=%s", hcfg.Domain))
	}

	return args
}
