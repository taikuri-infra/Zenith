// Package uninstall implements `zen uninstall`.
package uninstall

import (
	"fmt"

	"github.com/dotechhq/zenith/cli/internal/k8sops"
	"github.com/spf13/cobra"
)

var (
	flagKubeconfig string
	flagYes        bool
)

// Cmd is the `zen uninstall` command.
var Cmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Zenith and all of its data from the cluster",
	Long: `Uninstall the zenith Helm release and delete the zenith-system and zenith-apps
namespaces: the platform, every app and every managed database, with their volumes.
k3s, cert-manager and the database operators stay; remove k3s itself with
/usr/local/bin/k3s-uninstall.sh.

Take a backup first (zen backup). This cannot be undone.`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	Cmd.Flags().StringVar(&flagKubeconfig, "kubeconfig", "", "Kubeconfig of the cluster (default: $KUBECONFIG or k3s)")
	Cmd.Flags().BoolVar(&flagYes, "yes", false, "Confirm the removal of all data")
}

func run(cmd *cobra.Command, _ []string) error {
	if !flagYes {
		return fmt.Errorf("this deletes all Zenith data; re-run with --yes")
	}
	runner := k8sops.ExecRunner{}
	inst, ok := k8sops.Detect(cmd.Context(), runner, flagKubeconfig)
	if !ok {
		return fmt.Errorf("no Zenith installation found (use --kubeconfig)")
	}
	if err := k8sops.Uninstall(cmd.Context(), runner, inst); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Zenith removed.")
	return nil
}
