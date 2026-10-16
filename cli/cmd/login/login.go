package login

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Cmd is the login command.
var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to a Zenith server",
	Long: `Log in to your Zenith server with the email and password you use on the dashboard.

The session is stored in ~/.zen/config.yaml and renewed automatically.
A pasted token (--token) is also accepted, for example a deploy token for CI.`,
	RunE: runLogin,
}

var (
	flagEndpoint string
	flagToken    string
	flagEmail    string
	flagPassword string
)

func init() {
	Cmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "Zenith server URL, e.g. https://zenith.example.com")
	Cmd.Flags().StringVar(&flagEmail, "email", "", "Account email")
	Cmd.Flags().StringVar(&flagPassword, "password", "", "Account password (or set ZEN_PASSWORD)")
	Cmd.Flags().StringVar(&flagToken, "token", "", "Use a pasted token instead of email and password")
}

// loginInputs is everything runLoginTo needs; the command fills it from flags and prompts.
type loginInputs struct {
	endpoint   string
	email      string
	password   string
	token      string
	configPath string
	promptMFA  func() (string, error)
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	reader := bufio.NewReader(os.Stdin)
	in := loginInputs{endpoint: flagEndpoint, email: flagEmail, password: flagPassword, token: flagToken, configPath: config.DefaultConfigPath()}
	if in.password == "" {
		in.password = os.Getenv("ZEN_PASSWORD")
	}
	if in.endpoint == "" {
		in.endpoint = prompt(reader, fmt.Sprintf("Server URL [%s]", cfg.APIEndpoint), cfg.APIEndpoint)
	}
	if in.token == "" {
		if in.email == "" {
			in.email = prompt(reader, "Email", "")
		}
		if in.password == "" {
			in.password = promptSecret(reader, "Password")
		}
	}
	in.promptMFA = func() (string, error) { return prompt(reader, "Code", ""), nil }
	return runLoginTo(cmd.OutOrStdout(), in)
}

// runLoginTo performs the login and saves the session. It is the testable core of the command.
func runLoginTo(w io.Writer, in loginInputs) error {
	in.endpoint = strings.TrimRight(strings.TrimSpace(in.endpoint), "/")
	if in.endpoint == "" {
		return fmt.Errorf("server URL is required (--endpoint https://zenith.example.com)")
	}
	cfg, err := config.LoadFrom(in.configPath)
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.APIEndpoint = in.endpoint

	if in.token != "" {
		cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = in.token, "", time.Time{}
		if err := cfg.SaveTo(in.configPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Fprintf(w, "%s Logged in to %s with a token\n", tui.Green("✓"), in.endpoint)
		return nil
	}
	if in.email == "" || in.password == "" {
		return fmt.Errorf("email and password are required")
	}

	client := cliapi.NewClient(in.endpoint, "")
	ts, err := client.LoginWithPassword(in.email, in.password)
	var mfa *cliapi.ErrMFARequired
	if errors.As(err, &mfa) {
		if in.promptMFA == nil {
			return fmt.Errorf("this account needs a multi-factor code")
		}
		code, perr := in.promptMFA()
		if perr != nil {
			return perr
		}
		ts, err = client.LoginMFA(mfa.MFAToken, strings.TrimSpace(code))
	}
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = ts.AccessToken, ts.RefreshToken, ts.ExpiresAt
	if err := cfg.SaveTo(in.configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(w, "%s Logged in to %s as %s\n", tui.Green("✓"), in.endpoint, in.email)
	fmt.Fprintf(w, "  Config saved to: %s\n", in.configPath)
	return nil
}

func prompt(r *bufio.Reader, label, def string) string {
	fmt.Printf("%s %s: ", tui.Cyan("?"), label)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// promptSecret reads a password without echo when stdin is a terminal.
func promptSecret(r *bufio.Reader, label string) string {
	fmt.Printf("%s %s: ", tui.Cyan("?"), label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
