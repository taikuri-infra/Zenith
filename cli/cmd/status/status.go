package status

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	// Header
	fmt.Println(titleStyle.Render("  Zenith Project Status"))

	// Project info
	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	fmt.Printf("  %s %s\n", infoStyle.Render("Project:"), valStyle.Render("my-project"))
	fmt.Printf("  %s %s\n", infoStyle.Render("Region:"), valStyle.Render("fsn1"))
	fmt.Printf("  %s %s\n", infoStyle.Render("Plan:"), valStyle.Render("pro"))

	// Apps section
	fmt.Println()
	fmt.Println(sectionStyle.Render("  Apps"))
	fmt.Println(renderTable(
		[]string{"NAME", "STATUS", "REPLICAS", "CPU", "MEMORY"},
		[][]string{},
		headerStyle, cellStyle,
	))

	// Databases section
	fmt.Println(sectionStyle.Render("  Databases"))
	fmt.Println(renderTable(
		[]string{"NAME", "ENGINE", "STORAGE", "BACKUP", "STATUS"},
		[][]string{},
		headerStyle, cellStyle,
	))

	// Nodes section
	fmt.Println(sectionStyle.Render("  Nodes"))
	fmt.Println(renderTable(
		[]string{"NAME", "TYPE", "CPU", "MEMORY", "STATUS"},
		[][]string{},
		headerStyle, cellStyle,
	))

	// Cost summary
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
