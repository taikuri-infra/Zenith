package install

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/install"
	"github.com/dotechhq/zenith/cli/internal/installstate"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

// CLI flags for non-interactive (power-user) mode.
var (
	flagDomain        string
	flagProvider      string
	flagHetznerToken  string
	flagRegion        string
	flagDNSProvider   string
	flagDNSToken      string
	flagWithCluster   bool
	flagSSHHost       string
	flagSSHUser       string
	flagServerType    string
	flagResume        bool
	flagDryRun        bool
	flagChartVersion  string
	flagEdition       string
	flagLocal         bool
	flagFreeDomain    bool
	flagRegisterURL   string
	flagRegisterToken string
)

var Cmd = &cobra.Command{
	Use:   "install",
	Short: "Install Zenith Mission Control",
	Long: `Install the Zenith Mission Control platform.

Interactive wizard (default):
  zen install

Non-interactive (power-user) mode:
  zen install --domain embermind.app \
    --provider hetzner --hetzner-token hc_xxx \
    --region nuremberg \
    --dns-provider cloudflare --dns-token cf_xxx \
    --with-cluster`,
	RunE: runInstall,
}

func init() {
	f := Cmd.Flags()
	f.StringVar(&flagDomain, "domain", "", "Your domain (e.g., embermind.app)")
	f.StringVar(&flagProvider, "provider", "", "Server provider: hetzner or existing")
	f.StringVar(&flagHetznerToken, "hetzner-token", "", "Hetzner Cloud API token")
	f.StringVar(&flagRegion, "region", "", "Server region (nuremberg, falkenstein, helsinki, ashburn)")
	f.StringVar(&flagServerType, "server-type", "cx22", "Hetzner server type (cx22, cx32, cx42)")
	f.StringVar(&flagDNSProvider, "dns-provider", "", "DNS provider: cloudflare or manual")
	f.StringVar(&flagDNSToken, "dns-token", "", "Cloudflare API token")
	f.BoolVar(&flagWithCluster, "with-cluster", false, "Create first cluster immediately")
	f.StringVar(&flagSSHHost, "ssh-host", "", "SSH host/IP for existing server")
	f.StringVar(&flagSSHUser, "ssh-user", "root", "SSH user for existing server")
	f.BoolVar(&flagDryRun, "dry-run", false, "Simulate installation without making real API calls")
	f.BoolVar(&flagResume, "resume", false, "Resume a previously interrupted installation, skipping completed steps")
	f.StringVar(&flagChartVersion, "chart-version", "", "Helm chart version to install (default: latest)")
	f.StringVar(&flagEdition, "edition", "", "Edition: 'compose' (self-host on any Linux box, no Kubernetes) or 'cloud' (Hetzner/k8s)")
	f.BoolVar(&flagLocal, "local", false, "Compose edition: install on this machine instead of over SSH")
	f.BoolVar(&flagFreeDomain, "free-domain", false, "Compose edition: reserve a free <slug>.apps.freezenith.com with automatic HTTPS")
	f.StringVar(&flagRegisterURL, "register-url", "", "Override the subdomain-registration service URL (default: https://register.freezenith.com)")
	f.StringVar(&flagRegisterToken, "register-token", "", "Install token for the registration service (defaults to $FREEZENITH_REGISTER_TOKEN)")

	// Accept legacy --token flag as alias for --hetzner-token
	f.String("token", "", "Alias for --hetzner-token (deprecated)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Show header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		PaddingLeft(2).
		Render("Zenith Platform Installer")
	subtitle := lipgloss.NewStyle().
		Foreground(tui.ColorMuted).
		PaddingLeft(2).
		Render("Bootstrap Mission Control for your infrastructure")
	fmt.Println()
	fmt.Println(header)
	fmt.Println(subtitle)
	fmt.Println()

	// Handle --resume: load saved state and populate config from it
	var resumeState *installstate.State
	if flagResume {
		if !installstate.Exists() {
			return fmt.Errorf("--resume: no saved installation state found at ~/.zen/install-state.yaml")
		}
		var err error
		resumeState, err = installstate.Load()
		if err != nil {
			return fmt.Errorf("--resume: failed to load state: %w", err)
		}
		if len(resumeState.CompletedSteps) == 0 {
			fmt.Println(lipgloss.NewStyle().
				Foreground(tui.ColorMuted).
				PaddingLeft(2).
				Render("No completed steps found — starting from the beginning."))
		} else {
			fmt.Println(lipgloss.NewStyle().
				Foreground(tui.ColorMuted).
				PaddingLeft(2).
				Render(fmt.Sprintf("Resuming install — skipping %d completed step(s): %s",
					len(resumeState.CompletedSteps),
					strings.Join(resumeState.CompletedSteps, ", "))))
		}
		fmt.Println()
	}

	var cfg *install.Config

	if isNonInteractive(cmd) {
		// Non-interactive mode: build config from flags
		var err error
		cfg, err = buildConfigFromFlags(cmd)
		if err != nil {
			return err
		}
	} else if flagResume && resumeState != nil {
		// Resume mode without explicit flags: rebuild config from saved state
		cfg = configFromState(resumeState)
	} else {
		// Interactive wizard — asks edition first, then compose or cloud.
		result, err := install.RunInteractive()
		if err != nil {
			return err
		}
		if !result.Confirmed {
			fmt.Println(lipgloss.NewStyle().
				Foreground(tui.ColorMuted).
				PaddingLeft(2).
				Render("Installation cancelled."))
			return nil
		}
		cfg = result.Config
		if cfg.Edition == "compose" {
			// Operator-side defaults not asked in the wizard.
			cfg.RegisterURL = flagRegisterURL
			cfg.RegisterToken = registerTokenOrEnv()
		}
	}

	// Run installation steps with progress display
	if err := runSteps(cfg, resumeState); err != nil {
		return err
	}

	// Build and display results
	if cfg.Edition == "compose" {
		showComposeResult(cfg)
		return nil
	}
	result := install.BuildResult(cfg)
	showResult(cfg, result)

	return nil
}

// isNonInteractive checks if enough flags are set to skip the wizard.
func isNonInteractive(cmd *cobra.Command) bool {
	// --domain (cloud) or --edition (compose) means non-interactive mode.
	return flagDomain != "" || flagEdition != ""
}

// registerTokenOrEnv returns the --register-token flag, or $FREEZENITH_REGISTER_TOKEN.
func registerTokenOrEnv() string {
	if flagRegisterToken != "" {
		return flagRegisterToken
	}
	return os.Getenv("FREEZENITH_REGISTER_TOKEN")
}

// buildComposeConfigFromFlags builds a compose-edition Config from flags.
func buildComposeConfigFromFlags() (*install.Config, error) {
	cfg := &install.Config{
		Edition:       "compose",
		ComposeLocal:  flagLocal,
		Domain:        flagDomain, // optional; empty/localhost => local HTTP
		SSHUser:       flagSSHUser,
		FreeSubdomain: flagFreeDomain,
		RegisterURL:   flagRegisterURL,
		RegisterToken: registerTokenOrEnv(),
		DryRun:        flagDryRun,
	}
	// Optional: auto-create the A record for a custom domain on Cloudflare.
	if flagDNSProvider != "" {
		cfg.DNSProvider = install.DNSProvider(strings.ToLower(flagDNSProvider))
		cfg.CloudflareToken = flagDNSToken
	}
	if !flagLocal {
		if flagSSHHost == "" {
			return nil, fmt.Errorf("compose edition needs either --local or --ssh-host")
		}
		cfg.SSHHost = flagSSHHost
	}
	if cfg.Domain != "" && cfg.Domain != "localhost" {
		if err := install.ValidateDomain(cfg.Domain); err != nil {
			return nil, fmt.Errorf("invalid domain: %w", err)
		}
	}
	return cfg, nil
}

// buildConfigFromFlags creates a Config from CLI flags.
func buildConfigFromFlags(cmd *cobra.Command) (*install.Config, error) {
	if strings.ToLower(flagEdition) == "compose" {
		return buildComposeConfigFromFlags()
	}
	cfg := &install.Config{
		Domain:     flagDomain,
		ServerType: flagServerType,
	}

	// Handle legacy --token flag
	token := flagHetznerToken
	if token == "" {
		if legacyToken, _ := cmd.Flags().GetString("token"); legacyToken != "" {
			token = legacyToken
		}
	}

	// Determine provider
	provider := strings.ToLower(flagProvider)
	if provider == "" && token != "" {
		provider = "hetzner"
	}
	if provider == "" && flagSSHHost != "" {
		provider = "existing"
	}
	if provider == "" {
		provider = "hetzner"
	}

	switch provider {
	case "hetzner":
		cfg.MCProvider = install.ProviderHetzner
		cfg.HetznerToken = token
		if cfg.HetznerToken == "" {
			return nil, fmt.Errorf("--hetzner-token is required when using Hetzner provider")
		}
		if err := install.ValidateToken(cfg.HetznerToken); err != nil {
			return nil, fmt.Errorf("invalid Hetzner token: %w", err)
		}
		cfg.Region = resolveRegion(flagRegion)
	case "existing":
		cfg.MCProvider = install.ProviderExisting
		if flagSSHHost == "" {
			return nil, fmt.Errorf("--ssh-host is required when using existing server")
		}
		cfg.SSHHost = flagSSHHost
		cfg.SSHUser = flagSSHUser
	default:
		return nil, fmt.Errorf("unknown provider %q (use 'hetzner' or 'existing')", provider)
	}

	// Domain is required
	if cfg.Domain == "" {
		return nil, fmt.Errorf("--domain is required")
	}
	if err := install.ValidateDomain(cfg.Domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %w", err)
	}

	// DNS provider
	dnsProvider := strings.ToLower(flagDNSProvider)
	switch dnsProvider {
	case "cloudflare":
		cfg.DNSProvider = install.DNSCloudflare
		if flagDNSToken == "" {
			return nil, fmt.Errorf("--dns-token is required when using Cloudflare DNS")
		}
		cfg.CloudflareToken = flagDNSToken
	case "manual", "":
		cfg.DNSProvider = install.DNSManual
	default:
		return nil, fmt.Errorf("unknown dns-provider %q (use 'cloudflare' or 'manual')", dnsProvider)
	}

	// First cluster
	cfg.WithCluster = flagWithCluster
	if cfg.WithCluster {
		// Reuse same provider settings for the cluster
		cfg.ClusterProvider = cfg.MCProvider
		cfg.ClusterHetznerToken = cfg.HetznerToken
		cfg.ClusterRegion = cfg.Region
		cfg.ClusterServerType = cfg.ServerType
		cfg.ClusterSSHHost = cfg.SSHHost
		cfg.ClusterSSHUser = cfg.SSHUser
	}

	cfg.DryRun = flagDryRun
	cfg.ChartVersion = flagChartVersion
	return cfg, nil
}

// resolveRegion maps human-friendly region names to Hetzner region IDs.
func resolveRegion(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	aliases := map[string]string{
		"nuremberg":   "nbg1",
		"nurnberg":    "nbg1",
		"nbg1":        "nbg1",
		"falkenstein": "fsn1",
		"fsn1":        "fsn1",
		"helsinki":    "hel1",
		"hel1":        "hel1",
		"ashburn":     "ash",
		"ash":         "ash",
	}

	if id, ok := aliases[input]; ok {
		return id
	}

	// Default to Nuremberg
	if input == "" {
		return "nbg1"
	}

	return input
}

// runSteps executes all installation steps with a progress display.
// If resumeState is non-nil, steps already in resumeState.CompletedSteps are skipped.
// After each successful step the step name is persisted to resumeState (or a fresh state).
func runSteps(cfg *install.Config, resumeState *installstate.State) error {
	// Generate admin password before steps run so installZenithChart can pass it to Helm.
	// Cloud/Helm receives the password via a base64 values file (symbols OK).
	// Compose writes it into a .env, where '$' is interpolated — so the compose
	// path generates its own alphanumeric-safe secret in composeFetchStack.
	if cfg.AdminPassword == "" && cfg.Edition != "compose" {
		cfg.AdminPassword = install.GeneratePassword(16)
	}

	steps := install.GetInstallSteps(cfg)
	if cfg.Edition == "compose" {
		steps = install.GetComposeInstallSteps(cfg)
	}

	// Ensure we always have a state object for tracking progress.
	state := resumeState
	if state == nil {
		state = &installstate.State{
			Domain:     cfg.Domain,
			Provider:   string(cfg.MCProvider),
			Region:     cfg.Region,
			SSHKeyPath: cfg.SSHKeyPath,
		}
	}

	// Styles
	checkStyle := lipgloss.NewStyle().Foreground(tui.ColorSuccess)
	skipStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	spinStyle := lipgloss.NewStyle().Foreground(tui.ColorWarning)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError)
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	descStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	totalStart := time.Now()

	fmt.Println()
	for i, step := range steps {
		stepNum := fmt.Sprintf("[%d/%d]", i+1, len(steps))

		// Skip already-completed steps when resuming
		if installstate.IsStepComplete(state, step.Name) {
			fmt.Printf("  %s %s %s\n",
				skipStyle.Render("- "+stepNum),
				stepStyle.Render(step.Name),
				skipStyle.Render("(skipped — already completed)"),
			)
			continue
		}

		// Show spinner line
		fmt.Printf("  %s %s %s\n",
			spinStyle.Render("  "+stepNum),
			stepStyle.Render(step.Name),
			descStyle.Render("- "+step.Description),
		)

		start := time.Now()
		if err := step.Action(cfg); err != nil {
			// Move up and overwrite with error
			fmt.Printf("\033[1A\r  %s %s %s\n",
				errStyle.Render("x "+stepNum),
				stepStyle.Render(step.Name),
				errStyle.Render("- "+err.Error()),
			)
			return fmt.Errorf("step %d (%s) failed: %w", i+1, step.Name, err)
		}

		elapsed := time.Since(start)

		// Move up and overwrite with checkmark
		fmt.Printf("\033[1A\r  %s %s %s\n",
			checkStyle.Render("v "+stepNum),
			stepStyle.Render(step.Name),
			timeStyle.Render(fmt.Sprintf("(%s)", formatDuration(elapsed))),
		)

		// Persist completed step so --resume can skip it next time
		if err := installstate.MarkStepComplete(state, step.Name); err != nil {
			// Non-fatal: warn but do not abort the installation
			fmt.Printf("  %s\n", descStyle.Render(fmt.Sprintf("warning: failed to save step state: %v", err)))
		}
	}

	totalElapsed := time.Since(totalStart)
	fmt.Println()
	fmt.Printf("  %s %s\n",
		checkStyle.Render("v All steps completed"),
		timeStyle.Render(fmt.Sprintf("in %s", formatDuration(totalElapsed))),
	)

	return nil
}

