package composeops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func archiveEntries(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		out[h.Name] = string(b)
	}
}

func TestBackup_WritesEveryPart(t *testing.T) {
	inst := writeInstall(t)
	r := &fakeRunner{answers: map[string]string{
		"docker compose exec -T postgres psql":    "11111111-aaaa|postgresql|pg1\n22222222-bbbb|mysql|my1\n33333333-cccc|redis|rd1\n",
		"docker compose exec -T postgres pg_dump": "-- platform dump",
		"docker compose ps -q storage":            "cid-storage\n",
		"docker inspect":                          "zenith_storagedata\n",
		"docker exec zenith-db-11111111 pg_dump":  "-- pg1 dump",
		"docker exec zenith-db-22222222 sh -c":    "-- my1 dump",
	}}
	out := t.TempDir()
	path, err := Backup(context.Background(), r, inst, out, time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "zenith-backup-0.1.0-alpha.2-20261013T080000Z.tar.gz") {
		t.Fatalf("path = %s", path)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Fatalf("archive mode %v", st.Mode().Perm())
	}
	e := archiveEntries(t, path)
	for _, name := range []string{"manifest.json", "platform.sql", "storage.tar", "env", "databases/11111111-aaaa.sql", "databases/22222222-bbbb.sql", "databases/33333333-cccc.rdb"} {
		if _, ok := e[name]; !ok {
			t.Errorf("archive lacks %s (has %v)", name, keys(e))
		}
	}
	if e["platform.sql"] != "-- platform dump" || !strings.Contains(e["env"], "DB_PASSWORD=old") {
		t.Errorf("content wrong: platform=%q env=%q", e["platform.sql"], e["env"])
	}
	var m Manifest
	if err := json.Unmarshal([]byte(e["manifest.json"]), &m); err != nil || m.Version != "0.1.0-alpha.2" || len(m.Databases) != 3 || m.Format != 1 {
		t.Fatalf("manifest = %+v err %v", m, err)
	}
}

func TestBackup_DumpFailureLeavesNoArchive(t *testing.T) {
	inst := writeInstall(t)
	r := &fakeRunner{
		answers: map[string]string{"docker compose exec -T postgres psql": "11111111-aaaa|postgresql|pg1\n"},
		fail:    map[string]bool{"docker exec zenith-db-11111111 pg_dump": true},
	}
	out := t.TempDir()
	if _, err := Backup(context.Background(), r, inst, out, time.Now()); err == nil {
		t.Fatal("want error")
	}
	if left, _ := os.ReadDir(out); len(left) != 0 {
		t.Fatalf("partial files left: %v", left)
	}
}

func keys(m map[string]string) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	return k
}
