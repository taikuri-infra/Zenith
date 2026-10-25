// Package install implements `zen install`: Zenith on k3s, on this machine.
package install

import (
	"fmt"
	"os"

	"github.com/dotechhq/zenith/cli/internal/k8sops"
	"github.com/spf13/cobra"
)

var (
	flagDomain, flagEmail, flagVersion, flagAdminEmail, flagAdminPassword string
	flagDryRun                                                              bool
)

// Cmd is the `zen install` command.
var Cmd = &cobra.Command{
	Use:   "install",
	Short: "Install Zenith (k3s, database operators and the platform) on this machine",
	Long: `Install Zenith on this machine. Run it as root on a fresh Linux server.

It installs k3s, cert-manager and the database operators (CloudNativePG,
mariadb-operator, redis-operator), then the Zenith Helm chart. Safe to re-run:
existing secrets are kept. Without --domain the server's public IP is used as
<ip>.sslip.io.

Examples:
  sudo zen install
  sudo zen install --domain zenith.example.com --email you@example.com`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	f := Cmd.Flags()
	f.StringVar(&flagDomain, "domain", "", "Domain Zenith is served on (default <public-ip>.sslip.io)")
	f.StringVar(&flagEmail, "email", "", "Email for Let's Encrypt certificates")
	f.StringVar(&flagVersion, "version", "", "Zenith version to install (default latest)")
	f.StringVar(&flagAdminEmail, "admin-email", "", "Email of the first admin (default admin@<domain>)")
	f.StringVar(&flagAdminPassword, "admin-password", "", "Password of the first admin (default: generated and printed)")
	f.BoolVar(&flagDryRun, "dry-run", false, "Print the plan without changing anything")
}

func run(cmd *cobra.Command, _ []string) error {
	env := map[string]string{}
	set := func(k, v string) {
		if v != "" {
			env[k] = v
		}
	}
	set("ZENITH_DOMAIN", flagDomain)
	set("ZENITH_EMAIL", flagEmail)
	set("ZENITH_ADMIN_EMAIL", flagAdminEmail)
	set("ZENITH_ADMIN_PASSWORD", flagAdminPassword)
	if flagVersion != "" {
		v := trimV(flagVersion)
		set("ZENITH_VERSION", v)
		set("ZENITH_BRANCH", "v"+v)
	}
	if flagDryRun {
		env["ZENITH_DRY_RUN"] = "1"
	}
	if err := k8sops.InstallPlatform(cmd.Context(), env, os.Stdout); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	return nil
}

func trimV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}
