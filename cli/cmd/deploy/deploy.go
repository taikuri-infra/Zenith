package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagImage       string
	flagReplicas    int
	flagPort        int
	flagName        string
	flagEnvironment string
	flagNoWait      bool
)

var Cmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an application to Zenith",
	Long: `Deploy an application from the current directory or a pre-built Docker image.

Examples:
  zen deploy --image myrepo/myapp:latest   # Deploy pre-built image
  zen deploy --image myrepo/app:v1.2.3 --env staging
  zen deploy --name my-app                 # Deploy from current directory (requires --image for now)`,
	RunE: runDeploy,
}

func init() {
	Cmd.Flags().StringVar(&flagImage, "image", "", "Docker image URL to deploy (required)")
	Cmd.Flags().IntVar(&flagReplicas, "replicas", 1, "Number of replicas")
	Cmd.Flags().IntVar(&flagPort, "port", 0, "Container port")
	Cmd.Flags().StringVar(&flagName, "name", "", "App name on the platform")
	Cmd.Flags().StringVar(&flagEnvironment, "env", "production", "Target environment: staging or production")
	Cmd.Flags().BoolVar(&flagNoWait, "no-wait", false, "Don't wait for deployment to become healthy")
}

// ProjectType holds detected project information.
type ProjectType struct {
	Language   string
	Framework  string
	DetectedBy string
	Port       int
}

func runDeploy(cmd *cobra.Command, args []string) error {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Deploying to Zenith")
	fmt.Println(header)
	fmt.Println()

	if flagImage != "" {
		return deployImage()
	}

	return deployFromDirectory()
}

func deployFromDirectory() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Auto-detect project type
	detected := detectProject(cwd)
	if detected == nil {
		return fmt.Errorf("could not detect project type in %s\nAdd a Dockerfile or use --image flag", cwd)
	}

	appName := flagName
	if appName == "" {
		appName = filepath.Base(cwd)
	}
	appName = sanitizeName(appName)

	port := flagPort
	if port == 0 {
		port = detected.Port
	}

	// Display detection info
	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText)

	fmt.Printf("  %s %s\n", infoStyle.Render("Directory:"), valueStyle.Render(cwd))
	fmt.Printf("  %s %s\n", infoStyle.Render("Detected:"), valueStyle.Render(detected.Language))
	if detected.Framework != "" {
		fmt.Printf("  %s %s\n", infoStyle.Render("Framework:"), valueStyle.Render(detected.Framework))
	}
	fmt.Printf("  %s %s\n", infoStyle.Render("App name:"), valueStyle.Render(appName))
	fmt.Printf("  %s %d\n", infoStyle.Render("Port:"), port)
	fmt.Printf("  %s %d\n", infoStyle.Render("Replicas:"), flagReplicas)
	fmt.Println()

	// Building from source is not yet supported — use --image with a pre-built image.
	_ = port
	return fmt.Errorf("building from source is not yet supported\n\n  Build your image first, then run:\n  zen deploy --image <registry>/<image>:<tag> --name %s", appName)
}

func deployImage() error {
	appName := flagName
	if appName == "" {
		// Extract name from image (e.g. "myrepo/myapp:sha-abc" → "myapp")
		parts := strings.Split(flagImage, "/")
		last := parts[len(parts)-1]
		appName = strings.Split(last, ":")[0]
	}
	appName = sanitizeName(appName)

	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText)

	fmt.Printf("  %s %s\n", infoStyle.Render("Image:"), valueStyle.Render(flagImage))
	fmt.Printf("  %s %s\n", infoStyle.Render("App name:"), valueStyle.Render(appName))
	fmt.Printf("  %s %s\n", infoStyle.Render("Environment:"), valueStyle.Render(flagEnvironment))
	fmt.Println()

	return runDeploySteps(appName, flagImage)
}

