package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
)

// Cmd is the diff command.
var Cmd = &cobra.Command{
	Use:   "diff",
	Short: "Show diff between local manifests and cluster state",
	Long: `Compare local Zenith resource manifests against the current cluster state.

Shows additions, modifications, and deletions with colorized output.

Examples:
  zen diff -f app.yaml                    # Diff a single file
  zen diff -d ./zenith-export/            # Diff all files in a directory
  zen diff -d ./zenith-export/ -p my-app  # Diff against a specific project`,
	RunE: runDiff,
}

func init() {
	Cmd.Flags().StringVarP(&flagProject, "project", "p", "", "Project to diff against (defaults to current project)")
	Cmd.Flags().StringVarP(&flagFile, "file", "f", "", "Path to a manifest file")
	Cmd.Flags().StringVarP(&flagDirectory, "directory", "d", "", "Path to a directory of manifests")
}

// DiffEntry represents a single resource diff.
type DiffEntry struct {
	Kind       string
	Name       string
	Action     string // added, modified, deleted, unchanged
	LocalSpec  map[string]interface{}
	RemoteSpec map[string]interface{}
	Changes    []FieldChange
}

// FieldChange represents a change in a single field.
type FieldChange struct {
	Field    string
	OldValue string
	NewValue string
}

// Styles for diff output
var (
	addedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")) // green
	removedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")) // red
	changedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b")) // amber
	unchangedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")) // gray
)

func runDiff(cmd *cobra.Command, args []string) error {
	if flagFile == "" && flagDirectory == "" {
		return fmt.Errorf("specify either --file/-f or --directory/-d")
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Zenith Resource Diff")
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

	// Collect local manifests
	var localManifests []*export.ZenithManifest

	if flagFile != "" {
		m, err := export.ParseManifestFile(flagFile)
		if err != nil {
			return fmt.Errorf("parse file: %w", err)
		}
		localManifests = append(localManifests, m)
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

			localManifests = append(localManifests, m)
			return nil
		})
		if err != nil {
			return fmt.Errorf("read directory: %w", err)
		}
	}

	if len(localManifests) == 0 {
		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No manifest files found"))
		return nil
	}

	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	fmt.Printf("  %s %s\n", infoStyle.Render("Project:"),
		lipgloss.NewStyle().Foreground(tui.ColorText).Render(project))
	fmt.Printf("  %s %d files\n", infoStyle.Render("Local:"), len(localManifests))
	fmt.Println()

	// Fetch remote state
	remoteApps := fetchRemoteApps(client, project)
	remoteDbs := fetchRemoteDatabases(client, project)

	// Build diff entries
	var diffs []DiffEntry

	for _, m := range localManifests {
		entry := buildDiffEntry(m, remoteApps, remoteDbs)
		diffs = append(diffs, entry)
	}

	// Check for resources in remote but not in local (deletions)
	localKeys := make(map[string]bool)
	for _, m := range localManifests {
		key := fmt.Sprintf("%s/%s", m.Kind, m.Metadata.Name)
		localKeys[key] = true
	}

	for name, spec := range remoteApps {
		key := fmt.Sprintf("App/%s", name)
		if !localKeys[key] {
			diffs = append(diffs, DiffEntry{
				Kind:       "App",
				Name:       name,
				Action:     "deleted",
				RemoteSpec: spec,
			})
		}
	}

	for name, spec := range remoteDbs {
		key := fmt.Sprintf("Database/%s", name)
		if !localKeys[key] {
			diffs = append(diffs, DiffEntry{
				Kind:       "Database",
				Name:       name,
				Action:     "deleted",
				RemoteSpec: spec,
			})
		}
	}

	// Sort diffs by action priority: added, modified, deleted, unchanged
	actionOrder := map[string]int{"added": 0, "modified": 1, "deleted": 2, "unchanged": 3}
	sort.Slice(diffs, func(i, j int) bool {
		oi, oj := actionOrder[diffs[i].Action], actionOrder[diffs[j].Action]
		if oi != oj {
			return oi < oj
		}
		return diffs[i].Name < diffs[j].Name
	})

	// Display diffs
	added, modified, deleted, unchanged := 0, 0, 0, 0

	for _, d := range diffs {
		switch d.Action {
		case "added":
			added++
			renderAddedDiff(d)
		case "modified":
			modified++
			renderModifiedDiff(d)
		case "deleted":
			deleted++
			renderDeletedDiff(d)
		case "unchanged":
			unchanged++
			renderUnchangedDiff(d)
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).Render("  Summary:"))
	if added > 0 {
		fmt.Printf("    %s %d resource(s) to add\n", addedStyle.Render("+"), added)
	}
	if modified > 0 {
		fmt.Printf("    %s %d resource(s) to modify\n", changedStyle.Render("~"), modified)
	}
	if deleted > 0 {
		fmt.Printf("    %s %d resource(s) to delete\n", removedStyle.Render("-"), deleted)
	}
	if unchanged > 0 {
		fmt.Printf("    %s %d resource(s) unchanged\n", unchangedStyle.Render("="), unchanged)
	}
	if added == 0 && modified == 0 && deleted == 0 {
		fmt.Printf("    %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("No changes detected"))
	}
	fmt.Println()

	return nil
}

