package install

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/tui"
)

// WizardResult holds all values collected from the interactive wizard.
type WizardResult struct {
	Config    *Config
	Confirmed bool
}

// RunWizard runs the interactive TUI installation wizard using huh forms.
// It walks the user through 5 steps and returns a fully populated Config.
func RunWizard() (*WizardResult, error) {
	cfg := &Config{
		MCProvider:  ProviderHetzner,
		ServerType:  "cx22",
		Region:      "nbg1",
		SSHUser:     "root",
		DNSProvider: DNSCloudflare,
		WithCluster: false,
		ClusterProvider:   ProviderHetzner,
		ClusterServerType: "cx22",
		ClusterRegion:     "nbg1",
		ClusterSSHUser:    "root",
	}

	// --- Step 1: Mission Control Server ---
	var mcProviderStr string = "hetzner"

	step1Provider := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 1 of 5: Mission Control Server").
				Description("Where should we install Mission Control?"),
			huh.NewSelect[string]().
				Title("Server source").
				Options(
					huh.NewOption("Hetzner Cloud (we'll create a server)", "hetzner"),
					huh.NewOption("Existing server (I have one)", "existing"),
				).
				Value(&mcProviderStr),
		),
	).WithTheme(zenithTheme())

	if err := step1Provider.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}
	cfg.MCProvider = ServerProvider(mcProviderStr)

	// Step 1b: Provider-specific fields
	if cfg.MCProvider == ProviderHetzner {
		if err := runHetznerForm(
			"Mission Control",
			&cfg.HetznerToken,
			&cfg.Region,
			&cfg.ServerType,
		); err != nil {
			return nil, err
		}
	} else {
		if err := runExistingServerForm("Mission Control", &cfg.SSHHost, &cfg.SSHUser); err != nil {
			return nil, err
		}
	}

	// --- Step 2: Domain ---
	step2 := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 2 of 5: Domain").
				Description("Your domain will be used for:\n  mission.{domain} - Mission Control\n  cloud.{domain}   - User Platform"),
			huh.NewInput().
				Title("Your domain").
				Placeholder("embermind.app").
				Value(&cfg.Domain).
				Validate(func(s string) error {
					return ValidateDomain(s)
				}),
		),
	).WithTheme(zenithTheme())

	if err := step2.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}

	// --- Step 3: DNS ---
	var dnsProviderStr string = "cloudflare"

	step3Provider := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 3 of 5: DNS Configuration").
				Description("How should we manage DNS records?"),
			huh.NewSelect[string]().
				Title("DNS provider").
				Options(
					huh.NewOption("Cloudflare (automatic)", "cloudflare"),
					huh.NewOption("I'll add DNS records manually", "manual"),
				).
				Value(&dnsProviderStr),
		),
	).WithTheme(zenithTheme())

	if err := step3Provider.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}
	cfg.DNSProvider = DNSProvider(dnsProviderStr)

	if cfg.DNSProvider == DNSCloudflare {
		step3Token := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Cloudflare API Token").
					Description("Create one at dash.cloudflare.com -> My Profile -> API Tokens").
					Placeholder("cf_xxxxxxxxxxxx").
					Value(&cfg.CloudflareToken).
					Validate(func(s string) error {
						if len(s) < 10 {
							return fmt.Errorf("token is too short")
						}
						return nil
					}),
			),
		).WithTheme(zenithTheme())

		if err := step3Token.Run(); err != nil {
			return nil, fmt.Errorf("wizard cancelled")
		}
	}

	// --- Step 4: First Cluster (optional) ---
	step4Confirm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 4 of 5: First Cluster (Optional)").
				Description("You can create your first tenant cluster immediately,\nor do it later from Mission Control."),
			huh.NewConfirm().
				Title("Create your first cluster now?").
				Affirmative("Yes, set it up").
				Negative("No, I'll do it later").
				Value(&cfg.WithCluster),
		),
	).WithTheme(zenithTheme())

	if err := step4Confirm.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}

	if cfg.WithCluster {
		var clusterProviderStr string = "hetzner"

		step4Provider := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Cluster server source").
					Options(
						huh.NewOption("Hetzner Cloud (we'll create a server)", "hetzner"),
						huh.NewOption("Existing server (I have one)", "existing"),
					).
					Value(&clusterProviderStr),
			),
		).WithTheme(zenithTheme())

		if err := step4Provider.Run(); err != nil {
			return nil, fmt.Errorf("wizard cancelled")
		}
		cfg.ClusterProvider = ServerProvider(clusterProviderStr)

		if cfg.ClusterProvider == ProviderHetzner {
			// Reuse the MC Hetzner token by default
			cfg.ClusterHetznerToken = cfg.HetznerToken
			if err := runHetznerForm(
				"Cluster",
				&cfg.ClusterHetznerToken,
				&cfg.ClusterRegion,
				&cfg.ClusterServerType,
			); err != nil {
				return nil, err
			}
		} else {
			if err := runExistingServerForm("Cluster", &cfg.ClusterSSHHost, &cfg.ClusterSSHUser); err != nil {
				return nil, err
			}
		}
	}

	// --- Step 5: Summary + Pricing ---
	summary := buildSummary(cfg)
	var confirmed bool

	step5 := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 5 of 5: Summary").
				Description(summary),
			huh.NewConfirm().
				Title("Proceed with installation?").
				Affirmative("Install").
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithTheme(zenithTheme())

	if err := step5.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}

	return &WizardResult{
		Config:    cfg,
		Confirmed: confirmed,
	}, nil
}

