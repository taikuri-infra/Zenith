package root

import (
	"github.com/dotechhq/zenith/cli/cmd/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zen",
	Short: "Zenith - Kubernetes-native PaaS on Hetzner Cloud",
	Long: `Zenith is a 100% free, open-source, Kubernetes-native PaaS on Hetzner Cloud.

One command installs everything:
  zen install --provider hetzner --token hc_xxx

You get: Apps, Databases, Storage, Auth, Gateway, Monitoring, Registry.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(version.Cmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func GetRootCmd() *cobra.Command {
	return rootCmd
}
