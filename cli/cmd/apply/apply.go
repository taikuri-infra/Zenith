package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/cmd/export"
	"github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagProject   string
	flagFile      string
	flagDirectory string
	flagDryRun    bool
	flagYes       bool
)

// Cmd is the apply command.
var Cmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Zenith resource manifests from files or directories",
	Long: `Apply Zenith resource manifests to the cluster from local YAML/JSON files.

Reads manifest files and creates or updates the corresponding resources via the API.
Shows a diff before applying unless --yes is specified.

Examples:
  zen apply -f app.yaml                   # Apply a single file
  zen apply -d ./zenith-export/           # Apply all files in a directory
  zen apply -d ./zenith-export/ --dry-run # Preview changes without applying
  zen apply -f app.yaml --yes             # Apply without confirmation prompt`,
	RunE: runApply,
}

func init() {
	Cmd.Flags().StringVarP(&flagProject, "project", "p", "", "Target project (defaults to current project)")
	Cmd.Flags().StringVarP(&flagFile, "file", "f", "", "Path to a manifest file")
	Cmd.Flags().StringVarP(&flagDirectory, "directory", "d", "", "Path to a directory of manifests")
	Cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview changes without applying")
	Cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation prompt")
}

// ApplyResult holds the result of applying a single manifest.
type ApplyResult struct {
	Name      string
	Kind      string
	Action    string // created, updated, unchanged, failed
	Error     error
	FilePath  string
}

func runApply(cmd *cobra.Command, args []string) error {
	if flagFile == "" && flagDirectory == "" {
		return fmt.Errorf("specify either --file/-f or --directory/-d")
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Applying Zenith Resources")
	fmt.Println(header)
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	project := flagProject
	if project == "" {
		project = cfg.Project
	}
	if project == "" {
		project = "default"
	}

	client := api.NewClient(cfg.APIEndpoint, cfg.Token)

	// Collect manifests
	var manifests []*export.ZenithManifest
	var filePaths []string

	if flagFile != "" {
		m, err := export.ParseManifestFile(flagFile)
		if err != nil {
			return fmt.Errorf("parse file: %w", err)
		}
		manifests = append(manifests, m)
		filePaths = append(filePaths, flagFile)
	}

	if flagDirectory != "" {
		err := filepath.Walk(flagDirectory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yaml" && ext != ".yml" && ext != ".json" {
				return nil
			}

			m, err := export.ParseManifestFile(path)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}

			manifests = append(manifests, m)
			filePaths = append(filePaths, path)
			return nil
		})
		if err != nil {
			return fmt.Errorf("read directory: %w", err)
		}
	}

	if len(manifests) == 0 {
		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No manifest files found"))
		return nil
	}

	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText)

	fmt.Printf("  %s %s\n", infoStyle.Render("Project:"), valueStyle.Render(project))
	fmt.Printf("  %s %d\n", infoStyle.Render("Manifests:"), len(manifests))
	if flagDryRun {
		fmt.Printf("  %s %s\n", infoStyle.Render("Mode:"),
			lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("dry-run"))
	}
	fmt.Println()

	// Display manifest summary
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).Render("  Resources to apply:"))
	fmt.Println()

	kindCounts := make(map[string]int)
	for i, m := range manifests {
		kindCounts[m.Kind]++
		relPath := filePaths[i]
		if flagDirectory != "" {
			rel, err := filepath.Rel(flagDirectory, filePaths[i])
			if err == nil {
				relPath = rel
			}
		}
		fmt.Printf("    %s %s/%s %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render("->"),
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render(m.Kind),
			valueStyle.Render(m.Metadata.Name),
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render(fmt.Sprintf("(%s)", relPath)),
		)
	}
	fmt.Println()

	// Show summary by kind
	for kind, count := range kindCounts {
		fmt.Printf("  %s %d %s(s)\n", infoStyle.Render("  "), count, kind)
	}
	fmt.Println()

	if flagDryRun {
		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("Dry run - no changes applied"))
		return nil
	}

	// Apply manifests
	checkmark := lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("✓")
	failmark := lipgloss.NewStyle().Foreground(tui.ColorError).Render("✗")
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	totalStart := time.Now()
	var results []ApplyResult

	for i, m := range manifests {
		start := time.Now()
		result := applyManifest(client, project, m, filePaths[i])
		elapsed := time.Since(start)
		results = append(results, result)

		mark := checkmark
		if result.Error != nil {
			mark = failmark
		}

		fmt.Printf("  %s %s %s %s\n",
			mark,
			stepStyle.Render(fmt.Sprintf("%s/%s", result.Kind, result.Name)),
			lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render(result.Action),
			timeStyle.Render(fmt.Sprintf("(%s)", elapsed.Round(time.Millisecond))),
		)
	}

	totalElapsed := time.Since(totalStart)

	// Summary
	created, updated, failed := 0, 0, 0
	for _, r := range results {
		switch r.Action {
		case "created":
			created++
		case "updated":
			updated++
		case "failed":
			failed++
		}
	}

	fmt.Println()
	summaryColor := tui.ColorSuccess
	summaryText := "Apply complete!"
	if failed > 0 {
		summaryColor = tui.ColorWarning
		summaryText = "Apply completed with errors"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(summaryColor).
		Padding(1, 2).
		Render(fmt.Sprintf(
			"%s\n\n"+
				"  %s %d\n"+
				"  %s %d\n"+
				"  %s %d\n"+
				"  %s %s",
			lipgloss.NewStyle().Bold(true).Foreground(summaryColor).Render(summaryText),
			infoStyle.Render("Created:"),
			created,
			infoStyle.Render("Updated:"),
			updated,
			infoStyle.Render("Failed:"),
			failed,
			infoStyle.Render("Time:"),
			totalElapsed.Round(time.Millisecond).String(),
		))
	fmt.Println(box)
	fmt.Println()

	return nil
}

