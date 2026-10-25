//go:build e2e

package e2e

import (
	"testing"
)

// TestUpgrade: zen upgrade backs up, re-applies the release (here the candidate itself, since no
// earlier release was ever published for Kubernetes) and keeps the app and its data.
func TestUpgrade(t *testing.T) {
	if cfg.CandidateVersion == "" {
		t.Fatal("ZENITH_E2E_CANDIDATE_VERSION is not set")
	}
	st := requireDeployState(t)

	out, err := serverExec(t, upgradeWait, "zen", "upgrade", "--version", cfg.CandidateVersion, "--force")
	if err != nil {
		t.Fatalf("zen upgrade: %v\n%s", err, tail(out, 3000))
	}
	waitHealth(t, healthWait)
	resetAPI()
	c := apiLogin(t, cfg.AdminEmail, cfg.AdminPassword, healthWait)
	waitFor(t, "version "+cfg.CandidateVersion, healthWait, func() (bool, string) {
		info, err := c.ServerInfo()
		if err != nil {
			return false, err.Error()
		}
		return info.Version == cfg.CandidateVersion, "version=" + info.Version + " mode=" + info.Mode
	})
	waitAppRunning(t, c, func(a appInfo) bool { return a.ID == st.AppID }, appRunningWait)
	waitProbeRead(t, st, st.Marker, appRunningWait)
	zenStatus(t, st.AppName)
}
