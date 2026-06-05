package upgrade

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/healthcheck"
	"github.com/dotechhq/zenith/cli/internal/installstate"
	"github.com/dotechhq/zenith/cli/internal/sshclient"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagVersion string
	flagDryRun  bool
)

var Cmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Zenith to a newer version",
	Long: `Upgrade an existing Zenith installation to a newer chart version.

  zen upgrade                    # upgrade to latest
  zen upgrade --version v1.2.0   # upgrade to specific version
  zen upgrade --dry-run          # show what would change (requires helm-diff)`,
	RunE: runUpgrade,
}

func init() {
	f := Cmd.Flags()
	f.StringVar(&flagVersion, "version", "", "Chart version to upgrade to (default: latest)")
	f.BoolVar(&flagDryRun, "dry-run", false, "Show diff without applying changes (requires helm-diff plugin)")
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	checkStyle := lipgloss.NewStyle().Foreground(tui.ColorSuccess)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError)
	warnStyle := lipgloss.NewStyle().Foreground(tui.ColorWarning)
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	descStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).PaddingLeft(2).Render("Zenith Upgrade"))
	fmt.Println()

	// Load install state
	state, err := loadState()
	if err != nil {
		return fmt.Errorf("could not find install state: %w\n\nRun 'zen install' first", err)
	}

	domain := state.Domain
	serverIP := state.ServerIP

	step := func(num, total int, name, desc string) {
		fmt.Printf("  %s %s %s\n",
			warnStyle.Render(fmt.Sprintf("[%d/%d]", num, total)),
			stepStyle.Render(name),
			descStyle.Render("- "+desc),
		)
	}
	ok := func(num, total int, name string, elapsed time.Duration) {
		fmt.Printf("\033[1A\r  %s %s %s\n",
			checkStyle.Render(fmt.Sprintf("v [%d/%d]", num, total)),
			stepStyle.Render(name),
			descStyle.Render(fmt.Sprintf("(%s)", formatDuration(elapsed))),
		)
	}
	fail := func(num, total int, name, msg string) {
		fmt.Printf("\033[1A\r  %s %s %s\n",
			errStyle.Render(fmt.Sprintf("x [%d/%d]", num, total)),
			stepStyle.Render(name),
			errStyle.Render("- "+msg),
		)
	}

	total := 5
	if flagDryRun {
		total = 3
	}

	// Step 1: Pre-flight
	step(1, total, "Pre-flight check", "verifying cluster reachability and current version")
	t := time.Now()
	currentVersion, sshCli, err := preflight(serverIP, state.SSHKeyPath)
	if err != nil {
		fail(1, total, "Pre-flight check", err.Error())
		return err
	}
	defer sshCli.Close()
	ok(1, total, "Pre-flight check", time.Since(t))
	fmt.Printf("  %s current version: %s\n", descStyle.Render("  ->"), currentVersion)

	// Resolve target version
	targetVersion := flagVersion
	if targetVersion == "" {
		targetVersion = "latest"
	}

	// Step 2: Compatibility check
	step(2, total, "Compatibility check", fmt.Sprintf("checking %s → %s migration path", currentVersion, targetVersion))
	t = time.Now()
	if err := checkCompatibility(currentVersion, targetVersion); err != nil {
		fail(2, total, "Compatibility check", err.Error())
		return err
	}
	ok(2, total, "Compatibility check", time.Since(t))

	// Step 3: Dry run or real upgrade
	if flagDryRun {
		step(3, total, "Diff", "running helm diff upgrade to show changes")
		t = time.Now()
		diff, err := helmDiff(sshCli, targetVersion)
		if err != nil {
			fail(3, total, "Diff", err.Error())
			return err
		}
		ok(3, total, "Diff", time.Since(t))
		fmt.Println()
		fmt.Println(descStyle.Render("  Changes that would be applied:"))
		fmt.Println()
		for _, line := range strings.Split(diff, "\n") {
			fmt.Println("  " + line)
		}
		return nil
	}

	// Step 3: Upgrade
	step(3, total, "Upgrade", fmt.Sprintf("helm upgrade zenith → %s", targetVersion))
	t = time.Now()
	if err := helmUpgrade(sshCli, targetVersion); err != nil {
		fail(3, total, "Upgrade", err.Error())
		// Rollback
		fmt.Println()
		fmt.Println(warnStyle.Render("  Upgrade failed — initiating rollback..."))
		if rbErr := helmRollback(sshCli); rbErr != nil {
			fmt.Println(errStyle.Render("  Rollback also failed: " + rbErr.Error()))
		} else {
			fmt.Println(checkStyle.Render("  Rollback complete — previous version restored"))
		}
		return err
	}
	ok(3, total, "Upgrade", time.Since(t))

	// Step 4: Health check
	step(4, total, "Health check", "waiting for all components to become healthy")
	t = time.Now()
	healthURL := fmt.Sprintf("https://cloud.%s/api/health", domain)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := healthcheck.WaitUntilHealthy(ctx, healthcheck.Options{URL: healthURL}); err != nil {
		fail(4, total, "Health check", err.Error())
		fmt.Println()
		fmt.Println(warnStyle.Render("  Health check failed — initiating rollback..."))
		if rbErr := helmRollback(sshCli); rbErr != nil {
			fmt.Println(errStyle.Render("  Rollback also failed: " + rbErr.Error()))
		} else {
			fmt.Println(checkStyle.Render("  Rollback complete — previous version restored"))
		}
		return fmt.Errorf("health check failed after upgrade: %w", err)
	}
	ok(4, total, "Health check", time.Since(t))

	// Step 5: Report
	step(5, total, "Complete", "upgrade finished successfully")
	ok(5, total, "Complete", 0)

	fmt.Println()
	fmt.Println(checkStyle.Render("  Zenith upgraded successfully to " + targetVersion))
	fmt.Println()

	return nil
}

