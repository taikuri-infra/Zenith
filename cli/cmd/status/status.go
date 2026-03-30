package status

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status overview",
	Long: `Display a rich overview of the current project including apps,
databases, nodes, and cost estimates.

Example:
  zen status`,
	RunE: runStatus,
}

// ResourceStatus represents a resource with its status.
type ResourceStatus struct {
	Name     string
	Type     string
	Status   string
	Replicas string
	CPU      string
	Memory   string
	Extra    string
}

func runStatus(cmd *cobra.Command, args []string) error {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Foreground(tui.ColorText).
		Padding(0, 1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorText).
		MarginTop(1)

	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError)

	// Load config and create API client
	cfg, err := config.Load()
	if err != nil || cfg.Token == "" {
		fmt.Println(errStyle.Render("  Not logged in — run: zen login"))
		return nil
	}

	client := cliapi.NewClient(cfg.APIEndpoint, cfg.Token)

	// Header
	fmt.Println(titleStyle.Render("  Zenith Project Status"))

	// Fetch projects
	projects, err := client.ListProjects()
	if err != nil {
		fmt.Println(errStyle.Render("  Failed to fetch projects: " + err.Error()))
		return nil
	}
	if len(projects) == 0 {
		fmt.Println(infoStyle.Render("  No projects found. Create one at the dashboard."))
		return nil
	}

	// Select current project: prefer cfg.Project match by name or ID, else first project
	current := projects[0]
	for _, p := range projects {
		if cfg.Project != "" && (p.ID == cfg.Project || p.Name == cfg.Project || p.DisplayName == cfg.Project) {
			current = p
			break
		}
	}

	displayName := current.DisplayName
	if displayName == "" {
		displayName = current.Name
	}
	region := current.Region
	if region == "" {
		region = cfg.Region
	}
	plan := current.Plan
	if plan == "" {
		plan = "free"
	}

	fmt.Printf("  %s %s\n", infoStyle.Render("Project:"), valStyle.Render(displayName))
	fmt.Printf("  %s %s\n", infoStyle.Render("Region: "), valStyle.Render(region))
	fmt.Printf("  %s %s\n", infoStyle.Render("Plan:   "), valStyle.Render(plan))

	// Apps section
	fmt.Println()
	fmt.Println(sectionStyle.Render("  Apps"))

	apps, err := client.ListApps(current.ID)
	if err != nil {
		fmt.Println("  " + errStyle.Render("Failed to fetch apps: "+err.Error()))
	} else {
		appRows := make([][]string, 0, len(apps))
		for _, a := range apps {
			replicas := fmt.Sprintf("%d", a.Replicas)
			if replicas == "0" {
				replicas = "—"
			}
			appRows = append(appRows, []string{a.Name, a.Status, replicas, a.CPU, a.Memory})
		}
		fmt.Println(renderTable(
			[]string{"NAME", "STATUS", "REPLICAS", "CPU", "MEMORY"},
			appRows,
			headerStyle, cellStyle,
		))
	}

	// Databases section
	fmt.Println(sectionStyle.Render("  Databases"))

	dbs, err := client.ListDatabases(current.ID)
	if err != nil {
		fmt.Println("  " + errStyle.Render("Failed to fetch databases: "+err.Error()))
	} else {
		dbRows := make([][]string, 0, len(dbs))
		for _, d := range dbs {
			dbRows = append(dbRows, []string{d.Name, d.Engine, d.Storage, "—", d.Status})
		}
		fmt.Println(renderTable(
			[]string{"NAME", "ENGINE", "STORAGE", "BACKUP", "STATUS"},
			dbRows,
			headerStyle, cellStyle,
		))
	}

	// Cost summary (placeholder — requires billing API)
	fmt.Println(sectionStyle.Render("  Cost Estimate"))
	fmt.Printf("  %s %s\n\n",
		infoStyle.Render("Monthly:"),
		lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).Render("€0.00"))

	return nil
}

func renderTable(headers []string, rows [][]string, headerStyle, cellStyle lipgloss.Style) string {
	if len(rows) == 0 {
		return fmt.Sprintf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("  No resources found"))
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h) + 2
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell)+2 > widths[i] {
				widths[i] = len(cell) + 2
			}
		}
	}

	var sb strings.Builder

	// Header
	sb.WriteString("  ")
	for i, h := range headers {
		sb.WriteString(headerStyle.Width(widths[i]).Render(h))
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		sb.WriteString("  ")
		for i, cell := range row {
			if i < len(widths) {
				rendered := cell
				if headers[i] == "STATUS" {
					rendered = tui.StatusBadge(cell)
				}
				sb.WriteString(cellStyle.Width(widths[i]).Render(rendered))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderProgressBar creates a progress bar with the given percentage.
func RenderProgressBar(percent float64, width int) string {
	filled := int(percent / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	var color lipgloss.Color
	switch {
	case percent > 80:
		color = lipgloss.Color("#ef4444") // red
	case percent > 60:
		color = lipgloss.Color("#f59e0b") // amber
	default:
		color = lipgloss.Color("#10b981") // emerald
	}

	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(strings.Repeat("░", width-filled))

	return fmt.Sprintf("%s%s %3.0f%%", bar, empty, percent)
}
