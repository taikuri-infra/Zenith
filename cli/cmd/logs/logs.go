package logs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagFollow bool
	flagSince  string
	flagJSON   bool
	flagTail   int
)

var Cmd = &cobra.Command{
	Use:   "logs [app]",
	Short: "Stream application logs",
	Long: `Stream color-coded logs from an application.

Log levels are color-coded:
  INF = green, WRN = yellow, ERR = red, DBG = gray

Examples:
  zen logs my-app
  zen logs my-app --follow
  zen logs my-app --since 1h
  zen logs my-app --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

func init() {
	Cmd.Flags().BoolVarP(&flagFollow, "follow", "f", true, "Follow log output")
	Cmd.Flags().StringVar(&flagSince, "since", "", "Show logs since duration (e.g., 1h, 30m, 1d)")
	Cmd.Flags().BoolVar(&flagJSON, "json", false, "Output raw JSON logs")
	Cmd.Flags().IntVarP(&flagTail, "tail", "n", 100, "Number of recent lines to show")
}

func runLogs(cmd *cobra.Command, args []string) error {
	appName := args[0]

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render(fmt.Sprintf("  Logs: %s", appName))
	fmt.Println(header)

	if flagSince != "" {
		fmt.Printf("  %s %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Since:"),
			flagSince)
	}
	fmt.Println()

	// In production: connect to Loki or K8s log stream
	// For now, display a message indicating no logs available
	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No logs available. Deploy an app first with 'zen deploy'."))

	if flagFollow {
		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Press Ctrl+C to stop following"))
	}

	return nil
}

// FormatLogLine formats a log line with color coding based on level.
func FormatLogLine(timestamp, level, message, pod string) string {
	var levelStyle lipgloss.Style

	switch strings.ToUpper(level) {
	case "INF", "INFO":
		levelStyle = lipgloss.NewStyle().Foreground(tui.ColorSuccess)
	case "WRN", "WARN", "WARNING":
		levelStyle = lipgloss.NewStyle().Foreground(tui.ColorWarning)
	case "ERR", "ERROR":
		levelStyle = lipgloss.NewStyle().Foreground(tui.ColorError)
	case "DBG", "DEBUG":
		levelStyle = lipgloss.NewStyle().Foreground(tui.ColorMuted)
	default:
		levelStyle = lipgloss.NewStyle().Foreground(tui.ColorText)
	}

	tsStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	podStyle := lipgloss.NewStyle().Foreground(tui.ColorSecondary)

	return fmt.Sprintf("%s %s %s %s",
		tsStyle.Render(timestamp),
		levelStyle.Render(fmt.Sprintf("%-5s", level)),
		podStyle.Render(fmt.Sprintf("[%s]", pod)),
		message,
	)
}
