// Package backup provides the `zen backup` command for triggering and monitoring
// CNPG (CloudNativePG) database backups on the remote Zenith server.
package backup

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/installstate"
	"github.com/dotechhq/zenith/cli/internal/sshclient"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagReason string
	flagDomain string
)

// Cmd is the `zen backup` top-level command. Subcommands are registered below.
var Cmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage Zenith database backups",
	Long: `Trigger and monitor database backups for a Zenith installation.

Examples:
  zen backup create
  zen backup create --reason "before upgrade"`,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Trigger an immediate CNPG database backup",
	Long: `Annotate the zenith-postgres cluster to trigger an immediate CloudNativePG backup,
then wait until the backup completes.

Examples:
  zen backup create
  zen backup create --reason "before upgrade"
  zen backup create --domain mycompany.com`,
	RunE: runBackupCreate,
}

func init() {
	createCmd.Flags().StringVar(&flagReason, "reason", "", "Human-readable reason for this backup (informational only)")
	createCmd.Flags().StringVar(&flagDomain, "domain", "", "Domain of the Zenith installation to back up (uses saved state if omitted)")
	Cmd.AddCommand(createCmd)
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	muted := lipgloss.NewStyle().Foreground(tui.ColorMuted).PaddingLeft(2)
	successStyle := lipgloss.NewStyle().Foreground(tui.ColorSuccess).Bold(true).PaddingLeft(2)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError).PaddingLeft(2)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).PaddingLeft(2)

	fmt.Println()
	fmt.Println(headerStyle.Render("Zenith Backup"))
	if flagReason != "" {
		fmt.Println(muted.Render("Reason: " + sanitizeReason(flagReason)))
	}
	fmt.Println()

	// Load install state to get server IP and SSH key
	if !installstate.Exists() {
		return fmt.Errorf("no Zenith installation state found — run 'zen install' first")
	}
	state, err := installstate.Load()
	if err != nil {
		return fmt.Errorf("failed to load installation state: %w", err)
	}

	if state.ServerIP == "" {
		return fmt.Errorf("server IP is not set in installation state — cannot connect via SSH")
	}

	fmt.Println(muted.Render(fmt.Sprintf("Connecting to %s...", state.ServerIP)))

	// Build SSH config from saved state.
	// TODO: wire state.ServerHostKey into KnownHostKey once installstate adds the field
	// and sshclient.Config exposes KnownHostKey []byte (parallel tasks).
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

	cli, err := sshclient.DialWithRetry(sshCfg, 3, 5*time.Second)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer cli.Close()

	fmt.Println(muted.Render("Connected. Triggering immediate backup..."))

	// Trigger CNPG immediate backup via annotation.
	// KUBECONFIG prefix is required on k3s hosts where the default path differs.
	annotateCmd := `KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl annotate cluster.postgresql.cnpg.io/zenith-postgres -n zenith-system backup.cnpg.io/immediate="true" --overwrite 2>&1`
	out, err := cli.Run(annotateCmd)
	if err != nil {
		fmt.Println(errStyle.Render("Failed to annotate cluster: " + out))
		return fmt.Errorf("annotate failed: %w", err)
	}
	fmt.Println(muted.Render("Backup triggered. Waiting for completion (up to 5 minutes)..."))

	// Poll for backup completion — up to 5 minutes
	const (
		pollInterval = 5 * time.Second
		backupTimeout = 5 * time.Minute
	)
	deadline := time.Now().Add(backupTimeout)
	pollCmd := `KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl get backup -n zenith-system --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1:].status.phase}' 2>/dev/null`

	for time.Now().Before(deadline) {
		phase, pollErr := cli.Run(pollCmd)
		if pollErr == nil {
			phase = strings.TrimSpace(strings.Trim(phase, "'"))
		}

		if phase != "" {
			switch strings.ToLower(phase) {
			case "completed":
				fmt.Println()
				fmt.Println(successStyle.Render("Backup completed successfully!"))
				fmt.Println()
				return nil
			case "failed":
				fmt.Println()
				fmt.Println(errStyle.Render("Backup failed (CNPG reported phase=failed)."))
				fmt.Println()
				return fmt.Errorf("backup phase=failed")
			default:
				// In-progress phases: running, pending, etc.
				fmt.Printf("\r%s", muted.Render(fmt.Sprintf("  Backup phase: %-12s (waiting...)", phase)))
			}
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timed out waiting for backup to complete after %s", backupTimeout)
}

// sanitizeReason strips control characters and limits length to prevent terminal injection
// when displaying the --reason flag value in output.
func sanitizeReason(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 { // printable ASCII and beyond, no DEL
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) > 200 {
		result = result[:200]
	}
	return result
}
