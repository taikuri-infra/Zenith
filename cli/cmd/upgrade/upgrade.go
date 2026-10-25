// Package upgrade implements `zen upgrade`.
package upgrade

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dotechhq/zenith/cli/internal/k8sops"
	"github.com/spf13/cobra"
)

var (
	flagVersion, flagKubeconfig, flagBackupDir string
	flagDryRun, flagSkipBackup, flagForce      bool
)

// Cmd is the `zen upgrade` command.
var Cmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Zenith to a new version",
	Long: `Upgrade the Zenith installation to a new release.

Takes a backup first, upgrades the Helm release with the settings it already
has, and rolls back automatically if the upgrade fails.

Examples:
  zen upgrade
  zen upgrade --version 0.3.0
  zen upgrade --skip-backup
  zen upgrade --dry-run`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	f := Cmd.Flags()
	f.StringVar(&flagVersion, "version", "", "Version to upgrade to (default: the latest release)")
	f.StringVar(&flagKubeconfig, "kubeconfig", "", "Kubeconfig of the cluster (default: $KUBECONFIG or k3s)")
	f.StringVar(&flagBackupDir, "backup-dir", "", "Where the pre-upgrade backup goes (default ./zenith-backups)")
	f.BoolVar(&flagDryRun, "dry-run", false, "Show what would happen without doing it")
	f.BoolVar(&flagSkipBackup, "skip-backup", false, "Do not back up first")
	f.BoolVar(&flagForce, "force", false, "Allow a downgrade or re-install of the same version")
}

func run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	runner := k8sops.ExecRunner{}
	inst, ok := k8sops.Detect(ctx, runner, flagKubeconfig)
	if !ok {
		return fmt.Errorf("no Zenith installation found (use --kubeconfig)")
	}
	target := strings.TrimPrefix(flagVersion, "v")
	if target == "" {
		latest, err := k8sops.LatestRelease(ctx, "taikuri-infra/Zenith")
		if err != nil {
			return fmt.Errorf("resolve latest release: %w", err)
		}
		target = latest
	}
	res, err := k8sops.Upgrade(ctx, runner, inst, k8sops.UpgradeOptions{
		Target: target, Force: flagForce, SkipBackup: flagSkipBackup, DryRun: flagDryRun,
		BackupDir: flagBackupDir, Now: time.Now,
	}, os.Stdout)
	if res != nil && !flagDryRun {
		fmt.Printf("%s -> %s\n", res.From, res.To)
		if res.Backup != "" {
			fmt.Printf("backup: %s\n", res.Backup)
		}
		if res.RolledBack {
			fmt.Println("rolled back")
		} else if res.RollbackFailed {
			fmt.Println("rollback FAILED")
		}
	}
	return err
}
