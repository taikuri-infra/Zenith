//go:build e2e

package e2e

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// archivePath finds the archive zen backup printed: the last path-like token on a line naming zenith-backup-*.tar.gz.
func archivePath(out string) string {
	found := ""
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "zenith-backup-") || !strings.Contains(line, ".tar.gz") {
			continue
		}
		for _, f := range strings.Fields(line) {
			if strings.Contains(f, "zenith-backup-") && strings.HasSuffix(f, ".tar.gz") {
				found = f
			}
		}
	}
	return found
}

// TestBackup (server A): zen backup writes an archive on the server and the hook copies it into the artifacts.
func TestBackup(t *testing.T) {
	out, err := serverExec(t, backupWait, "zen", "backup", "--dir", "/opt/zenith")
	if err != nil {
		t.Fatalf("zen backup: %v\n%s", err, tail(out, 3000))
	}
	remote := archivePath(out)
	if remote == "" {
		t.Fatalf("no zenith-backup-*.tar.gz path in zen backup output:\n%s", out)
	}
	if len(cfg.CopyFromServer) == 0 {
		t.Fatal("ZENITH_E2E_COPY_FROM_SERVER is not set")
	}
	local := filepath.Join(cfg.Artifacts, path.Base(remote))
	if out, err := runCmd(t, copyWait, nil, append(append([]string{}, cfg.CopyFromServer...), remote, local)...); err != nil {
		t.Fatalf("copy %s: %v\n%s", remote, err, out)
	}
	fi, err := os.Stat(local)
	if err != nil {
		t.Fatalf("archive not copied: %v", err)
	}
	if fi.Size() <= 1024 {
		t.Fatalf("archive %s is %d bytes (want > 1 KB)", local, fi.Size())
	}
	logf(t, "archive %s (%d bytes)", local, fi.Size())
}

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

	email, password := cfg.ExpectAdminEmail, cfg.ExpectAdminPassword
	if email == "" {
		email = cfg.AdminEmail
	}
	if password == "" {
		password = cfg.AdminPassword
	}
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
