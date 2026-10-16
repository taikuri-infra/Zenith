package logout

import (
	"fmt"
	"io"
	"strings"
	"time"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

// Cmd is the logout command.
var Cmd = &cobra.Command{
	Use:   "logout",
	Short: "Forget the saved session",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogout(cmd.OutOrStdout(), config.DefaultConfigPath())
	},
}

// runLogout revokes a login session server-side (best effort) and clears it from the config at path.
// Deploy tokens (znt_id_…) are not sessions, so the server is not asked to revoke them.
func runLogout(w io.Writer, path string) error {
	cfg, err := config.LoadFrom(path)
	if err != nil {
		return err
	}
	if cfg.Token != "" && !strings.HasPrefix(cfg.Token, "znt_id_") {
		client := cliapi.NewClient(cfg.APIEndpoint, cfg.Token)
		client.RefreshToken = cfg.RefreshToken
		_ = client.Logout() // best effort
	}
	cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = "", "", time.Time{}
	if err := cfg.SaveTo(path); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s Logged out of %s\n", tui.Green("✓"), cfg.APIEndpoint)
	return nil
}
