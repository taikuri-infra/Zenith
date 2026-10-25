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

// TestBackup (server A): zen backup writes an archive and the hook copies it into the artifacts.
func TestBackup(t *testing.T) {
	requireDeployState(t) // never archive an install without the probe app and its row
	out, err := serverExec(t, backupWait, "zen", "backup", "--out", filepath.Join(cfg.Artifacts, "server-backups"))
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