// runHetznerForm shows the Hetzner-specific inputs: token, region, server type.
func runHetznerForm(label string, token *string, region *string, serverType *string) error {
	regionOptions := buildRegionOptions()
	serverTypeOptions := buildServerTypeOptions()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("%s - Hetzner API Token", label)).
				Description("Create one at console.hetzner.cloud -> API Tokens").
				Placeholder("hc_xxxxxxxxxxxx").
				Value(token).
				Validate(func(s string) error {
					return ValidateToken(s)
				}),
			huh.NewSelect[string]().
				Title("Region").
				Options(regionOptions...).
				Value(region),
			huh.NewSelect[string]().
				Title("Server type").
				Options(serverTypeOptions...).
				Value(serverType),
		),
	).WithTheme(zenithTheme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("wizard cancelled")
	}
	return nil
}

// runExistingServerForm shows the inputs for an existing server: host, user.
func runExistingServerForm(label string, host *string, user *string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("%s - SSH Host / IP", label)).
				Placeholder("203.0.113.10").
				Value(host).
				Validate(func(s string) error {
					if len(strings.TrimSpace(s)) == 0 {
						return fmt.Errorf("host is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("SSH User").
				Placeholder("root").
				Value(user),
		),
	).WithTheme(zenithTheme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("wizard cancelled")
	}
	return nil
}

// buildRegionOptions creates huh.Option entries for the region selector.
func buildRegionOptions() []huh.Option[string] {
	opts := make([]huh.Option[string], len(Regions))
	for i, r := range Regions {
		label := fmt.Sprintf("%s (%s, %s)", r.Name, r.ID, r.Country)
		opts[i] = huh.NewOption(label, r.ID)
	}
	return opts
}

// buildServerTypeOptions creates huh.Option entries for the server type selector.
func buildServerTypeOptions() []huh.Option[string] {
	opts := make([]huh.Option[string], len(ServerTypes))
	for i, s := range ServerTypes {
		label := fmt.Sprintf("%s - %s  %s/mo", s.Name, s.Description, formatEuro(s.Price))
		opts[i] = huh.NewOption(label, s.ID)
	}
	return opts
}

// buildSummary renders a text summary table of all installation choices.
func buildSummary(cfg *Config) string {
	var b strings.Builder
	divider := strings.Repeat("-", 50)

	labelStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	priceStyle := lipgloss.NewStyle().Foreground(tui.ColorPrimary).Bold(true)

	row := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			labelStyle.Render(fmt.Sprintf("%-22s", label)),
			valueStyle.Render(value),
		))
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).Render("  Mission Control"))
	b.WriteString("\n  " + divider + "\n")

	if cfg.MCProvider == ProviderHetzner {
		row("Provider:", "Hetzner Cloud")
		if r := GetRegionByID(cfg.Region); r != nil {
			row("Region:", fmt.Sprintf("%s (%s)", r.Name, r.ID))
		} else {
			row("Region:", cfg.Region)
		}
		if st := GetServerTypeByID(cfg.ServerType); st != nil {
			row("Server:", fmt.Sprintf("%s - %s", st.Name, st.Description))
			row("Cost:", fmt.Sprintf("%s/mo", formatEuro(st.Price)))
		}
	} else {
		row("Provider:", "Existing Server")
		row("SSH Host:", cfg.SSHHost)
		row("SSH User:", cfg.SSHUser)
		row("Cost:", formatEuro(0))
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).Render("  Domain & DNS"))
	b.WriteString("\n  " + divider + "\n")
	row("Domain:", cfg.Domain)
	row("Mission Control:", fmt.Sprintf("mission.%s", cfg.Domain))
	row("Cloud Platform:", fmt.Sprintf("cloud.%s", cfg.Domain))
	if cfg.DNSProvider == DNSCloudflare {
		row("DNS:", "Cloudflare (automatic)")
	} else {
		row("DNS:", "Manual")
	}

	if cfg.WithCluster {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).Render("  First Cluster"))
		b.WriteString("\n  " + divider + "\n")

		if cfg.ClusterProvider == ProviderHetzner {
			row("Provider:", "Hetzner Cloud")
			if r := GetRegionByID(cfg.ClusterRegion); r != nil {
				row("Region:", fmt.Sprintf("%s (%s)", r.Name, r.ID))
			}
			if st := GetServerTypeByID(cfg.ClusterServerType); st != nil {
				row("Server:", fmt.Sprintf("%s - %s", st.Name, st.Description))
				row("Cost:", fmt.Sprintf("%s/mo", formatEuro(st.Price)))
			}
		} else {
			row("Provider:", "Existing Server")
			row("SSH Host:", cfg.ClusterSSHHost)
			row("SSH User:", cfg.ClusterSSHUser)
			row("Cost:", formatEuro(0))
		}
	} else {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorMuted).Render("  First Cluster"))
		b.WriteString("\n  " + divider + "\n")
		row("Status:", "Skipped (create later from Mission Control)")
	}

	// Total pricing
	total := EstimateMonthlyCost(cfg)
	b.WriteString("\n  " + divider + "\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		labelStyle.Render(fmt.Sprintf("%-22s", "Estimated total:")),
		priceStyle.Render(fmt.Sprintf("%s/mo", formatEuro(total))),
	))

	if total == 0 {
		b.WriteString(fmt.Sprintf("  %s\n",
			labelStyle.Render("(using existing servers - no Hetzner costs)"),
		))
	}

	return b.String()
}

// formatEuro formats a price as a Euro currency string.
func formatEuro(price float64) string {
	if price == 0 {
		return "FREE"
	}
	return fmt.Sprintf("\u20ac%.2f", price)
}

// zenithTheme returns a huh theme that matches the Zenith design system.
func zenithTheme() *huh.Theme {
	t := huh.ThemeDracula()

	// Override key colors to match our emerald accent
	t.Focused.Title = t.Focused.Title.Foreground(tui.ColorPrimary)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(tui.ColorPrimary)
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Background(tui.ColorPrimary).
		Foreground(lipgloss.Color("#000000"))
	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Foreground(tui.ColorMuted)
	t.Focused.NoteTitle = t.Focused.NoteTitle.
		Foreground(tui.ColorPrimary).
		Bold(true)

	return t
}
