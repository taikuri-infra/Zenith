// Package backup implements `zen backup` and `zen backup restore`.
package backup

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/cli/internal/k8sops"
	"github.com/spf13/cobra"
)

var (
	flagKubeconfig, flagOut string
	flagYes                 bool
)

// Cmd is the `zen backup` command.
var Cmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up Zenith: platform database, object storage, managed databases and secrets",
	Long: `Write one archive with the platform database, object storage, every managed
database and the zenith-secrets Secret (it holds the encryption keys).

Run it where kubectl reaches the cluster (on a k3s server it works as root).
Copy the archive off the server: it lives on the same disk as the data.

Examples:
  zen backup
  zen backup --out /mnt/backups
  zen backup restore zenith-backup-0.2.0-20261025T120000Z.tar.gz --yes`,
	Args: cobra.NoArgs,
	RunE: runBackup,
}

var restoreCmd = &cobra.Command{
	Use:   "restore <archive>",
	Short: "Restore Zenith from a zen backup archive",
	Long: `Replace the platform database, object storage, managed databases and secrets
of this installation with the contents of an archive written by "zen backup".
The API is stopped while the data is replaced.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	Cmd.Flags().StringVar(&flagKubeconfig, "kubeconfig", "", "Kubeconfig of the cluster (default: $KUBECONFIG or k3s)")
	Cmd.Flags().StringVar(&flagOut, "out", "zenith-backups", "Directory for the archive")
	restoreCmd.Flags().BoolVar(&flagYes, "yes", false, "Skip the confirmation prompt")
	restoreCmd.Flags().StringVar(&flagKubeconfig, "kubeconfig", "", "Kubeconfig of the cluster (default: $KUBECONFIG or k3s)")
	Cmd.AddCommand(restoreCmd)
}

func runBackup(cmd *cobra.Command, _ []string) error {
	runner := k8sops.ExecRunner{}
	inst, ok := k8sops.Detect(cmd.Context(), runner, flagKubeconfig)
	if !ok {
		return fmt.Errorf("no Zenith installation found (use --kubeconfig)")
	}
	path, err := k8sops.Backup(cmd.Context(), runner, inst, flagOut, time.Now())
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), path)
	fmt.Fprintln(cmd.OutOrStdout(), "This archive contains your secrets (the encryption keys).")
	fmt.Fprintln(cmd.OutOrStdout(), "Copy it off this server: it is on the same disk as the data.")
	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	runner := k8sops.ExecRunner{}
	inst, ok := k8sops.Detect(cmd.Context(), runner, flagKubeconfig)
	if !ok {
		return fmt.Errorf("no Zenith installation found (use --kubeconfig)")
	}
	out := cmd.OutOrStdout()
	if !flagYes {
		fmt.Fprintf(out, "This replaces all data of the installation in namespace %s. Type 'restore' to continue: ", inst.Namespace)
		line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
		if strings.TrimSpace(line) != "restore" {
			return fmt.Errorf("aborted")
		}
	}
	m, err := k8sops.Restore(cmd.Context(), runner, inst, args[0], k8sops.RestoreOptions{})
	if m == nil {
		return err // nothing restored; cobra prints the (already descriptive) error once
	}
	failed := 0
	if err != nil {
		if j, ok := err.(interface{ Unwrap() []error }); ok {
			failed = len(j.Unwrap())
		} else {
			failed = 1
		}
	}
	fmt.Fprintf(out, "Restored archive from Zenith %s: %d of %d managed database(s) restored, %d failed.\n",
		m.Version, len(m.Databases)-failed, len(m.Databases), failed)
	if err != nil {
		fmt.Fprintln(out, "Failures:")
		fmt.Fprintln(out, err.Error())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		return fmt.Errorf("restore finished with %d failed database(s)", failed)
	}
	fmt.Fprintln(out, "Restore complete.")
	return nil
}