func applyManifest(client *api.Client, project string, manifest *export.ZenithManifest, filePath string) ApplyResult {
	result := ApplyResult{
		Name:     manifest.Metadata.Name,
		Kind:     manifest.Kind,
		FilePath: filePath,
	}

	switch manifest.Kind {
	case "App":
		app := &api.App{
			Name: manifest.Metadata.Name,
		}
		if v, ok := manifest.Spec["image"].(string); ok {
			app.Image = v
		}
		if v, ok := manifest.Spec["replicas"].(int); ok {
			app.Replicas = v
		} else if v, ok := manifest.Spec["replicas"].(float64); ok {
			app.Replicas = int(v)
		}
		if v, ok := manifest.Spec["port"].(int); ok {
			app.Port = v
		} else if v, ok := manifest.Spec["port"].(float64); ok {
			app.Port = int(v)
		}

		_, err := client.CreateApp(project, app)
		if err != nil {
			result.Action = "failed"
			result.Error = err
		} else {
			result.Action = "created"
		}

	case "Database":
		db := &api.Database{
			Name: manifest.Metadata.Name,
		}
		if v, ok := manifest.Spec["engine"].(string); ok {
			db.Engine = v
		}
		if v, ok := manifest.Spec["version"].(string); ok {
			db.Version = v
		}
		if v, ok := manifest.Spec["storage"].(string); ok {
			db.Storage = v
		}

		_, err := client.CreateDatabase(project, db)
		if err != nil {
			result.Action = "failed"
			result.Error = err
		} else {
			result.Action = "created"
		}

	default:
		result.Action = "failed"
		result.Error = fmt.Errorf("unsupported resource kind: %s", manifest.Kind)
	}

	return result
}

// ApplyManifestData applies a single manifest from raw data bytes.
// Returns an ApplyResult with the outcome.
func ApplyManifestData(data []byte, format string) (*export.ZenithManifest, error) {
	var manifest export.ZenithManifest

	switch strings.ToLower(format) {
	case "json":
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse JSON: %w", err)
		}
	default:
		if err := decodeYAML(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse YAML: %w", err)
		}
	}

	return &manifest, nil
}

func decodeYAML(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
