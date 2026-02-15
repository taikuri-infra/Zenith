package version

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/lipgloss"
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
