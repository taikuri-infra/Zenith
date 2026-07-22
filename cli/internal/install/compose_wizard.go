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
		if err := runOwnDomainForms(cfg); err != nil {
			return nil, err
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

// runOwnDomainForms handles the "my own domain" branch: it first asks how the
// server is reached (which decides the certificate strategy), then collects the
// domain and whatever that strategy needs. The option copy is written so someone
// who has never heard of NAT or a certificate authority can still choose right.
func runOwnDomainForms(cfg *Config) error {
	tlsChoice := TLSHTTP01
	tlsForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("How do browsers reach this server?").
				Description("This decides how we get an HTTPS certificate. Pick the line that matches your server."),
			huh.NewSelect[string]().
				Title("HTTPS").
				Options(
					huh.NewOption("It's on the public internet (a cloud VPS with a public IP) — free real certificate", TLSHTTP01),
					huh.NewOption("It's behind my home/office network, but its domain's DNS is on Cloudflare — free real certificate", TLSDNS01),
					huh.NewOption("It's an internal or offline server, e.g. zenith.lan — self-signed certificate", TLSSelfSigned),
				).
				Value(&tlsChoice),
		),
	).WithTheme(zenithTheme())
	if err := tlsForm.Run(); err != nil {
		return fmt.Errorf("wizard cancelled")
	}
	cfg.TLSMode = tlsChoice

	switch tlsChoice {
	case TLSSelfSigned:
		// Internal name: no public CA, no email, no DNS token — just the name.
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Internal domain").
					Description("Use the name this server answers to on your network (e.g. zenith.lan).\nAfter install we'll give you one file to trust, once, on each machine."),
				huh.NewInput().
					Title("Internal domain").
					Placeholder("zenith.lan").
					Value(&cfg.Domain).
					Validate(ValidateDomain),
			),
		).WithTheme(zenithTheme())
		if err := form.Run(); err != nil {
			return fmt.Errorf("wizard cancelled")
		}
		return nil

	case TLSDNS01:
		// Behind NAT with a real domain: a Cloudflare token proves DNS control so
		// Let's Encrypt can still issue a real certificate.
		form := huh.NewForm(
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
				huh.NewInput().
					Title("Cloudflare API token").
					Description("DNS:Edit on your own zone. We use it to answer the certificate challenge — the server never needs to be reachable from the internet.").
					Value(&cfg.CloudflareToken).
					Validate(minTokenLen),
			),
		).WithTheme(zenithTheme())
		if err := form.Run(); err != nil {
			return fmt.Errorf("wizard cancelled")
		}
		cfg.DNSProvider = DNSCloudflare
		return nil

	default: // TLSHTTP01
		useCF := false
		form := huh.NewForm(
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
		if err := form.Run(); err != nil {
			return fmt.Errorf("wizard cancelled")
		}
		if useCF {
			cfg.DNSProvider = DNSCloudflare
			tokenForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Cloudflare API token").
						Description("DNS:Edit on your own zone. Create at dash.cloudflare.com -> My Profile -> API Tokens").
						Value(&cfg.CloudflareToken).
						Validate(minTokenLen),
				),
			).WithTheme(zenithTheme())
			if err := tokenForm.Run(); err != nil {
				return fmt.Errorf("wizard cancelled")
			}
		}
		return nil
	}
}

func minTokenLen(s string) error {
	if len(strings.TrimSpace(s)) < 10 {
		return fmt.Errorf("token is too short")
	}
	return nil
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
		fmt.Fprintf(&b, "Domain:  %s\n", cfg.Domain)
		switch tlsMode(cfg) {
		case TLSSelfSigned:
			b.WriteString("HTTPS:   self-signed certificate (you'll trust one file per machine)")
		case TLSDNS01:
			b.WriteString("HTTPS:   Let's Encrypt via Cloudflare DNS (works behind NAT)")
		default:
			if cfg.DNSProvider == DNSCloudflare {
				b.WriteString("HTTPS:   Let's Encrypt (DNS record created via Cloudflare)")
			} else {
				b.WriteString("HTTPS:   Let's Encrypt (you'll add one DNS record)")
			}
		}
	}
	return b.String()
}
