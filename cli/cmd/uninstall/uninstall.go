package uninstall

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/dotechhq/zenith/cli/internal/install"
	"github.com/dotechhq/zenith/cli/internal/installstate"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagLocal   bool
	flagSSHHost string
	flagSSHUser string
	flagDir     string
	flagDryRun  bool
)

// Cmd is the `zen uninstall` command — tears down a compose-edition install.
var Cmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Tear down a self-host (compose) FreeZenith install",
	Long: `Stop and remove the self-host (compose) stack and clear local zen state.

  zen uninstall --local
  zen uninstall --ssh-host 1.2.3.4 --ssh-user root`,
	RunE: runUninstall,
}

func init() {
	f := Cmd.Flags()
	f.BoolVar(&flagLocal, "local", false, "Uninstall from this machine")
	f.StringVar(&flagSSHHost, "ssh-host", "", "SSH host/IP of the target")
	f.StringVar(&flagSSHUser, "ssh-user", "root", "SSH user")
	f.StringVar(&flagDir, "dir", "zenith", "Install directory on the target")
	f.BoolVar(&flagDryRun, "dry-run", false, "Show what would happen without doing it")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	if !flagLocal && flagSSHHost == "" {
		return fmt.Errorf("uninstall needs either --local or --ssh-host")
	}
	cfg := &install.Config{
		Edition:      "compose",
		ComposeLocal: flagLocal,
		SSHHost:      flagSSHHost,
		SSHUser:      flagSSHUser,
		InstallDir:   flagDir,
		DryRun:       flagDryRun,
	}
	fmt.Println(lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("Tearing down FreeZenith..."))
	if err := install.ComposeUninstall(cfg); err != nil {
		return err
	}
	if path, err := installstate.DefaultPath(); err == nil {
		_ = os.Remove(path)
	}
	fmt.Println(lipgloss.NewStyle().Foreground(tui.ColorSuccess).
		Render("Uninstalled — stack removed and local state cleared."))
	return nil
}