// configFromState reconstructs a minimal install.Config from persisted state,
// used when --resume is passed without explicit flags.
func configFromState(s *installstate.State) *install.Config {
	cfg := &install.Config{
		Domain:     s.Domain,
		SSHKeyPath: s.SSHKeyPath,
		Region:     s.Region,
	}
	switch install.ServerProvider(s.Provider) {
	case install.ProviderHetzner:
		cfg.MCProvider = install.ProviderHetzner
	case install.ProviderExisting:
		cfg.MCProvider = install.ProviderExisting
		cfg.SSHHost = s.ServerIP
	default:
		cfg.MCProvider = install.ProviderHetzner
	}
	return cfg
}

// showResult displays the final success box with credentials and URLs.
// showComposeResult prints the "you're live" screen for the compose edition.
func showComposeResult(cfg *install.Config) {
	fmt.Println()
	title := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorSuccess).
		Render("FreeZenith is live!")

	dashURL := "http://localhost:3000"
	if cfg.Domain != "" && cfg.Domain != "localhost" {
		dashURL = "https://" + cfg.Domain
	}
	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@localhost"
	}

	labelStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	urlStyle := lipgloss.NewStyle().Foreground(tui.ColorPrimary).Bold(true).Underline(true)
	warnStyle := lipgloss.NewStyle().Foreground(tui.ColorWarning).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Dashboard:"), urlStyle.Render(dashURL)))
	content.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Email:"), valueStyle.Render(adminEmail)))
	content.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Password:"), warnStyle.Render(cfg.AdminPassword)))
	content.WriteString("\n")
	target := "this machine"
	if !cfg.ComposeLocal {
		target = cfg.SSHHost
	}
	content.WriteString(mutedStyle.Render(fmt.Sprintf("  Running on %s — manage with: zen status, zen logs, zen uninstall", target)))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorPrimary).
		Padding(1, 2).
		Render(content.String())
	fmt.Println(box)
}

