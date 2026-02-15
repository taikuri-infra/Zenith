package root

import (
	"github.com/dotechhq/zenith/cli/cmd/apply"
	"github.com/dotechhq/zenith/cli/cmd/db"
	"github.com/dotechhq/zenith/cli/cmd/deploy"
	"github.com/dotechhq/zenith/cli/cmd/diff"
	exportcmd "github.com/dotechhq/zenith/cli/cmd/export"
	"github.com/dotechhq/zenith/cli/cmd/install"
	"github.com/dotechhq/zenith/cli/cmd/logs"
	"github.com/dotechhq/zenith/cli/cmd/status"
	"github.com/dotechhq/zenith/cli/cmd/top"
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
	rootCmd.AddCommand(install.Cmd)
	rootCmd.AddCommand(deploy.Cmd)
	rootCmd.AddCommand(status.Cmd)
	rootCmd.AddCommand(top.Cmd)
	rootCmd.AddCommand(logs.Cmd)
	rootCmd.AddCommand(db.Cmd)
	rootCmd.AddCommand(exportcmd.Cmd)
	rootCmd.AddCommand(apply.Cmd)
	rootCmd.AddCommand(diff.Cmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func GetRootCmd() *cobra.Command {
	return rootCmd
}
