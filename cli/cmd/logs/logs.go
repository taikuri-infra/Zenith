package logs

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagFollow bool
	flagSince  string
	flagJSON   bool
	flagTail   int
	flagLevel  string
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
	Cmd.Flags().BoolVarP(&flagFollow, "follow", "f", true, "Follow log output (stream)")
	Cmd.Flags().StringVar(&flagSince, "since", "1h", "Show logs since duration (e.g., 1h, 6h, 24h, 7d)")
	Cmd.Flags().BoolVar(&flagJSON, "json", false, "Output raw JSON logs")
	Cmd.Flags().IntVarP(&flagTail, "tail", "n", 100, "Number of recent lines to show (non-follow mode)")
	Cmd.Flags().StringVar(&flagLevel, "level", "", "Filter by log level (info, warn, error, debug)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	appName := args[0]

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render(fmt.Sprintf("  Logs: %s", appName))
	fmt.Println(header)

	if flagSince != "" && !flagFollow {
		fmt.Printf("  %s %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Since:"),
			flagSince)
	}
	fmt.Println()

	// Load config and create client
	cfg, err := config.Load()
	if err != nil || cfg.Token == "" {
		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorError).Render("Not logged in — run: zen login"))
		return nil
	}

	client := cliapi.NewClient(cfg.APIEndpoint, cfg.Token)

	// Find the app by name across all user apps
	apps, err := client.ListApps("")
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	var appID string
	for _, a := range apps {
		if a.Name == appName {
			appID = a.ID
			break
		}
	}
	if appID == "" {
		return fmt.Errorf("app '%s' not found — check the name or run 'zen status'", appName)
	}

	if flagFollow {
		// Handle Ctrl+C gracefully
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		done := make(chan struct{})

		go func() {
			<-sigCh
			fmt.Println()
			close(done)
		}()

		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Press Ctrl+C to stop following"))
		fmt.Println()

		streamErr := client.StreamAppLogs(appID, func(entry cliapi.LogEntry) bool {
			select {
			case <-done:
				return false
			default:
			}
			printLogEntry(entry)
			return true
		})

		// Wait for signal goroutine to finish
		select {
		case <-done:
		default:
		}

		if streamErr != nil {
			return fmt.Errorf("log stream error: %w", streamErr)
		}
		return nil
	}

	// Non-streaming: fetch log history
	result, err := client.GetAppLogs(appID, flagLevel, flagSince, flagTail)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(result.Entries) == 0 {
		fmt.Printf("  %s\n\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No logs found for this time range."))
		return nil
	}

	for _, entry := range result.Entries {
		printLogEntry(entry)
	}

	return nil
}

func printLogEntry(entry cliapi.LogEntry) {
	if flagJSON {
		fmt.Printf(`{"timestamp":%q,"level":%q,"line":%q}`+"\n",
			entry.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			entry.Level, entry.Line)
		return
	}

	ts := entry.Timestamp.Format("15:04:05")
	pod := ""
	if entry.Labels != nil {
		pod = entry.Labels["pod"]
	}

	fmt.Println(FormatLogLine(ts, entry.Level, entry.Line, pod))
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

	podPart := ""
	if pod != "" {
		podPart = " " + podStyle.Render(fmt.Sprintf("[%s]", pod))
	}

	return fmt.Sprintf("%s %s%s %s",
		tsStyle.Render(timestamp),
		levelStyle.Render(fmt.Sprintf("%-5s", level)),
		podPart,
		message,
	)
}
