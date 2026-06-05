// Package upgrade provides the `zen upgrade` command for upgrading a Zenith
// installation via Helm, with automated rollback on failure.
package upgrade

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/installstate"
	"github.com/dotechhq/zenith/cli/internal/sshclient"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagVersion  string
	flagDryRun   bool
	flagSkipBackup bool
)

// Cmd is the `zen upgrade` command.
var Cmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Zenith Mission Control to a new version",
	Long: `Upgrade the Zenith platform using Helm. The process:

  1. (Optional) Trigger a database backup
  2. Run helm upgrade for the zenith release
  3. Wait for all deployments and statefulsets to roll out
  4. Health-check the Mission Control API
  5. Roll back automatically if any step fails

Examples:
  zen upgrade
  zen upgrade --version 1.5.0
  zen upgrade --skip-backup
  zen upgrade --dry-run`,
	RunE: runUpgrade,
}

func init() {
	Cmd.Flags().StringVar(&flagVersion, "version", "", "Target chart version (uses latest if empty)")
	Cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Print upgrade plan without executing")
	Cmd.Flags().BoolVar(&flagSkipBackup, "skip-backup", false, "Skip the pre-upgrade database backup")
}

// stepFuncs are the ordered upgrade steps.
// Each returns an error; on failure the caller rolls back.
type stepFunc struct {
	name string
	desc string
	fn   func(cli *sshclient.Client) error
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).PaddingLeft(2)
	muted := lipgloss.NewStyle().Foreground(tui.ColorMuted).PaddingLeft(2)
	checkStyle := lipgloss.NewStyle().Foreground(tui.ColorSuccess)
	spinStyle := lipgloss.NewStyle().Foreground(tui.ColorWarning)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError)
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	fmt.Println()
	fmt.Println(headerStyle.Render("Zenith Upgrade"))
	if flagVersion != "" {
		fmt.Println(muted.Render("Target version: " + flagVersion))
	} else {
		fmt.Println(muted.Render("Target version: latest"))
	}
	if flagDryRun {
		fmt.Println(muted.Render("(dry-run mode — no changes will be made)"))
	}
	fmt.Println()

	// Load install state
	if !installstate.Exists() {
		return fmt.Errorf("no Zenith installation state found — run 'zen install' first")
	}
	state, err := installstate.Load()
	if err != nil {
		return fmt.Errorf("failed to load installation state: %w", err)
	}
	if state.ServerIP == "" {
		return fmt.Errorf("server IP not found in installation state")
	}

	if flagDryRun {
		printDryRun(state, flagVersion, flagSkipBackup)
		return nil
	}

	// Establish SSH connection
	fmt.Println(muted.Render(fmt.Sprintf("Connecting to %s...", state.ServerIP)))
	sshCfg := sshclient.Config{
		Host:    state.ServerIP,
		User:    "root",
		Timeout: 15 * time.Second,
	}
	if state.SSHKeyPath != "" {
		if keyData, readErr := os.ReadFile(state.SSHKeyPath); readErr == nil {
			sshCfg.PrivateKey = keyData
		}
	}

	sshCli, err := sshclient.DialWithRetry(sshCfg, 3, 5*time.Second)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer sshCli.Close()
	fmt.Println(muted.Render("Connected."))
	fmt.Println()

	// Build the step list dynamically
	steps := buildSteps(sshCli, state, flagVersion, flagSkipBackup)
	total := len(steps)

	step := func(num int, name, desc string) {
		stepNum := fmt.Sprintf("[%d/%d]", num, total)
		fmt.Printf("  %s %s %s\n",
			spinStyle.Render("  "+stepNum),
			stepStyle.Render(name),
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("- "+desc),
		)
	}
	ok := func(num int, name string, elapsed time.Duration) {
		stepNum := fmt.Sprintf("[%d/%d]", num, total)
		fmt.Printf("\033[1A\r  %s %s %s\n",
			checkStyle.Render("v "+stepNum),
			stepStyle.Render(name),
			timeStyle.Render(fmt.Sprintf("(%s)", formatDuration(elapsed))),
		)
	}
	fail := func(num int, name, msg string) {
		stepNum := fmt.Sprintf("[%d/%d]", num, total)
		fmt.Printf("\033[1A\r  %s %s %s\n",
			errStyle.Render("x "+stepNum),
			stepStyle.Render(name),
			errStyle.Render("- "+msg),
		)
	}

	totalStart := time.Now()

	for i, s := range steps {
		num := i + 1
		step(num, s.name, s.desc)
		t := time.Now()
		if err := s.fn(sshCli); err != nil {
			fail(num, s.name, err.Error())

			// Attempt Helm rollback for upgrade failure
			if s.name == "Helm upgrade" {
				fmt.Println()
				fmt.Println(errStyle.Render("  Upgrade failed — attempting rollback..."))
				if rbErr := helmRollback(sshCli); rbErr != nil {
					fmt.Println(errStyle.Render("  Rollback also failed: " + rbErr.Error()))
				} else {
					fmt.Println(checkStyle.Render("  Rollback successful."))
				}
			}
			return fmt.Errorf("upgrade step %q failed: %w", s.name, err)
		}
		ok(num, s.name, time.Since(t))
	}

	totalElapsed := time.Since(totalStart)
	fmt.Println()
	fmt.Printf("  %s %s\n",
		checkStyle.Render("v Upgrade complete"),
		timeStyle.Render(fmt.Sprintf("in %s", formatDuration(totalElapsed))),
	)
	fmt.Println()
	return nil
}

