package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dotechhq/zenith/services/cli/internal/api"
	"github.com/dotechhq/zenith/services/cli/internal/config"
)

var (
	version = "dev"
	cfg     *config.Config
	client  *api.Client
)

func main() {
	cfg = config.Load()
	client = api.New(cfg)

	rootCmd := &cobra.Command{
		Use:     "zenith",
		Short:   "Zenith PaaS CLI",
		Long:    "Deploy, manage, and monitor your apps on the Zenith platform.",
		Version: version,
	}

	rootCmd.AddCommand(
		loginCmd(),
		logoutCmd(),
		projectCmd(),
		appsCmd(),
		dbCmd(),
		storageCmd(),
		domainsCmd(),
		logsCmd(),
		metricsCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func requireAuth() {
	if !cfg.IsLoggedIn() {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'zenith login' first.")
		os.Exit(1)
	}
}