func runDeploySteps(appName, image string) error {
	checkmark := lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("✓")
	errorMark := lipgloss.NewStyle().Foreground(tui.ColorError).Render("✗")
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	muteStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	// Load config and authenticate
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Token == "" {
		return fmt.Errorf("not logged in — run: zen login")
	}

	client := cliapi.NewClient(cfg.APIEndpoint, cfg.Token)

	// Step 1: Trigger deploy
	totalStart := time.Now()
	fmt.Printf("  %s %s\n", muteStyle.Render("→"), stepStyle.Render("Triggering deploy..."))

	start := time.Now()
	result, err := client.Deploy(appName, image, flagEnvironment, flagReplicas)
	if err != nil {
		fmt.Printf("  %s %s\n", errorMark, err.Error())
		return fmt.Errorf("deploy failed: %w", err)
	}
	fmt.Printf("  %s %s %s\n", checkmark, stepStyle.Render("Deploy triggered"),
		timeStyle.Render(fmt.Sprintf("(%s)", time.Since(start).Round(time.Millisecond))))
	fmt.Printf("  %s deployment ID: %s\n", muteStyle.Render("→"), result.DeploymentID)

	if flagNoWait {
		fmt.Printf("\n  %s Deploy accepted. Check status with: zen status\n", muteStyle.Render("→"))
		return nil
	}

	// Step 2: Poll for healthy status
	fmt.Printf("  %s %s\n", muteStyle.Render("→"), stepStyle.Render("Waiting for deployment..."))
	start = time.Now()

	const maxAttempts = 36 // 36 × 5s = 3 min
	var finalStatus string
	var finalURL string
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(5 * time.Second)
		status, err := client.GetDeploymentStatus(result.DeploymentID)
		if err != nil {
			continue // transient error, keep polling
		}
		finalURL = status.URL
		switch status.Status {
		case "active", "running", "healthy":
			finalStatus = status.Status
			goto done
		case "failed", "error", "crash_loop":
			fmt.Printf("  %s %s\n", errorMark, stepStyle.Render("Deployment failed: "+status.Status))
			return fmt.Errorf("deployment %s: status=%s", result.DeploymentID, status.Status)
		default:
			fmt.Printf("  %s status: %s (%ds/%ds)\n",
				muteStyle.Render("·"),
				status.Status,
				int(time.Since(start).Seconds()),
				maxAttempts*5,
			)
		}
	}
	fmt.Printf("  %s %s\n", errorMark, "Timed out waiting for deployment (3 min)")
	fmt.Printf("  %s Check status with: zen status\n", muteStyle.Render("→"))
	return nil

done:
	elapsed := time.Since(start)
	fmt.Printf("  %s %s %s\n", checkmark, stepStyle.Render("Deployment "+finalStatus),
		timeStyle.Render(fmt.Sprintf("(%s)", elapsed.Round(time.Second))))

	totalElapsed := time.Since(totalStart)

	// Success box
	fmt.Println()
	successLines := fmt.Sprintf(
		"%s\n\n  %s %s\n  %s %s\n  %s %s",
		lipgloss.NewStyle().Bold(true).Foreground(tui.ColorSuccess).Render("Deployed successfully!"),
		muteStyle.Render("App:"), appName,
		muteStyle.Render("Time:"), totalElapsed.Round(time.Second).String(),
		muteStyle.Render("URL:"), finalURL,
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSuccess).
		Padding(1, 2).
		Render(successLines)
	fmt.Println(box)

	fmt.Println()
	fmt.Printf("  %s zen logs %s\n", muteStyle.Render("→"), appName)
	fmt.Printf("  %s zen status\n", muteStyle.Render("→"))
	fmt.Println()

	return nil
}

// detectProject auto-detects the project type in a directory.
func detectProject(dir string) *ProjectType {
	checks := []struct {
		file      string
		language  string
		framework string
		port      int
	}{
		{"Dockerfile", "Docker", "", 8080},
		{"go.mod", "Go", "", 8080},
		{"package.json", "Node.js", "", 3000},
		{"requirements.txt", "Python", "", 5000},
		{"Pipfile", "Python", "Pipenv", 5000},
		{"pyproject.toml", "Python", "", 5000},
		{"Gemfile", "Ruby", "Rails", 3000},
		{"pom.xml", "Java", "Maven", 8080},
		{"build.gradle", "Java", "Gradle", 8080},
		{"Cargo.toml", "Rust", "", 8080},
		{"mix.exs", "Elixir", "Mix", 4000},
		{"composer.json", "PHP", "", 8080},
	}

	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(dir, check.file)); err == nil {
			pt := &ProjectType{
				Language:   check.language,
				Framework:  check.framework,
				DetectedBy: check.file,
				Port:       check.port,
			}

			// Refine detection for Node.js
			if check.file == "package.json" {
				if _, err := os.Stat(filepath.Join(dir, "next.config.js")); err == nil {
					pt.Framework = "Next.js"
					pt.Port = 3000
				} else if _, err := os.Stat(filepath.Join(dir, "next.config.mjs")); err == nil {
					pt.Framework = "Next.js"
					pt.Port = 3000
				} else if _, err := os.Stat(filepath.Join(dir, "nuxt.config.ts")); err == nil {
					pt.Framework = "Nuxt"
					pt.Port = 3000
				} else if _, err := os.Stat(filepath.Join(dir, "vite.config.ts")); err == nil {
					pt.Framework = "Vite"
					pt.Port = 5173
				}
			}

			return pt
		}
	}

	return nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
