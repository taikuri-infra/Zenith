package k3s

import (
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/cli/internal/sshclient"
)

const installScriptURL = "https://get.k3s.io"

// DefaultK3sVersion is the pinned k3s version used when no version is specified.
// Exported so installer.go can reference it in log output and documentation.
const DefaultK3sVersion = "v1.34.3+k3s1"

// Options controls k3s installation behaviour.
type Options struct {
	// Version to install, e.g. "v1.29.4+k3s1". Empty = latest stable.
	Version string
	// ExtraArgs are additional KEY=VALUE env vars passed to the installer.
	ExtraArgs []string
	// DisableComponents lists bundled k3s components to skip (e.g. "traefik").
	DisableComponents []string
}

// Install downloads and runs the k3s installer on the remote host.
// If opts.Version is empty, DefaultK3sVersion is used to prevent supply chain risk
// from pulling an unpinned "latest" release.
func Install(c *sshclient.Client, opts Options) error {
	if opts.Version == "" {
		opts.Version = DefaultK3sVersion
	}
	env := buildEnv(opts)
	var cmd string
	if env != "" {
		cmd = fmt.Sprintf("curl -sfL %s | %s sh -s - --write-kubeconfig-mode 600", installScriptURL, env)
	} else {
		cmd = fmt.Sprintf("curl -sfL %s | sh -s - --write-kubeconfig-mode 600", installScriptURL)
	}
	out, err := c.Run(cmd)
	if err != nil {
		return fmt.Errorf("k3s install: %w\nOutput: %s", err, out)
	}
	return nil
}

// GetKubeconfig retrieves /etc/rancher/k3s/k3s.yaml from the remote host.
func GetKubeconfig(c *sshclient.Client) (string, error) {
	out, err := c.Run("cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", fmt.Errorf("get kubeconfig: %w", err)
	}
	return out, nil
}

// WaitForReady polls until k3s is ready (all nodes show Ready) or timeout.
func WaitForReady(c *sshclient.Client, timeoutSeconds int) error {
	cmd := fmt.Sprintf(
		"timeout %d sh -c 'until k3s kubectl get nodes 2>/dev/null | grep -q \" Ready\"; do sleep 3; done'",
		timeoutSeconds,
	)
	out, err := c.Run(cmd)
	if err != nil {
		return fmt.Errorf("k3s not ready after %ds: %w\nOutput: %s", timeoutSeconds, err, out)
	}
	return nil
}

// GetNodeStatus returns the output of 'k3s kubectl get nodes'.
func GetNodeStatus(c *sshclient.Client) (string, error) {
	return c.Run("k3s kubectl get nodes")
}

func buildEnv(opts Options) string {
	vars := map[string]string{}
	if opts.Version != "" {
		vars["INSTALL_K3S_VERSION"] = opts.Version
	}
	if len(opts.DisableComponents) > 0 {
		vars["INSTALL_K3S_EXEC"] = "--disable " + strings.Join(opts.DisableComponents, " --disable ")
	}
	for _, a := range opts.ExtraArgs {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}
	if len(vars) == 0 {
		return ""
	}
	parts := make([]string, 0, len(vars))
	for k, v := range vars {
		parts = append(parts, fmt.Sprintf("%s=%q", k, v))
	}
	return strings.Join(parts, " ")
}
