//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

// TestBackupRestore (server C, run with ZENITH_E2E_BACKUP_FILE): restoring A's archive brings back A's admin,
// the probe app, its Postgres and the row.
func TestBackupRestore(t *testing.T) {
	if cfg.BackupFile == "" {
		t.Skip("ZENITH_E2E_BACKUP_FILE not set: not a restore run")
	}
	out, err := serverExec(t, restoreWait, "zen", "backup", "restore", cfg.BackupFile, "--yes", "--dir", "/opt/zenith")
	if err != nil {
		t.Fatalf("zen backup restore: %v\n%s", err, tail(out, 3000))
	}
	waitHealth(t, healthWait)

	// The admin of the backed-up server must log in; server C's own admin proves nothing.
	if cfg.ExpectAdminEmail == "" {
		t.Fatal("ZENITH_E2E_EXPECT_ADMIN_EMAIL is required for a restore run (the backed-up server's admin)")
	}
	if cfg.ExpectAdminPassword == "" {
		t.Fatal("ZENITH_E2E_EXPECT_ADMIN_PASSWORD is required for a restore run (the backed-up server's admin)")
	}
	email, password := cfg.ExpectAdminEmail, cfg.ExpectAdminPassword
	resetAPI()
	c := apiLogin(t, email, password, healthWait)

	app := waitAppRunning(t, c, func(a appInfo) bool { return a.Name == probeAppName }, appRunningWait)
	st := &state{AppID: app.ID, AppName: app.Name, AppURL: strings.TrimRight(app.URL, "/"), DBName: probeDBName, Marker: "alpha-1"}

	var db *dbInfo
	waitFor(t, "database "+probeDBName+" ready", dbReadyWait, func() (bool, string) {
		var dbs []dbInfo
		raw, err := call(t, c, "GET", "/api/v1/apps/"+st.AppID+"/databases", nil, &dbs)
		if err != nil {
			return false, err.Error()
		}
		for i := range dbs {
			if dbs[i].Name == probeDBName {
				db = &dbs[i]
				return db.Status == "ready", "status=" + db.Status
			}
		}
		return false, "not listed: " + raw
	})
	st.DBID = db.ID
	waitProbeRead(t, st, st.Marker, appRunningWait)
}
