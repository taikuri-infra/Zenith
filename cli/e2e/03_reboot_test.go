//go:build e2e

package e2e

import (
	"testing"
)

// TestReboot: after a server reboot the app, its data, the database and the domain come back on their own.
func TestReboot(t *testing.T) {
	st := requireDeployState(t)
	if len(cfg.RebootCmd) == 0 {
		t.Fatal("ZENITH_E2E_REBOOT_CMD is not set")
	}
	if out, err := runCmd(t, rebootCmdWait, nil, cfg.RebootCmd...); err != nil {
		t.Fatalf("reboot: %v\n%s", err, tail(out, 3000))
	}
	waitHealth(t, healthWait)
	resetAPI()
	c := apiLogin(t, cfg.AdminEmail, cfg.AdminPassword, healthWait)

	waitAppRunning(t, c, func(a appInfo) bool { return a.ID == st.AppID }, appRunningWait)
	waitProbeRead(t, st, st.Marker, appRunningWait)
	waitDBReady(t, c, st.AppID, st.DBID, dbReadyWait)
	if st.DomainID != "" {
		waitDomainActive(t, c, st.AppID, st.DomainID, domainWait)
	} else {
		t.Errorf("no domain in state: the domain step of TestDeployWithDatabaseAndDomain did not run")
	}
	zenStatus(t, st.AppName, st.DBName)
}
