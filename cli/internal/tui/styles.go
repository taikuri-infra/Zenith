package tui

import "github.com/charmbracelet/lipgloss"

// Color palette - matches Zenith design system (emerald accent)
var (
	ColorPrimary   = lipgloss.Color("#10b981") // Emerald 500
	ColorSecondary = lipgloss.Color("#6366f1") // Indigo 500
	ColorSuccess   = lipgloss.Color("#22c55e") // Green 500
	ColorWarning   = lipgloss.Color("#f59e0b") // Amber 500
	ColorError     = lipgloss.Color("#ef4444") // Red 500
	ColorMuted     = lipgloss.Color("#6b7280") // Gray 500
	ColorText      = lipgloss.Color("#f3f4f6") // Gray 100
	ColorBg        = lipgloss.Color("#111827") // Gray 900
	ColorBorder    = lipgloss.Color("#374151") // Gray 700
)

// Reusable styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Padding(0, 1)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	StatusRunning = SuccessStyle.Render("Running")
	StatusPending = WarningStyle.Render("Pending")
	StatusFailed  = ErrorStyle.Render("Failed")
	StatusStopped = SubtitleStyle.Render("Stopped")
)

// Color helper functions for inline text coloring.
func Green(s string) string  { return SuccessStyle.Render(s) }
func Red(s string) string    { return ErrorStyle.Render(s) }
func Yellow(s string) string { return WarningStyle.Render(s) }
func Cyan(s string) string   { return lipgloss.NewStyle().Foreground(ColorSecondary).Render(s) }

func StatusBadge(phase string) string {
	switch phase {
	case "Running", "Ready", "Active":
		return SuccessStyle.Render(phase)
	case "Pending", "Provisioning", "Creating", "Configuring", "Building", "Deploying":
		return WarningStyle.Render(phase)
	case "Failed":
		return ErrorStyle.Render(phase)
	case "Stopped", "Deleting", "Suspended":
		return SubtitleStyle.Render(phase)
	default:
		return lipgloss.NewStyle().Foreground(ColorText).Render(phase)
	}
}