func showResult(cfg *install.Config, result *install.InstallResult) {
	fmt.Println()

	// Success header
	successTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorSuccess).
		Render("Zenith installed successfully!")

	// Build the content
	var content strings.Builder

	content.WriteString(successTitle)
	content.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted).Width(20)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	urlStyle := lipgloss.NewStyle().Foreground(tui.ColorPrimary).Bold(true).Underline(true)
	warnStyle := lipgloss.NewStyle().Foreground(tui.ColorWarning).Bold(true)

	row := func(label, value string) {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render(label),
			valueStyle.Render(value),
		))
	}
	urlRow := func(label, value string) {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render(label),
			urlStyle.Render(value),
		))
	}

	content.WriteString("  URLS\n")
	urlRow("Mission Control:", result.MissionControlURL)
	urlRow("Cloud Platform:", result.CloudURL)
	content.WriteString("\n")

	content.WriteString("  CREDENTIALS\n")
	row("Username:", result.AdminUser)
	content.WriteString(fmt.Sprintf("  %s %s\n",
		labelStyle.Render("Password:"),
		warnStyle.Render(result.AdminPassword),
	))
	content.WriteString(fmt.Sprintf("  %s\n",
		warnStyle.Render("!! SAVE THIS PASSWORD NOW — it will not be stored anywhere !!"),
	))
	content.WriteString("\n")

	content.WriteString("  SERVER\n")
	row("IP Address:", result.ServerIP)
	if cfg.MCProvider == install.ProviderHetzner {
		if r := install.GetRegionByID(cfg.Region); r != nil {
			row("Region:", r.Name)
		}
	}

	if result.ClusterName != "" {
		content.WriteString("\n")
		content.WriteString("  FIRST CLUSTER\n")
		row("Name:", result.ClusterName)
		row("IP:", result.ClusterIP)
	}

	// Render the bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSuccess).
		Padding(1, 2).
		Render(content.String())
	fmt.Println(box)

	// DNS instructions if manual
	if cfg.DNSProvider == install.DNSManual {
		fmt.Println()
		dnsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tui.ColorWarning).
			Padding(1, 2).
			Render(fmt.Sprintf(
				"%s\n\n"+
					"  Add these DNS records to your domain registrar:\n\n"+
					"  %s  A  %s  ->  %s\n"+
					"  %s  A  %s  ->  %s\n",
				lipgloss.NewStyle().Bold(true).Foreground(tui.ColorWarning).Render("ACTION REQUIRED: Add DNS Records"),
				lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Type"),
				lipgloss.NewStyle().Foreground(tui.ColorText).Render(fmt.Sprintf("mission.%s", cfg.Domain)),
				lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render(result.ServerIP),
				lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Type"),
				lipgloss.NewStyle().Foreground(tui.ColorText).Render(fmt.Sprintf("cloud.%s", cfg.Domain)),
				lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render(result.ServerIP),
			))
		fmt.Println(dnsBox)
	}

	// Next steps
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).PaddingLeft(2).Render("Next steps:"))
	nextSteps := []string{
		fmt.Sprintf("Open Mission Control: %s", result.MissionControlURL),
		"Run 'zen status' to check platform health",
		"Run 'zen deploy' to deploy your first app",
	}
	for _, step := range nextSteps {
		fmt.Printf("  %s %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("->"),
			step)
	}
	fmt.Println()
}

// formatDuration produces a human-friendly duration string.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
