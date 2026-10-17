package e2e

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is everything the suite needs; the harness provides it as environment variables.
type Config struct {
	Endpoint, Domain, AdminEmail, AdminPassword, CAFile, Zen string
	InstallSeconds                                           int
	ServerExec, RebootCmd, CopyFromServer                    []string
	CandidateVersion, ProbeImage, Artifacts                  string
	BackupFile, ExpectAdminEmail, ExpectAdminPassword        string
}

// Load reads ZENITH_E2E_* and reports the first missing required variable.
func Load() (Config, error) {
	get := func(k string) string { return strings.TrimSpace(os.Getenv("ZENITH_E2E_" + k)) }
	c := Config{Endpoint: get("ENDPOINT"), Domain: get("DOMAIN"), AdminEmail: get("ADMIN_EMAIL"), AdminPassword: get("ADMIN_PASSWORD"),
		CAFile: get("CA_FILE"), Zen: get("ZEN"), CandidateVersion: get("CANDIDATE_VERSION"), ProbeImage: get("PROBE_IMAGE"),
		Artifacts: get("ARTIFACTS"), BackupFile: get("BACKUP_FILE"), ExpectAdminEmail: get("EXPECT_ADMIN_EMAIL"), ExpectAdminPassword: get("EXPECT_ADMIN_PASSWORD"),
		ServerExec: SplitCmd(get("SERVER_EXEC")), RebootCmd: SplitCmd(get("REBOOT_CMD")), CopyFromServer: SplitCmd(get("COPY_FROM_SERVER"))}
	for _, req := range [][2]string{{"ENDPOINT", c.Endpoint}, {"DOMAIN", c.Domain}, {"ADMIN_EMAIL", c.AdminEmail}, {"ADMIN_PASSWORD", c.AdminPassword}, {"ZEN", c.Zen}} {
		if req[1] == "" {
			return c, fmt.Errorf("ZENITH_E2E_%s is required", req[0])
		}
	}
	if s := get("INSTALL_SECONDS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return c, fmt.Errorf("ZENITH_E2E_INSTALL_SECONDS: %w", err)
		}
		c.InstallSeconds = n
	}
	if c.Zen == "" {
		c.Zen = "zen"
	}
	return c, nil
}

// SplitCmd splits a hook on whitespace; an empty string is no hook.
func SplitCmd(s string) []string { return strings.Fields(s) }