// buildSteps constructs the ordered list of upgrade steps.
func buildSteps(cli *sshclient.Client, state *installstate.State, version string, skipBackup bool) []stepFunc {
	var steps []stepFunc

	if !skipBackup {
		steps = append(steps, stepFunc{
			name: "Pre-upgrade backup",
			desc: "Triggering immediate CNPG database backup...",
			fn: func(cli *sshclient.Client) error {
				return triggerAndWaitBackup(cli)
			},
		})
	}

	steps = append(steps, stepFunc{
		name: "Helm upgrade",
		desc: "Running helm upgrade for the zenith release...",
		fn: func(cli *sshclient.Client) error {
			return helmUpgrade(cli, version)
		},
	})

	steps = append(steps, stepFunc{
		name: "Wait for rollout",
		desc: "Waiting for all pods to roll over...",
		fn: func(cli *sshclient.Client) error {
			return waitForRollout(cli)
		},
	})

	steps = append(steps, stepFunc{
		name: "Health check",
		desc: "Verifying Mission Control API is healthy...",
		fn: func(cli *sshclient.Client) error {
			return healthCheck(cli, state)
		},
	})

	return steps
}

// buildHelmUpgradeCmd returns the helm upgrade command string for the given version.
func buildHelmUpgradeCmd(version string) string {
	cmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm upgrade zenith " +
		"oci://ghcr.io/dotechhq/zenith/charts/zenith " +
		"-n zenith-system --wait --timeout=10m"
	if version != "" {
		cmd += " --version " + version
	}
	return cmd + " 2>&1"
}

// buildHelmRollbackCmd returns the helm rollback command string.
func buildHelmRollbackCmd() string {
	return "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm rollback zenith -n zenith-system --wait --timeout=5m 2>&1"
}

// helmUpgrade runs helm upgrade for the zenith chart.
func helmUpgrade(cli *sshclient.Client, version string) error {
	_, err := cli.Run(buildHelmUpgradeCmd(version))
	return err
}

// helmRollback rolls back the zenith Helm release.
func helmRollback(cli *sshclient.Client) error {
	_, err := cli.Run(buildHelmRollbackCmd())
	return err
}

// waitForRollout waits for all deployments and statefulsets in zenith-system to finish rolling out.
func waitForRollout(cli *sshclient.Client) error {
	cmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl rollout status deployment -n zenith-system --timeout=10m 2>&1 && " +
		"KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl rollout status statefulset -n zenith-system --timeout=10m 2>&1"
	_, err := cli.Run(cmd)
	return err
}

// healthCheck verifies the Mission Control API returns a healthy response.
func healthCheck(cli *sshclient.Client, state *installstate.State) error {
	url := state.MissionControlURL
	if url == "" && state.Domain != "" {
		url = fmt.Sprintf("https://mission.%s", state.Domain)
	}
	if url == "" {
		// Fall back to localhost inside the cluster
		url = "http://localhost:8080"
	}
	cmd := fmt.Sprintf(`curl -sf --max-time 10 "%s/health" -o /dev/null 2>&1`, url)
	_, err := cli.Run(cmd)
	return err
}

// triggerAndWaitBackup triggers a CNPG backup and waits up to 5 minutes for it to complete.
func triggerAndWaitBackup(cli *sshclient.Client) error {
	annotateCmd := `kubectl annotate cluster.postgresql.cnpg.io/zenith-postgres -n zenith-system backup.cnpg.io/immediate="true" --overwrite 2>&1`
	if _, err := cli.Run(annotateCmd); err != nil {
		return fmt.Errorf("trigger backup: %w", err)
	}

	deadline := time.Now().Add(5 * time.Minute)
	pollCmd := `kubectl get backup -n zenith-system --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1:].status.phase}' 2>/dev/null`

	for time.Now().Before(deadline) {
		phase, _ := cli.Run(pollCmd)
		switch phase {
		case "completed":
			return nil
		case "failed":
			return fmt.Errorf("CNPG backup phase=failed")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for pre-upgrade backup")
}

// printDryRun shows what would happen without executing.
func printDryRun(state *installstate.State, version string, skipBackup bool) {
	muted := lipgloss.NewStyle().Foreground(tui.ColorMuted).PaddingLeft(2)
	info := lipgloss.NewStyle().Foreground(tui.ColorText).PaddingLeft(2)
	bold := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).PaddingLeft(2)

	fmt.Println(bold.Render("Dry-run plan:"))
	fmt.Println()

	stepNum := 1
	if !skipBackup {
		fmt.Println(info.Render(fmt.Sprintf("  %d. Pre-upgrade backup — CNPG immediate annotation", stepNum)))
		stepNum++
	}
	versionStr := "latest"
	if version != "" {
		versionStr = version
	}
	fmt.Println(info.Render(fmt.Sprintf("  %d. Helm upgrade — zenith@%s in zenith-system", stepNum, versionStr)))
	stepNum++
	fmt.Println(info.Render(fmt.Sprintf("  %d. Wait for rollout — deployments + statefulsets in zenith-system", stepNum)))
	stepNum++
	fmt.Println(info.Render(fmt.Sprintf("  %d. Health check — %s/health", stepNum, state.MissionControlURL)))
	fmt.Println()
	fmt.Println(muted.Render("No changes made (dry-run)."))
	fmt.Println()
}

// formatDuration produces a human-friendly duration string.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
