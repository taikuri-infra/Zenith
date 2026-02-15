package top

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "top",
	Short: "Real-time resource monitor",
	Long: `Display a real-time, htop-style resource monitoring dashboard.

Updates every 2 seconds with CPU, memory, and network usage for all
apps and databases in the current project.

Keys:
  ↑/↓  Navigate
  s    Sort by column
  q    Quit`,
	RunE: runTop,
}

type model struct {
	width      int
	height     int
	cursor     int
	sortColumn int
	apps       []appResource
	databases  []dbResource
	quitting   bool
}

type appResource struct {
	Name     string
	Replicas string
	CPU      string
	Memory   string
	NetIn    string
	NetOut   string
}

type dbResource struct {
	Name        string
	Engine      string
	Connections int
	Storage     string
	QPS         int
}

type tickMsg struct{}

func initialModel() model {
	return model{
		width:  80,
		height: 24,
		apps:   []appResource{},
		databases: []dbResource{},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			total := len(m.apps) + len(m.databases)
			if m.cursor < total-1 {
				m.cursor++
			}
		case "s":
			m.sortColumn = (m.sortColumn + 1) % 6
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	// Header
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Zenith Resource Monitor")
	sb.WriteString(title + "\n\n")

	// Overall resource bars
	cpuLabel := lipgloss.NewStyle().Foreground(tui.ColorMuted).Width(10).Render("CPU")
	memLabel := lipgloss.NewStyle().Foreground(tui.ColorMuted).Width(10).Render("Memory")

	sb.WriteString(fmt.Sprintf("  %s %s\n", cpuLabel, renderBar(0, 30)))
	sb.WriteString(fmt.Sprintf("  %s %s\n", memLabel, renderBar(0, 30)))

	// Apps section
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).Render("  Apps") + "\n")

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary)
	sb.WriteString(fmt.Sprintf("  %-20s %-10s %-12s %-12s %-10s %-10s\n",
		headerStyle.Render("NAME"),
		headerStyle.Render("INSTANCES"),
		headerStyle.Render("CPU"),
		headerStyle.Render("MEMORY"),
		headerStyle.Render("NET IN"),
		headerStyle.Render("NET OUT")))

	if len(m.apps) == 0 {
		sb.WriteString(fmt.Sprintf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No apps running")))
	}

	for _, app := range m.apps {
		sb.WriteString(fmt.Sprintf("  %-20s %-10s %-12s %-12s %-10s %-10s\n",
			app.Name, app.Replicas, app.CPU, app.Memory, app.NetIn, app.NetOut))
	}

	// Databases section
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).Render("  Databases") + "\n")

	sb.WriteString(fmt.Sprintf("  %-20s %-10s %-12s %-12s %-10s\n",
		headerStyle.Render("NAME"),
		headerStyle.Render("ENGINE"),
		headerStyle.Render("CONNS"),
		headerStyle.Render("STORAGE"),
		headerStyle.Render("QPS")))

	if len(m.databases) == 0 {
		sb.WriteString(fmt.Sprintf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No databases running")))
	}

	for _, db := range m.databases {
		sb.WriteString(fmt.Sprintf("  %-20s %-10s %-12d %-12s %-10d\n",
			db.Name, db.Engine, db.Connections, db.Storage, db.QPS))
	}

	// Footer
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(tui.ColorMuted).Render(
		"  [↑↓] navigate  [s] sort  [q] quit"))
	sb.WriteString("\n")

	return sb.String()
}

func renderBar(percent float64, width int) string {
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
		color = lipgloss.Color("#ef4444")
	case percent > 60:
		color = lipgloss.Color("#f59e0b")
	default:
		color = lipgloss.Color("#10b981")
	}

	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(strings.Repeat("░", width-filled))

	return fmt.Sprintf("%s%s %3.0f%%", bar, empty, percent)
}

func runTop(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
