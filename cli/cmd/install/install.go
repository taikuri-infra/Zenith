package install

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/install"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagToken      string
	flagRegion     string
	flagServerType string
	flagNonInteractive bool
)

var Cmd = &cobra.Command{
	Use:   "install",
	Short: "Install Zenith platform on Hetzner Cloud",
	Long: `Install the Zenith platform on a new Hetzner Cloud server.

This creates a management plane with k3s, CAPI, and all Zenith components.
The entire process takes about 3 minutes.

Example:
  zen install --provider hetzner --token hc_xxx`,
	RunE: runInstall,
}

func init() {
	Cmd.Flags().StringVar(&flagToken, "token", "", "Hetzner Cloud API token")
	Cmd.Flags().StringVar(&flagRegion, "region", "", "Hetzner Cloud region (fsn1, nbg1, hel1, ash, hil)")
	Cmd.Flags().StringVar(&flagServerType, "server-type", "", "Server type (cx22, cx32, cx42)")
	Cmd.Flags().BoolVar(&flagNonInteractive, "non-interactive", false, "Skip interactive prompts")
	// --provider flag is accepted but hetzner is the only option
	Cmd.Flags().String("provider", "hetzner", "Cloud provider (only hetzner supported)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg := &install.Config{}

	// Show header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Zenith Platform Installer")
	fmt.Println(header)
	fmt.Println()

	if flagNonInteractive {
		// Use flags directly
		if flagToken == "" {
			return fmt.Errorf("--token is required in non-interactive mode")
		}
		cfg.HetznerToken = flagToken
		cfg.Region = flagRegion
		if cfg.Region == "" {
			cfg.Region = "fsn1"
		}
		cfg.ServerType = flagServerType
		if cfg.ServerType == "" {
			cfg.ServerType = "cx22"
		}
	} else {
		// Interactive wizard
		var err error
		cfg, err = runWizard()
		if err != nil {
			return err
		}
	}

	// Validate
	if err := install.ValidateToken(cfg.HetznerToken); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Run installation steps
	return runSteps(cfg)
}

func runWizard() (*install.Config, error) {
	var token, regionSel, serverTypeSel string

	// Pre-fill from flags
	token = flagToken
	regionSel = flagRegion
	serverTypeSel = flagServerType

	regionOptions := make([]huh.Option[string], len(install.Regions))
	for i, r := range install.Regions {
		label := fmt.Sprintf("%s - %s, %s", r.ID, r.Name, r.Country)
		regionOptions[i] = huh.NewOption(label, r.ID)
	}
	if regionSel == "" {
		regionSel = "fsn1"
	}

	serverTypeOptions := make([]huh.Option[string], len(install.ServerTypes))
	for i, s := range install.ServerTypes {
		label := fmt.Sprintf("%s - %s (€%.2f/mo)", s.ID, s.Description, s.Price)
		serverTypeOptions[i] = huh.NewOption(label, s.ID)
	}
	if serverTypeSel == "" {
		serverTypeSel = "cx22"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Hetzner Cloud API Token").
				Description("Create one at console.hetzner.cloud → API Tokens").
				Placeholder("hc_xxxxxxxxxxxx").
				Value(&token).
				Validate(func(s string) error {
					return install.ValidateToken(s)
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Region").
				Description("Select the Hetzner Cloud region for your management server").
				Options(regionOptions...).
				Value(&regionSel),
			huh.NewSelect[string]().
				Title("Server Type").
				Description("Select the server type for your management plane").
				Options(serverTypeOptions...).
				Value(&serverTypeSel),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled: %w", err)
	}

	return &install.Config{
		HetznerToken: token,
		Region:       regionSel,
		ServerType:   serverTypeSel,
	}, nil
}

func runSteps(cfg *install.Config) error {
	steps := install.GetInstallSteps(cfg)

	checkmark := lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("✓")
	spinner := lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("⠋")
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	totalStart := time.Now()

	fmt.Println()
	for i, step := range steps {
		fmt.Printf("  %s %s\n", spinner, stepStyle.Render(step.Name))

		start := time.Now()
		if err := step.Action(cfg); err != nil {
			errMark := lipgloss.NewStyle().Foreground(tui.ColorError).Render("✗")
			fmt.Printf("\r  %s %s - %s\n", errMark,
				stepStyle.Render(step.Name),
				lipgloss.NewStyle().Foreground(tui.ColorError).Render(err.Error()))
			return fmt.Errorf("step %d failed: %w", i+1, err)
		}
		elapsed := time.Since(start)

		// Move up and overwrite the spinner line
		fmt.Printf("\033[1A\r  %s %s %s\n", checkmark,
			stepStyle.Render(step.Name),
			timeStyle.Render(fmt.Sprintf("(%s)", elapsed.Round(time.Millisecond))))
	}

	totalElapsed := time.Since(totalStart)

	// Success box
	fmt.Println()
	successBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSuccess).
		Padding(1, 2).
		Render(fmt.Sprintf(
			"%s\n\n"+
				"  %s %s\n"+
				"  %s %s\n"+
				"  %s %s\n\n"+
				"  %s",
			lipgloss.NewStyle().Bold(true).Foreground(tui.ColorSuccess).Render("Zenith installed successfully!"),
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Region:"),
			cfg.Region,
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Server:"),
			cfg.ServerType,
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Time:"),
			totalElapsed.Round(time.Millisecond).String(),
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Next: Open the welcome wizard in your browser"),
		))
	fmt.Println(successBox)

	// Next steps
	fmt.Println()
	nextSteps := []string{
		"Open the management panel to create your first cluster",
		"Run 'zen status' to check platform health",
		"Run 'zen deploy' to deploy your first app",
	}
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).Render("  Next steps:"))
	for _, step := range nextSteps {
		fmt.Printf("  %s %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("→"),
			strings.TrimSpace(step))
	}
	fmt.Println()

	return nil
}