func buildDiffEntry(manifest *export.ZenithManifest, remoteApps map[string]map[string]interface{}, remoteDbs map[string]map[string]interface{}) DiffEntry {
	entry := DiffEntry{
		Kind:      manifest.Kind,
		Name:      manifest.Metadata.Name,
		LocalSpec: manifest.Spec,
	}

	switch manifest.Kind {
	case "App":
		if remote, ok := remoteApps[manifest.Metadata.Name]; ok {
			entry.RemoteSpec = remote
			changes := compareSpecs(manifest.Spec, remote)
			if len(changes) > 0 {
				entry.Action = "modified"
				entry.Changes = changes
			} else {
				entry.Action = "unchanged"
			}
		} else {
			entry.Action = "added"
		}
	case "Database":
		if remote, ok := remoteDbs[manifest.Metadata.Name]; ok {
			entry.RemoteSpec = remote
			changes := compareSpecs(manifest.Spec, remote)
			if len(changes) > 0 {
				entry.Action = "modified"
				entry.Changes = changes
			} else {
				entry.Action = "unchanged"
			}
		} else {
			entry.Action = "added"
		}
	default:
		entry.Action = "added"
	}

	return entry
}

func fetchRemoteApps(client *api.Client, project string) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	apps, err := client.ListApps(project)
	if err != nil {
		return result
	}

	for _, app := range apps {
		result[app.Name] = map[string]interface{}{
			"image":    app.Image,
			"replicas": app.Replicas,
			"port":     app.Port,
		}
	}

	return result
}

func fetchRemoteDatabases(client *api.Client, project string) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	dbs, err := client.ListDatabases(project)
	if err != nil {
		return result
	}

	for _, db := range dbs {
		result[db.Name] = map[string]interface{}{
			"engine":  db.Engine,
			"version": db.Version,
			"storage": db.Storage,
		}
	}

	return result
}

// compareSpecs compares two spec maps and returns field-level changes.
func compareSpecs(local, remote map[string]interface{}) []FieldChange {
	var changes []FieldChange

	// Check local fields against remote
	for key, localVal := range local {
		remoteVal, exists := remote[key]
		localStr := fmt.Sprintf("%v", localVal)
		if !exists {
			changes = append(changes, FieldChange{
				Field:    key,
				NewValue: localStr,
			})
		} else {
			remoteStr := fmt.Sprintf("%v", remoteVal)
			if localStr != remoteStr {
				changes = append(changes, FieldChange{
					Field:    key,
					OldValue: remoteStr,
					NewValue: localStr,
				})
			}
		}
	}

	// Check remote fields not in local
	for key, remoteVal := range remote {
		if _, exists := local[key]; !exists {
			changes = append(changes, FieldChange{
				Field:    key,
				OldValue: fmt.Sprintf("%v", remoteVal),
			})
		}
	}

	return changes
}

func renderAddedDiff(d DiffEntry) {
	fmt.Printf("  %s %s %s/%s\n",
		addedStyle.Render("+"),
		addedStyle.Bold(true).Render("ADD"),
		d.Kind,
		d.Name,
	)

	if d.LocalSpec != nil {
		specYAML, _ := yaml.Marshal(d.LocalSpec)
		for _, line := range strings.Split(strings.TrimSpace(string(specYAML)), "\n") {
			fmt.Printf("    %s\n", addedStyle.Render("+ "+line))
		}
	}
	fmt.Println()
}

func renderModifiedDiff(d DiffEntry) {
	fmt.Printf("  %s %s %s/%s\n",
		changedStyle.Render("~"),
		changedStyle.Bold(true).Render("MOD"),
		d.Kind,
		d.Name,
	)

	for _, change := range d.Changes {
		if change.OldValue == "" {
			// New field
			fmt.Printf("    %s\n", addedStyle.Render(fmt.Sprintf("+ %s: %s", change.Field, change.NewValue)))
		} else if change.NewValue == "" {
			// Removed field
			fmt.Printf("    %s\n", removedStyle.Render(fmt.Sprintf("- %s: %s", change.Field, change.OldValue)))
		} else {
			// Changed field
			fmt.Printf("    %s\n", removedStyle.Render(fmt.Sprintf("- %s: %s", change.Field, change.OldValue)))
			fmt.Printf("    %s\n", addedStyle.Render(fmt.Sprintf("+ %s: %s", change.Field, change.NewValue)))
		}
	}
	fmt.Println()
}

func renderDeletedDiff(d DiffEntry) {
	fmt.Printf("  %s %s %s/%s\n",
		removedStyle.Render("-"),
		removedStyle.Bold(true).Render("DEL"),
		d.Kind,
		d.Name,
	)

	if d.RemoteSpec != nil {
		specYAML, _ := yaml.Marshal(d.RemoteSpec)
		for _, line := range strings.Split(strings.TrimSpace(string(specYAML)), "\n") {
			fmt.Printf("    %s\n", removedStyle.Render("- "+line))
		}
	}
	fmt.Println()
}

func renderUnchangedDiff(d DiffEntry) {
	fmt.Printf("  %s %s %s/%s\n",
		unchangedStyle.Render("="),
		unchangedStyle.Render("OK "),
		d.Kind,
		d.Name,
	)
}

// CompareSpecs is the exported version of compareSpecs for testing.
func CompareSpecs(local, remote map[string]interface{}) []FieldChange {
	return compareSpecs(local, remote)
}
