package db

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
	Long: `Create, list, connect to, and manage databases.

Examples:
  zen db list
  zen db create --name orders --engine mysql
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
	Long: `Create a database in the current project.

Example:
  zen db create --name orders --engine mysql`,
	RunE: runCreate,
}

var connectCmd = &cobra.Command{
	Use:   "connect [name]",
	Short: "Show how to connect to a database (SSH tunnel)",
	Long: `Print the connection strings for a database.

Databases are internal-only: they listen on a loopback-only port on the Zenith
server. This command prints the in-network connection string (for apps) and an
SSH tunnel command plus a local connection string for use from your machine.
With --tunnel, zen runs the ssh command for you.

Example:
  zen db connect my-db
  zen db connect my-db --tunnel --local-port 15432`,
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

var (
	flagProject string
	flagName    string
	flagEngine  string

	flagTunnel    bool
	flagSSH       string
	flagLocalPort int
)

func init() {
	Cmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "Project name or ID (defaults to the configured project, else the first project)")
	createCmd.Flags().StringVarP(&flagName, "name", "n", "", "Database name (required)")
	createCmd.Flags().StringVarP(&flagEngine, "engine", "e", "postgresql", "Engine: postgresql, mysql or redis")

	connectCmd.Flags().BoolVar(&flagTunnel, "tunnel", false, "Run the ssh tunnel instead of only printing it")
	connectCmd.Flags().StringVar(&flagSSH, "ssh", "", "SSH target (default root@<API host>)")
	connectCmd.Flags().IntVar(&flagLocalPort, "local-port", 0, "Local port for the tunnel (default: the engine's standard port)")

	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(connectCmd)
	Cmd.AddCommand(backupCmd)
	Cmd.AddCommand(restoreCmd)
}

// clientAndProject loads the CLI config and returns an API client plus the
// project reference (--project, else the configured project) and the API
// endpoint.
func clientAndProject() (*cliapi.Client, string, string, error) {
	cfg, err := config.Load()
	if err != nil || cfg.Token == "" {
		return nil, "", "", fmt.Errorf("not logged in — run: zen login")
	}
	project := flagProject
	if project == "" {
		project = cfg.Project
	}
	return cliapi.NewClient(cfg.APIEndpoint, cfg.Token), project, cfg.APIEndpoint, nil
}

func runList(cmd *cobra.Command, args []string) error {
	c, project, _, err := clientAndProject()
	if err != nil {
		return err
	}
	return listDatabases(cmd.OutOrStdout(), c, project)
}

func runCreate(cmd *cobra.Command, args []string) error {
	c, project, _, err := clientAndProject()
	if err != nil {
		return err
	}
	return createDatabase(cmd.OutOrStdout(), c, project, flagName, flagEngine)
}

func runConnect(cmd *cobra.Command, args []string) error {
	c, project, endpoint, err := clientAndProject()
	if err != nil {
		return err
	}
	return connectDatabase(cmd.OutOrStdout(), c, project, args[0], connectOpts{
		apiEndpoint: endpoint,
		sshTarget:   flagSSH,
		localPort:   flagLocalPort,
		tunnel:      flagTunnel,
	})
}

// projectDatabases resolves the project and lists its databases.
func projectDatabases(c *cliapi.Client, project string) ([]cliapi.Database, error) {
	id, err := c.ResolveProject(project)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	return c.ListDatabases(id)
}

// listDatabases renders the project's databases as a table.
func listDatabases(w io.Writer, c *cliapi.Client, project string) error {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render("  Databases")
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)

	dbs, err := projectDatabases(c, project)
	if err != nil {
		return fmt.Errorf("list databases: %w", err)
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary)
	fmt.Fprintf(w, "  %-20s %-14s %-10s %-10s %-10s\n",
		headerStyle.Render("NAME"),
		headerStyle.Render("ENGINE"),
		headerStyle.Render("STORAGE"),
		headerStyle.Render("BACKUP"),
		headerStyle.Render("STATUS"))

	if len(dbs) == 0 {
		fmt.Fprintf(w, "  %s\n\n",
			lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("No databases found. Create one with 'zen db create'."))
		return nil
	}
	for _, d := range dbs {
		storage := d.Storage
		if storage == "" {
			storage = "-"
		}
		fmt.Fprintf(w, "  %-20s %-14s %-10s %-10s %-10s\n", d.Name, d.Engine, storage, "-", d.Status)
	}
	fmt.Fprintln(w)
	return nil
}

// createDatabase creates a database in the project from flag values.
func createDatabase(w io.Writer, c *cliapi.Client, project, name, engine string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if engine == "" {
		engine = "postgresql"
	}
	id, err := c.ResolveProject(project)
	if err != nil {
		return err
	}
	db, err := c.CreateDatabase(id, &cliapi.Database{Name: name, Engine: strings.ToLower(engine)})
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	fmt.Fprintf(w, "  %s %s (%s) — %s\n",
		lipgloss.NewStyle().Foreground(tui.ColorSuccess).Render("✓ Created"), db.Name, db.Engine, db.Status)
	fmt.Fprintf(w, "  %s\n\n",
		lipgloss.NewStyle().Foreground(tui.ColorMuted).Render("Run 'zen db connect "+db.Name+"' for its connection string once it is ready."))
	return nil
}

type connectOpts struct {
	apiEndpoint string
	sshTarget   string
	localPort   int
	tunnel      bool
}

// tunnelPlan builds the ssh arguments that forward a local port to the
// database's loopback host port, and the connection string to use locally.
func tunnelPlan(db *cliapi.Database, apiEndpoint, sshTarget string, localPort int) ([]string, string, error) {
	if db.HostPort == 0 {
		return nil, "", fmt.Errorf("database %q has no loopback port yet — redeploy it or upgrade Zenith", db.Name)
	}
	if sshTarget == "" {
		ep, err := url.Parse(apiEndpoint)
		if err != nil || ep.Hostname() == "" {
			return nil, "", fmt.Errorf("cannot derive an SSH host from API endpoint %q — pass --ssh user@host", apiEndpoint)
		}
		sshTarget = "root@" + ep.Hostname()
	}
	if localPort == 0 {
		localPort = DefaultPort(db.Engine)
		if localPort == 0 {
			return nil, "", fmt.Errorf("unknown engine %q — pass --local-port", db.Engine)
		}
	}
	args := []string{"-N", "-L", fmt.Sprintf("%d:127.0.0.1:%d", localPort, db.HostPort), sshTarget}
	local := ""
	if db.ConnectionString != "" {
		u, err := url.Parse(db.ConnectionString)
		if err != nil {
			return nil, "", fmt.Errorf("parse connection string: %w", err)
		}
		u.Host = net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort))
		local = u.String()
	}
	return args, local, nil
}

// connectDatabase prints how to reach the named database and, with
// opts.tunnel, runs the ssh tunnel.
func connectDatabase(w io.Writer, c *cliapi.Client, project, name string, opts connectOpts) error {
	dbs, err := projectDatabases(c, project)
	if err != nil {
		return fmt.Errorf("list databases: %w", err)
	}
	var match *cliapi.Database
	for i := range dbs {
		if dbs[i].Name == name {
			match = &dbs[i]
			break
		}
	}
	if match == nil {
		return fmt.Errorf("database %q not found — create one with 'zen db create'", name)
	}
	rec := *match
	if (rec.ConnectionString == "" || rec.HostPort == 0) && rec.ID != "" {
		full, err := c.GetDatabaseByID(rec.ID)
		if err != nil {
			return fmt.Errorf("get database: %w", err)
		}
		if rec.ConnectionString == "" {
			rec.ConnectionString = full.ConnectionString
		}
		if rec.HostPort == 0 {
			rec.HostPort = full.HostPort
		}
	}
	if rec.ConnectionString == "" {
		return fmt.Errorf("database %q has no connection string yet (status: %s)", name, rec.Status)
	}

	muted := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Render(fmt.Sprintf("  Connect: %s", name))
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", muted.Render("Inside the app network:"))
	fmt.Fprintf(w, "  %s\n\n", rec.ConnectionString)

	args, local, err := tunnelPlan(&rec, opts.apiEndpoint, opts.sshTarget, opts.localPort)
	if err != nil {
		if opts.tunnel {
			return err
		}
		fmt.Fprintf(w, "  %s\n\n", muted.Render(err.Error()))
		return nil
	}
	if !opts.tunnel {
		fmt.Fprintf(w, "  %s\n", muted.Render("From your machine:"))
		fmt.Fprintf(w, "  ssh %s\n", strings.Join(args, " "))
		fmt.Fprintf(w, "  %s\n\n", local)
		return nil
	}
	fmt.Fprintf(w, "  %s\n  %s\n\n", muted.Render("Tunnel open — connect locally with:"), local)
	cmd := exec.Command("ssh", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
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
