package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Cmd is the dev command.
var Cmd = &cobra.Command{
	Use:   "dev",
	Short: "Connect local dev environment to Zenith Cloud services",
	Long: `Start a local development session connected to your Zenith Cloud staging services.

This command:
  1. Reads docker-compose.yml from the current directory
  2. Finds your project on Zenith Cloud
  3. Fetches staging environment managed services (Postgres, Redis, etc.)
  4. Writes connection strings to .env.zenith
  5. Adds .env.zenith to .gitignore

Then run your app normally (npm run dev, go run ., etc.) and it will
connect to the remote staging services automatically.`,
	RunE: runDev,
}

var flagProject string

func init() {
	Cmd.Flags().StringVar(&flagProject, "project", "", "Project name or ID (auto-detected from docker-compose.yml if omitted)")
}

// composeFile represents a minimal docker-compose.yml structure.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string            `yaml:"image"`
	Build       interface{}       `yaml:"build"`
	Ports       []string          `yaml:"ports"`
	Environment interface{}       `yaml:"environment"`
	DependsOn   interface{}       `yaml:"depends_on"`
}

// managedServiceInfo holds connection details for a managed service.
type managedServiceInfo struct {
	Name          string `json:"name"`
	ServiceType   string `json:"service_type"`
	ConnectionURL string `json:"connection_url"`
	InternalHost  string `json:"internal_host"`
	Port          int    `json:"port"`
	Status        string `json:"status"`
}

// environmentInfo holds environment details.
type environmentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func runDev(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config — run 'zen login' first: %w", err)
	}
	if cfg.Token == "" {
		return fmt.Errorf("not logged in — run 'zen login' first")
	}

	client := api.NewClient(cfg.APIEndpoint, cfg.Token)

	// Step 1: Find docker-compose.yml
	composePath := findComposeFile()
	if composePath == "" {
		fmt.Printf("%s No docker-compose.yml found in current directory\n", tui.Red("✗"))
		return fmt.Errorf("docker-compose.yml not found")
	}
	fmt.Printf("%s Found %s\n", tui.Green("✓"), filepath.Base(composePath))

	// Step 2: Parse to detect project name
	composeData, err := os.ReadFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose composeFile
	if err := yaml.Unmarshal(composeData, &compose); err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Step 3: Find project on Zenith
	projectID := flagProject
	if projectID == "" {
		// Try .zenith.yaml first
		projectID = readZenithConfig()
	}
	if projectID == "" {
		projectID = cfg.Project
	}

	if projectID == "" {
		// Try to match by directory name
		cwd, _ := os.Getwd()
		dirName := filepath.Base(cwd)
		projects, err := client.ListProjects()
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}
		for _, p := range projects {
			if strings.EqualFold(p.Name, dirName) || p.ID == dirName {
				projectID = p.ID
				break
			}
		}
		if projectID == "" && len(projects) > 0 {
			fmt.Printf("\n%s Could not auto-detect project. Available projects:\n", tui.Yellow("!"))
			for _, p := range projects {
				fmt.Printf("  - %s (%s)\n", p.Name, p.ID)
			}
			return fmt.Errorf("specify project with --project flag")
		}
	}

	if projectID == "" {
		return fmt.Errorf("no projects found — create one at your Zenith dashboard first")
	}

	fmt.Printf("%s Found project on Zenith Cloud\n", tui.Green("✓"))

	// Step 4: Get dev info (environments + managed services)
	var devInfo struct {
		ProjectName     string               `json:"project_name"`
		ManagedServices []managedServiceInfo  `json:"managed_services"`
		Environments    []environmentInfo     `json:"environments"`
	}
	if err := client.DoRaw("GET", fmt.Sprintf("/api/v1/projects/%s/dev-info", projectID), nil, &devInfo); err != nil {
		return fmt.Errorf("failed to fetch dev info: %w", err)
	}

	// Show environment info
	for _, env := range devInfo.Environments {
		if env.Name == "staging" {
			fmt.Printf("%s Staging environment found\n", tui.Green("✓"))
			break
		}
	}

	msResp := struct{ Items []managedServiceInfo }{ Items: devInfo.ManagedServices }

	if len(msResp.Items) == 0 {
		fmt.Printf("%s No managed services found in this project\n", tui.Yellow("!"))
		fmt.Println("  Add services via your Zenith dashboard or docker-compose import")
		return nil
	}

	// Step 5: Generate .env.zenith
	fmt.Printf("\n%s Managed services:\n", tui.Cyan("→"))
	envVars := map[string]string{}

	for _, ms := range msResp.Items {
		if ms.Status != "ready" {
			fmt.Printf("  %s %s (%s) — %s\n", tui.Yellow("⏳"), ms.Name, ms.ServiceType, ms.Status)
			continue
		}

		fmt.Printf("  %s %s (%s)\n", tui.Green("✓"), ms.Name, ms.ServiceType)

		// Map service to standard env vars
		switch ms.ServiceType {
		case "postgresql":
			envVars["DATABASE_URL"] = ms.ConnectionURL
			envVars["PGHOST"] = ms.InternalHost
			envVars["PGPORT"] = fmt.Sprintf("%d", ms.Port)
		case "redis":
			envVars["REDIS_URL"] = ms.ConnectionURL
		case "mysql":
			envVars["MYSQL_URL"] = ms.ConnectionURL
		case "mongodb":
			envVars["MONGODB_URL"] = ms.ConnectionURL
		case "rabbitmq":
			envVars["RABBITMQ_URL"] = ms.ConnectionURL
		}
	}

	if len(envVars) == 0 {
		fmt.Printf("\n%s No ready services to connect to\n", tui.Yellow("!"))
		return nil
	}

	// Write .env.zenith
	envFile := ".env.zenith"
	f, err := os.Create(envFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", envFile, err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# Generated by zen dev — DO NOT COMMIT\n")
	fmt.Fprintf(f, "# Project: %s\n\n", projectID)
	for k, v := range envVars {
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}

	fmt.Printf("\n%s Environment variables written to %s\n", tui.Green("✓"), envFile)

	// Add to .gitignore
	addToGitignore(envFile)

	// Print summary
	fmt.Printf("\n%s Ready! Run your app:\n", tui.Green("✓"))
	fmt.Printf("  source .env.zenith && npm run dev\n")
	fmt.Printf("  # or: source .env.zenith && go run .\n")
	fmt.Printf("  # or: set -a && source .env.zenith && python manage.py runserver\n")

	return nil
}

// findComposeFile looks for docker-compose.yml variants in the current directory.
func findComposeFile() string {
	names := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, name := range names {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

// readZenithConfig reads project_id from .zenith.yaml if it exists.
func readZenithConfig() string {
	data, err := os.ReadFile(".zenith.yaml")
	if err != nil {
		return ""
	}
	var cfg struct {
		ProjectID string `yaml:"project_id"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.ProjectID
}

// addToGitignore adds a file to .gitignore if not already present.
func addToGitignore(filename string) {
	gitignorePath := ".gitignore"
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		if strings.Contains(string(data), filename) {
			return // Already in .gitignore
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		f.WriteString("\n")
	}
	f.WriteString(filename + "\n")
	fmt.Printf("%s Added %s to .gitignore\n", tui.Green("✓"), filename)
}

