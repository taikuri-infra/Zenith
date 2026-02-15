package db

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
	Long: `Create, list, connect to, and manage databases.

Examples:
  zen db list
  zen db create
  zen db connect my-db
  zen db backup my-db`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all databases",
	RunE:  runList,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new database",
	RunE:  runCreate,
}

var connectCmd = &cobra.Command{
	Use:   "connect [name]",
	Short: "Connect to a database shell",
	Long: `Open an interactive database shell with auto port-forwarding.

Launches the appropriate shell based on engine:
  PostgreSQL → psql
  MySQL      → mysql
  Redis      → redis-cli
  MongoDB    → mongosh

Example:
  zen db connect my-db`,
	Args: cobra.ExactArgs(1),
	RunE: runConnect,
}

var backupCmd = &cobra.Command{
	Use:   "backup [name]",
	Short: "Create a manual backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackup,
}

var restoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "Restore from a backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runRestore,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(connectCmd)
	Cmd.AddCommand(backupCmd)
	Cmd.AddCommand(restoreCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Databases")
	fmt.Println(header)
	fmt.Println()

	// In production: fetch from API
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary)
	fmt.Printf("  %-20s %-14s %-10s %-10s %-10s\n",
		headerStyle.Render("NAME"),
		headerStyle.Render("ENGINE"),
		headerStyle.Render("STORAGE"),
		headerStyle.Render("BACKUP"),
		headerStyle.Render("STATUS"))

	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No databases found. Create one with 'zen db create'."))

	return nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Create Database")
	fmt.Println(header)
	fmt.Println()

	// In production: interactive huh form
	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Database creation wizard coming soon. Use the API directly for now."))

	return nil
}

func runConnect(cmd *cobra.Command, args []string) error {
	dbName := args[0]

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render(fmt.Sprintf("  Connect: %s", dbName))
	fmt.Println(header)
	fmt.Println()

	// In production: look up database, port-forward, launch shell
	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render(
			fmt.Sprintf("Database '%s' not found. Create one with 'zen db create'.", dbName)))

	return nil
}

func runBackup(cmd *cobra.Command, args []string) error {
	dbName := args[0]

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render(fmt.Sprintf("  Backup: %s", dbName))
	fmt.Println(header)
	fmt.Println()

	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render(
			fmt.Sprintf("Database '%s' not found.", dbName)))

	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	dbName := args[0]

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render(fmt.Sprintf("  Restore: %s", dbName))
	fmt.Println(header)
	fmt.Println()

	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render(
			fmt.Sprintf("Database '%s' not found.", dbName)))

	return nil
}

// ShellCommand returns the appropriate database shell command for an engine.
func ShellCommand(engine string) (string, []string) {
	switch strings.ToLower(engine) {
	case "postgresql", "postgres":
		return "psql", []string{"-h", "127.0.0.1"}
	case "mysql":
		return "mysql", []string{"-h", "127.0.0.1"}
	case "redis":
		return "redis-cli", []string{"-h", "127.0.0.1"}
	case "mongodb", "mongo":
		return "mongosh", []string{"--host", "127.0.0.1"}
	default:
		return "", nil
	}
}

// DefaultPort returns the default port for a database engine.
func DefaultPort(engine string) int {
	switch strings.ToLower(engine) {
	case "postgresql", "postgres":
		return 5432
	case "mysql":
		return 3306
	case "redis":
		return 6379
	case "mongodb", "mongo":
		return 27017
	default:
		return 0
	}
}
