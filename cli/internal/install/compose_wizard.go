package install

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// RunInteractive asks which edition to install, then runs the matching wizard.
// This is the entry point for a bare `zen install` with no flags.
func RunInteractive() (*WizardResult, error) {
	edition := "compose"
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("FreeZenith Installer").
				Description("How do you want to run FreeZenith?"),
			huh.NewSelect[string]().
				Title("Edition").
				Options(
					huh.NewOption("Self-host on any Linux box (Docker, no Kubernetes)", "compose"),
					huh.NewOption("Managed cloud (Hetzner + Kubernetes)", "cloud"),
				).
				Value(&edition),
		),
	).WithTheme(zenithTheme())
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}
	if edition == "cloud" {
		return RunWizard()
	}
	return RunComposeWizard()
}

// RunComposeWizard collects settings for a self-host (compose) install: where to
// install and how the domain / HTTPS should work.
func RunComposeWizard() (*WizardResult, error) {
	cfg := &Config{Edition: "compose", SSHUser: "root"}

	// --- Step 1: target ---
	target := "local"
	targetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 1 of 3: Where to install").
				Description("On this machine, or on a remote server over SSH?"),
			huh.NewSelect[string]().
				Title("Target").
				Options(
					huh.NewOption("This machine", "local"),
					huh.NewOption("A remote server (SSH)", "ssh"),
				).
				Value(&target),
		),
	).WithTheme(zenithTheme())
	if err := targetForm.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}
	cfg.ComposeLocal = target == "local"
	if !cfg.ComposeLocal {
		sshForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("SSH host / IP").
					Value(&cfg.SSHHost).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("host is required")
						}
						return nil
					}),
				huh.NewInput().Title("SSH user").Placeholder("root").Value(&cfg.SSHUser),
			),
		).WithTheme(zenithTheme())
		if err := sshForm.Run(); err != nil {
			return nil, fmt.Errorf("wizard cancelled")
		}
		if strings.TrimSpace(cfg.SSHUser) == "" {
			cfg.SSHUser = "root"
		}
	}

	// --- Step 2: domain / HTTPS ---
	choice := "free"
	domainForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Step 2 of 3: Domain & HTTPS").
				Description("How should people reach your FreeZenith?"),
			huh.NewSelect[string]().
				Title("Domain").
				Options(
					huh.NewOption("Free subdomain + automatic HTTPS (<slug>.apps.freezenith.com)", "free"),
					huh.NewOption("My own domain", "own"),
					huh.NewOption("Local only (http://localhost:3000)", "local"),
				).
				Value(&choice),
		),
	).WithTheme(zenithTheme())
	if err := domainForm.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}

	switch choice {
	case "free":
		cfg.FreeSubdomain = true
	case "local":
		cfg.Domain = "localhost"
	case "own":
		useCF := false
		ownForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Your domain").
					Placeholder("app.example.com").
					Value(&cfg.Domain).
					Validate(ValidateDomain),
				huh.NewInput().
					Title("Email for the HTTPS certificate").
					Placeholder("you@example.com").
					Value(&cfg.AdminEmail),
				huh.NewConfirm().
					Title("Is your domain's DNS on Cloudflare?").
					Description("If yes, I can create the DNS record for you; otherwise I'll show you the record to add.").
					Value(&useCF),
			),
		).WithTheme(zenithTheme())
		if err := ownForm.Run(); err != nil {
			return nil, fmt.Errorf("wizard cancelled")
		}
		if useCF {
			cfg.DNSProvider = DNSCloudflare
			tokenForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Cloudflare API token").
						Description("DNS:Edit on your own zone. Create at dash.cloudflare.com -> My Profile -> API Tokens").
						Value(&cfg.CloudflareToken).
						Validate(func(s string) error {
							if len(strings.TrimSpace(s)) < 10 {
								return fmt.Errorf("token is too short")
							}
							return nil
						}),
				),
			).WithTheme(zenithTheme())
			if err := tokenForm.Run(); err != nil {
				return nil, fmt.Errorf("wizard cancelled")
			}
		}
	}

	// --- Step 3: review + confirm ---
	confirmed := false
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Step 3 of 3: Review").Description(buildComposeSummary(cfg)),
			huh.NewConfirm().Title("Install now?").Value(&confirmed),
		),
	).WithTheme(zenithTheme())
	if err := confirmForm.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled")
	}

	return &WizardResult{Config: cfg, Confirmed: confirmed}, nil
}

func buildComposeSummary(cfg *Config) string {
	var b strings.Builder
	if cfg.ComposeLocal {
		b.WriteString("Target:  this machine\n")
	} else {
		fmt.Fprintf(&b, "Target:  %s@%s\n", cfg.SSHUser, cfg.SSHHost)
	}
	switch {
	case cfg.FreeSubdomain:
		b.WriteString("Domain:  free <slug>.apps.freezenith.com (automatic HTTPS)")
	case cfg.Domain == "" || cfg.Domain == "localhost":
		b.WriteString("Domain:  localhost (http, no certificate)")
	default:
		fmt.Fprintf(&b, "Domain:  %s", cfg.Domain)
		if cfg.DNSProvider == DNSCloudflare {
			b.WriteString(" (DNS record created via Cloudflare)")
		} else {
			b.WriteString(" (you'll add one DNS record)")
		}
	}
	return b.String()
}
