package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var logoStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#10b981"))

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#6b7280"))

var valueStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f3f4f6"))

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of zen CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(logoStyle.Render("  Zenith"))
		fmt.Println()
		fmt.Printf("  %s %s\n", labelStyle.Render("Version:"), valueStyle.Render(Version))
		fmt.Printf("  %s %s\n", labelStyle.Render("Commit:"), valueStyle.Render(GitCommit))
		fmt.Printf("  %s %s\n", labelStyle.Render("Built:"), valueStyle.Render(BuildTime))
		fmt.Printf("  %s %s\n", labelStyle.Render("Go:"), valueStyle.Render(runtime.Version()))
		fmt.Printf("  %s %s/%s\n", labelStyle.Render("OS/Arch:"), valueStyle.Render(runtime.GOOS), valueStyle.Render(runtime.GOARCH))
		if cfg, err := config.Load(); err == nil && cfg.APIEndpoint != "" {
			fmt.Printf("  %s %s\n", labelStyle.Render("Server: "), valueStyle.Render(serverVersion(cfg.APIEndpoint)))
		}
	},
}

func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    GitCommit,
		"buildTime": BuildTime,
		"go":        runtime.Version(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	}
}

// serverVersion asks the API for its version; "unreachable" on any failure.
func serverVersion(endpoint string) string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(strings.TrimRight(endpoint, "/") + "/api/v1/version")
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	var body struct {
		Version string `json:"version"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&body) != nil || body.Version == "" {
		return "unreachable"
	}
	return body.Version
}
