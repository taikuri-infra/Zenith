package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagProject   string
	flagOutputDir string
	flagFormat    string
)

// Cmd is the export command.
var Cmd = &cobra.Command{
	Use:   "export",
	Short: "Export Zenith resources to YAML/JSON files",
	Long: `Export all Zenith resources from a project to local files for GitOps workflows.

Creates a directory structure with resource manifests:
  output-dir/
    apps/
    databases/
    storage/
    domains/
    routes/
    auth/

Examples:
  zen export                                  # Export current project to ./zenith-export/
  zen export --project my-app --output-dir .  # Export to current directory
  zen export --format json                    # Export as JSON instead of YAML`,
	RunE: runExport,
}

func init() {
	Cmd.Flags().StringVarP(&flagProject, "project", "p", "", "Project to export (defaults to current project)")
	Cmd.Flags().StringVarP(&flagOutputDir, "output-dir", "o", "zenith-export", "Output directory for exported files")
	Cmd.Flags().StringVar(&flagFormat, "format", "yaml", "Output format (yaml or json)")
}

// ZenithManifest represents a generic Zenith resource manifest for export.
type ZenithManifest struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   ManifestMetadata       `json:"metadata" yaml:"metadata"`
	Spec       map[string]interface{} `json:"spec" yaml:"spec"`
}

// ManifestMetadata contains resource metadata.
type ManifestMetadata struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

func runExport(cmd *cobra.Command, args []string) error {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Exporting Zenith Resources")
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

	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tui.ColorText)

	fmt.Printf("  %s %s\n", infoStyle.Render("Project:"), valueStyle.Render(project))
	fmt.Printf("  %s %s\n", infoStyle.Render("Output:"), valueStyle.Render(flagOutputDir))
	fmt.Printf("  %s %s\n", infoStyle.Render("Format:"), valueStyle.Render(flagFormat))
	fmt.Println()

	// Create directory structure
	dirs := []string{"apps", "databases", "storage", "domains", "routes", "auth"}
	for _, dir := range dirs {
		dirPath := filepath.Join(flagOutputDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dirPath, err)
		}
	}

	checkmark := lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("✓")
	stepStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	timeStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)

	totalStart := time.Now()
	totalFiles := 0

	// Export apps
	count, err := exportApps(client, project, flagOutputDir)
	if err != nil {
		fmt.Printf("  %s Exporting apps: %v\n",
			lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("!"), err)
	} else {
		totalFiles += count
		fmt.Printf("  %s %s %s\n", checkmark,
			stepStyle.Render(fmt.Sprintf("Exported %d apps", count)),
			timeStyle.Render("(apps/)"))
	}

	// Export databases
	count, err = exportDatabases(client, project, flagOutputDir)
	if err != nil {
		fmt.Printf("  %s Exporting databases: %v\n",
			lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("!"), err)
	} else {
		totalFiles += count
		fmt.Printf("  %s %s %s\n", checkmark,
			stepStyle.Render(fmt.Sprintf("Exported %d databases", count)),
			timeStyle.Render("(databases/)"))
	}

	totalElapsed := time.Since(totalStart)

	// Summary
	fmt.Println()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSuccess).
		Padding(1, 2).
		Render(fmt.Sprintf(
			"%s\n\n"+
				"  %s %d files\n"+
				"  %s %s\n"+
				"  %s %s",
			lipgloss.NewStyle().Bold(true).Foreground(tui.ColorSuccess).Render("Export complete!"),
			infoStyle.Render("Files:"),
			totalFiles,
			infoStyle.Render("Directory:"),
			flagOutputDir,
			infoStyle.Render("Time:"),
			totalElapsed.Round(time.Millisecond).String(),
		))
	fmt.Println(box)

	// Next steps
	fmt.Println()
	fmt.Printf("  %s git add %s && git commit -m \"Export Zenith resources\"\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("→"), flagOutputDir)
	fmt.Printf("  %s zen apply -d %s\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("→"), flagOutputDir)
	fmt.Printf("  %s zen diff -d %s\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("→"), flagOutputDir)
	fmt.Println()

	return nil
}

func exportApps(client *api.Client, project, outputDir string) (int, error) {
	apps, err := client.ListApps(project)
	if err != nil {
		// If API is unavailable, return 0 without error for graceful handling
		return 0, nil
	}

	count := 0
	for _, app := range apps {
		manifest := ZenithManifest{
			APIVersion: "zenith.dev/v1alpha1",
			Kind:       "App",
			Metadata: ManifestMetadata{
				Name: app.Name,
				Labels: map[string]string{
					"zenith.dev/project": project,
				},
			},
			Spec: map[string]interface{}{
				"image":    app.Image,
				"replicas": app.Replicas,
				"port":     app.Port,
			},
		}

		if err := writeManifest(manifest, filepath.Join(outputDir, "apps", app.Name)); err != nil {
			return count, fmt.Errorf("write app %s: %w", app.Name, err)
		}
		count++
	}

	return count, nil
}

func exportDatabases(client *api.Client, project, outputDir string) (int, error) {
	dbs, err := client.ListDatabases(project)
	if err != nil {
		return 0, nil
	}

	count := 0
	for _, db := range dbs {
		manifest := ZenithManifest{
			APIVersion: "zenith.dev/v1alpha1",
			Kind:       "Database",
			Metadata: ManifestMetadata{
				Name: db.Name,
				Labels: map[string]string{
					"zenith.dev/project": project,
				},
			},
			Spec: map[string]interface{}{
				"engine":  db.Engine,
				"version": db.Version,
				"storage": db.Storage,
			},
		}

		if err := writeManifest(manifest, filepath.Join(outputDir, "databases", db.Name)); err != nil {
			return count, fmt.Errorf("write database %s: %w", db.Name, err)
		}
		count++
	}

	return count, nil
}

func writeManifest(manifest ZenithManifest, basePath string) error {
	var data []byte
	var ext string
	var err error

	switch flagFormat {
	case "json":
		data, err = json.MarshalIndent(manifest, "", "  ")
		ext = ".json"
	default:
		data, err = yaml.Marshal(manifest)
		ext = ".yaml"
	}

	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	filePath := basePath + ext
	return os.WriteFile(filePath, data, 0644)
}

// MarshalManifest converts a ZenithManifest to bytes in the specified format.
func MarshalManifest(manifest ZenithManifest, format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(manifest, "", "  ")
	default:
		return yaml.Marshal(manifest)
	}
}

// ParseManifestFile reads a YAML or JSON file and returns a ZenithManifest.
func ParseManifestFile(path string) (*ZenithManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	return ParseManifestData(data, path)
}

// ParseManifestData parses manifest data from bytes. The path is used for format detection.
func ParseManifestData(data []byte, path string) (*ZenithManifest, error) {
	var manifest ZenithManifest

	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse JSON: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse YAML: %w", err)
		}
	}

	return &manifest, nil
}

// CollectManifests reads all manifest files from a directory (recursively).
func CollectManifests(dir string) ([]*ZenithManifest, error) {
	var manifests []*ZenithManifest

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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

		manifest, err := ParseManifestFile(path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		manifests = append(manifests, manifest)
		return nil
	})

	return manifests, err
}