func loadState() (*installstate.State, error) {
	return installstate.Load()
}

func preflight(serverIP, sshKeyPath string) (currentVersion string, cli *sshclient.Client, err error) {
	cfg := sshclient.Config{
		Host:    serverIP,
		Port:    22,
		User:    "root",
		Timeout: 15 * time.Second,
	}
	if sshKeyPath != "" {
		key, readErr := os.ReadFile(sshKeyPath)
		if readErr != nil {
			return "", nil, fmt.Errorf("read SSH key %s: %w", sshKeyPath, readErr)
		}
		cfg.PrivateKey = key
	}
	cli, err = sshclient.DialWithRetry(cfg, 3, 10*time.Second)
	if err != nil {
		return "", nil, fmt.Errorf("cannot reach server %s: %w", serverIP, err)
	}

	// Get current helm release version
	out, err := cli.Run("helm list -n zenith-system -o json 2>/dev/null | grep -o '\"chart\":\"[^\"]*\"' | head -1")
	if err != nil {
		out = "unknown"
	}
	currentVersion = strings.TrimSpace(out)
	if currentVersion == "" {
		currentVersion = "unknown"
	}

	// Check disk space (warn if < 2GB free)
	diskOut, _ := cli.Run("df -BG / | tail -1 | awk '{print $4}' | tr -d 'G'")
	diskOut = strings.TrimSpace(diskOut)
	if diskOut != "" && diskOut < "2" {
		return currentVersion, cli, fmt.Errorf("insufficient disk space: %sGB free (need at least 2GB)", diskOut)
	}

	return currentVersion, cli, nil
}

// checkCompatibility blocks upgrades that skip more than one minor version.
func checkCompatibility(current, target string) error {
	// If either is unknown or "latest", allow
	if current == "unknown" || target == "latest" || target == "" {
		return nil
	}
	// Parse major.minor from semver strings
	cv := parseMajorMinor(current)
	tv := parseMajorMinor(target)
	if cv[0] == 0 && cv[1] == 0 {
		return nil // can't parse, allow
	}
	if tv[0] == 0 && tv[1] == 0 {
		return nil
	}
	minorDiff := tv[1] - cv[1]
	if tv[0] > cv[0] {
		return fmt.Errorf("major version upgrades must be done one step at a time (current: %s, target: %s)", current, target)
	}
	if minorDiff > 1 {
		return fmt.Errorf(
			"cannot skip minor versions: upgrade one minor version at a time (current: %s, target: %s)\n"+
				"  Upgrade path: %s → v%d.%d.0 first",
			current, target, current, cv[0], cv[1]+1,
		)
	}
	return nil
}

func helmDiff(cli *sshclient.Client, version string) (string, error) {
	chartRef := "oci://ghcr.io/dotechhq/zenith/charts/zenith"
	versionFlag := ""
	if version != "" && version != "latest" {
		versionFlag = " --version " + version
	}
	out, err := cli.Run(fmt.Sprintf(
		"helm diff upgrade zenith %s%s -n zenith-system 2>&1 || echo 'helm-diff plugin not installed — run: helm plugin install https://github.com/databus23/helm-diff'",
		chartRef, versionFlag,
	))
	return out, err
}

func helmUpgrade(cli *sshclient.Client, version string) error {
	chartRef := "oci://ghcr.io/dotechhq/zenith/charts/zenith"
	versionFlag := ""
	if version != "" && version != "latest" {
		versionFlag = " --version " + version
	}
	_, err := cli.Run(fmt.Sprintf(
		"helm upgrade zenith %s%s -n zenith-system --reuse-values --wait --timeout 15m 2>&1",
		chartRef, versionFlag,
	))
	return err
}

func helmRollback(cli *sshclient.Client) error {
	_, err := cli.Run("helm rollback zenith -n zenith-system --wait --timeout 5m 2>&1")
	return err
}

func parseMajorMinor(version string) [2]int {
	// Strip leading 'v' and 'zenith-' prefix
	v := strings.TrimPrefix(version, "v")
	v = strings.TrimPrefix(v, "zenith-")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return [2]int{0, 0}
	}
	var major, minor int
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	return [2]int{major, minor}
}


func formatDuration(d time.Duration) string {
	if d == 0 {
		return "done"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
