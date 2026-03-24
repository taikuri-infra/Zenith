package login

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

// Cmd is the login command.
var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Zenith Cloud",
	Long: `Log in to Zenith Cloud by providing your API endpoint and token.

The token can be found in your Zenith dashboard under Settings > API Tokens.`,
	RunE: runLogin,
}

var (
	flagEndpoint string
	flagToken    string
)

func init() {
	Cmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "Zenith API endpoint URL")
	Cmd.Flags().StringVar(&flagToken, "token", "", "Authentication token")
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	reader := bufio.NewReader(os.Stdin)

	// Get endpoint
	endpoint := flagEndpoint
	if endpoint == "" {
		defaultEndpoint := cfg.APIEndpoint
		if defaultEndpoint == "" {
			defaultEndpoint = "https://api.freezenith.com"
		}
		fmt.Printf("%s API Endpoint [%s]: ", tui.Cyan("?"), defaultEndpoint)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			endpoint = input
		} else {
			endpoint = defaultEndpoint
		}
	}

	// Get token
	token := flagToken
	if token == "" {
		fmt.Printf("%s Token: ", tui.Cyan("?"))
		input, _ := reader.ReadString('\n')
		token = strings.TrimSpace(input)
	}

	if token == "" {
		return fmt.Errorf("token is required")
	}

	cfg.APIEndpoint = endpoint
	cfg.Token = token

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n%s Logged in successfully!\n", tui.Green("✓"))
	fmt.Printf("  Endpoint: %s\n", endpoint)
	fmt.Printf("  Config saved to: %s\n", config.DefaultConfigPath())

	return nil
}
