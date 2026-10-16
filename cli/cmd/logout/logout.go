package logout

import (
	"fmt"
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.Token != "" && cfg.RefreshToken != "" {
			_ = cliapi.NewClientFromConfig(cfg).Logout() // best effort
		}
		cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = "", "", time.Time{}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s Logged out of %s\n", tui.Green("✓"), cfg.APIEndpoint)
		return nil
	},
}
