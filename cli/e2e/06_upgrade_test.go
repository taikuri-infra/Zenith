//go:build e2e

package e2e

import (
	"testing"
)

// TestUpgrade (server B, installed from the previous release): zen upgrade to the candidate keeps the app and its data.
func TestUpgrade(t *testing.T) {
	if cfg.CandidateVersion == "" {
		t.Fatal("ZENITH_E2E_CANDIDATE_VERSION is not set")
	}
	c := api(t)
	before, err := c.ServerInfo()
	logf(t, "before upgrade: %+v %v", before, err)
	st := deployProbe(t, c, "before-upgrade")

	out, err := serverExec(t, upgradeWait, "zen", "upgrade", "--version", cfg.CandidateVersion, "--dir", "/opt/zenith")
	if errNotNewer(out, err) {
		logf(t, "zen upgrade says the candidate is not newer than the install; retrying with --force")
		out, err = serverExec(t, upgradeWait, "zen", "upgrade", "--version", cfg.CandidateVersion, "--dir", "/opt/zenith", "--force")
	}
	if err != nil {
		t.Fatalf("zen upgrade: %v\n%s", err, tail(out, 3000))
	}
	waitHealth(t, healthWait)
	resetAPI()
	c = apiLogin(t, cfg.AdminEmail, cfg.AdminPassword, healthWait)
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
