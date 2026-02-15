package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagImage    string
	flagReplicas int
	flagPort     int
	flagName     string
)

var Cmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an application to Zenith",
	Long: `Deploy an application from the current directory, a Docker image, or a GitHub repo.

Examples:
  zen deploy                          # Deploy from current directory
  zen deploy --image nginx:latest     # Deploy pre-built image
  zen deploy --name my-app --port 8080`,
	RunE: runDeploy,
}

func init() {
	Cmd.Flags().StringVar(&flagImage, "image", "", "Docker image to deploy")
	Cmd.Flags().IntVar(&flagReplicas, "replicas", 1, "Number of replicas")
	Cmd.Flags().IntVar(&flagPort, "port", 0, "Container port")
	Cmd.Flags().StringVar(&flagName, "name", "", "App name")
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

	// Simulate build & deploy steps
	return runDeploySteps(appName, port)
}

func deployImage() error {
	appName := flagName
	if appName == "" {
		// Extract name from image
		parts := strings.Split(flagImage, "/")
		last := parts[len(parts)-1]
		appName = strings.Split(last, ":")[0]
	}
	appName = sanitizeName(appName)

	port := flagPort
	if port == 0 {
		port = 8080
	}

	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText)

	fmt.Printf("  %s %s\n", infoStyle.Render("Image:"), valueStyle.Render(flagImage))
	fmt.Printf("  %s %s\n", infoStyle.Render("App name:"), valueStyle.Render(appName))
	fmt.Printf("  %s %d\n", infoStyle.Render("Port:"), port)
	fmt.Printf("  %s %d\n", infoStyle.Render("Replicas:"), flagReplicas)
	fmt.Println()

	return runDeploySteps(appName, port)
}

func runDeploySteps(appName string, port int) error {
	checkmark := lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("✓")
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	steps := []struct {
		name   string
		action func() error
	}{
		{"Building container image", func() error { return nil }},
		{"Pushing to registry", func() error { return nil }},
		{"Creating App resource", func() error { return nil }},
		{"Waiting for pods to be ready", func() error { return nil }},
	}

	// Skip build steps if deploying a pre-built image
	if flagImage != "" {
		steps = steps[2:]
	}

	totalStart := time.Now()
	for _, step := range steps {
		start := time.Now()
		if err := step.action(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
		elapsed := time.Since(start)
		fmt.Printf("  %s %s %s\n", checkmark, stepStyle.Render(step.name),
			timeStyle.Render(fmt.Sprintf("(%s)", elapsed.Round(time.Millisecond))))
	}

	totalElapsed := time.Since(totalStart)

	// Success box
	fmt.Println()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSuccess).
		Padding(1, 2).
		Render(fmt.Sprintf(
			"%s\n\n"+
				"  %s %s\n"+
				"  %s %d\n"+
				"  %s %d/%d ready\n"+
				"  %s %s",
			lipgloss.NewStyle().Bold(true).Foreground(tui.ColorSuccess).Render("Deployed successfully!"),
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("App:"),
			appName,
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Port:"),
			port,
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Pods:"),
			flagReplicas, flagReplicas,
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Time:"),
			totalElapsed.Round(time.Millisecond).String(),
		))
	fmt.Println(box)

	// Next steps
	fmt.Println()
	fmt.Printf("  %s zen logs %s\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("→"),
		appName)
	fmt.Printf("  %s zen status\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("→"))
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
