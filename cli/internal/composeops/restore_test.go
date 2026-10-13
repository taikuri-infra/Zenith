package composeops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeArchive(t *testing.T, m Manifest, env string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "b.tar.gz")
	f, _ := os.Create(path)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	mj, _ := json.Marshal(m)
	files := map[string]string{"manifest.json": string(mj), "platform.sql": "-- p", "storage.tar": "", "env": env}
	for _, d := range m.Databases {
		ext := ".sql"
		if d.Engine == "redis" {
			ext = ".rdb"
		}
		files["databases/"+d.ID+ext] = "-- " + d.Name
	}
	for n, c := range files {
		tw.WriteHeader(&tar.Header{Name: n, Mode: 0o600, Size: int64(len(c))})
		tw.Write([]byte(c))
	}
	tw.Close()
	gz.Close()
	f.Close()
	return path
}

func TestRestore_ResetsControlPlanePassword(t *testing.T) {
	inst := writeInstall(t) // .env has DB_PASSWORD=old, ZENITH_VERSION=0.1.0-alpha.2
	arch := makeArchive(t, Manifest{Format: 1, Version: "0.1.0-alpha.2"},
		"DB_PASSWORD=fromarchive\nSECRETS_ENCRYPTION_KEY=abc\nZENITH_DOMAIN=old.example.com\nZENITH_VERSION=0.1.0-alpha.1\n")
	r := &fakeRunner{answers: map[string]string{"docker compose ps -q storage": "cid\n", "docker inspect": "vol\n"}}
	if _, err := Restore(context.Background(), r, inst, arch, RestoreOptions{Wait: time.Second, Poll: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if inst.EnvGet("DB_PASSWORD") != "fromarchive" || inst.EnvGet("SECRETS_ENCRYPTION_KEY") != "abc" {
		t.Fatal("secret keys not restored")
	}
	if inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" || inst.EnvGet("ZENITH_DOMAIN") != "" {
		t.Fatal("non-secret keys must stay the target's")
	}
	joined := strings.Join(r.cmds(), "\n")
	if !strings.Contains(joined, "ALTER USER zenith PASSWORD 'fromarchive'") {
		t.Fatalf("control-plane password not reset:\n%s", joined)
	}
	stop := strings.Index(joined, "docker compose stop api web")
	load := strings.Index(joined, "psql -U zenith -d zenith -v ON_ERROR_STOP=1")
	up := strings.Index(joined+"\n", "docker compose up -d\n")
	if stop < 0 || load < 0 || up < 0 || !(stop < load && load < up) {
		t.Fatalf("order wrong (stop %d, load %d, up %d):\n%s", stop, load, up, joined)
	}
}

func TestRestore_LoadsEachDatabaseAfterHealthy(t *testing.T) {
	inst := writeInstall(t)
	m := Manifest{Format: 1, Version: "0.1.0-alpha.2", Databases: []DBRef{
		{ID: "11111111-aaaa", Engine: "postgresql", Name: "pg1"},
		{ID: "33333333-cccc", Engine: "redis", Name: "rd1"},
	}}
	r := &fakeRunner{answers: map[string]string{
		"docker compose ps -q storage": "cid\n", "docker inspect -f {{range": "vol\n",
		"docker inspect -f {{.State.Health.Status}}": "healthy\n",
	}}
	if _, err := Restore(context.Background(), r, inst, makeArchive(t, m, "DB_PASSWORD=x\n"), RestoreOptions{Wait: time.Second, Poll: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(r.cmds(), "\n")
	for _, want := range []string{"docker exec -i zenith-db-11111111", "docker stop zenith-db-33333333", "docker start zenith-db-33333333"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}

func TestRestore_RefusesNewerArchive(t *testing.T) {
	inst := writeInstall(t)
	arch := makeArchive(t, Manifest{Format: 1, Version: "0.2.0"}, "")
	if _, err := Restore(context.Background(), &fakeRunner{}, inst, arch, RestoreOptions{}); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("err = %v", err)
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.1.0-alpha.2", "0.1.0-alpha.10", -1},
		{"0.1.0-alpha.2", "0.1.0", -1},
		{"0.2.0", "0.1.9", 1},
		{"v0.1.0", "0.1.0", 0},
	} {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func rawArchive(t *testing.T, entries [][2]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "raw.tar.gz")
	f, _ := os.Create(path)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		tw.WriteHeader(&tar.Header{Name: e[0], Mode: 0o600, Size: int64(len(e[1]))})
		tw.Write([]byte(e[1]))
	}
	tw.Close()
	gz.Close()
	f.Close()
	return path
}

func mj(m Manifest) string { b, _ := json.Marshal(m); return string(b) }

func TestRestore_IncompleteArchiveRunsNothing(t *testing.T) {
	m := Manifest{Format: 1, Version: "0.1.0-alpha.2", Databases: []DBRef{{ID: "11111111-aaaa", Engine: "postgresql", Name: "pg1"}}}
	cases := map[string][][2]string{
		"platform.sql":                {{"manifest.json", mj(m)}, {"storage.tar", ""}, {"env", ""}, {"databases/11111111-aaaa.sql", "x"}},
		"databases/11111111-aaaa.sql": {{"manifest.json", mj(m)}, {"platform.sql", "x"}, {"storage.tar", ""}, {"env", ""}},
	}
	for missing, entries := range cases {
		inst := writeInstall(t)
		r := &fakeRunner{}
		_, err := Restore(context.Background(), r, inst, rawArchive(t, entries), RestoreOptions{Wait: time.Second, Poll: time.Millisecond})
		if err == nil || !strings.Contains(err.Error(), "archive incomplete: missing "+missing) {
			t.Fatalf("%s: err = %v", missing, err)
		}
		if len(r.cmds()) != 0 {
			t.Fatalf("docker ran: %v", r.cmds())
		}
	}
}

func TestRestore_RejectsBadDatabaseID(t *testing.T) {
	m := Manifest{Format: 1, Version: "0.1.0-alpha.2", Databases: []DBRef{{ID: "x; rm -rf /data #", Engine: "redis", Name: "r"}}}
	r := &fakeRunner{}
	_, err := Restore(context.Background(), r, writeInstall(t), makeArchive(t, m, ""), RestoreOptions{})
	if err == nil || !strings.Contains(err.Error(), "invalid database id") || len(r.cmds()) != 0 {
		t.Fatalf("err = %v, cmds = %v", err, r.cmds())
	}
}

func TestRestore_RejectsTraversal(t *testing.T) {
	for _, name := range []string{"../evil", "/abs/evil"} {
		r := &fakeRunner{}
		arch := rawArchive(t, [][2]string{{name, "x"}})
		_, err := Restore(context.Background(), r, writeInstall(t), arch, RestoreOptions{})
		if err == nil || !strings.Contains(err.Error(), "unsafe path") || len(r.cmds()) != 0 {
			t.Fatalf("%s: err = %v, cmds = %v", name, err, r.cmds())
		}
	}
}

func TestRestore_SecretAllowlist(t *testing.T) {
	inst := writeInstall(t)
	keys := []string{"JWT_SECRET", "SECRETS_ENCRYPTION_KEY", "OLD_SECRETS_KEYS", "DB_PASSWORD", "S3_ACCESS_KEY", "S3_SECRET_KEY", "ADMIN_EMAIL", "ADMIN_PASSWORD"}
	env := "ZENITH_DOMAIN=old.example.com\nDOCKER_GID=999\n"
	for _, k := range keys {
		env += k + "=arch-" + k + "\n"
	}
	r := &fakeRunner{answers: map[string]string{"docker compose ps -q storage": "cid\n", "docker inspect": "vol\n"}}
	if _, err := Restore(context.Background(), r, inst, makeArchive(t, Manifest{Format: 1, Version: "0.1.0-alpha.2"}, env), RestoreOptions{Wait: time.Second, Poll: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if got := inst.EnvGet(k); got != "arch-"+k {
			t.Errorf("%s = %q", k, got)
		}
	}
	if inst.EnvGet("ZENITH_DOMAIN") == "old.example.com" || inst.EnvGet("DOCKER_GID") == "999" {
		t.Fatal("non-listed keys were copied")
	}
}

func TestRestore_OrderingAndFailureCollection(t *testing.T) {
	m := Manifest{Format: 1, Version: "0.1.0-alpha.2", Databases: []DBRef{
		{ID: "11111111-aaaa", Engine: "postgresql", Name: "pg1"},
		{ID: "33333333-cccc", Engine: "redis", Name: "rd1"},
	}}
	answers := map[string]string{
		"docker compose ps -q storage": "cid\n", "docker inspect -f {{range": "vol\n",
		"docker inspect -f {{.State.Health.Status}}": "healthy\n",
	}
	r := &fakeRunner{answers: answers}
	if _, err := Restore(context.Background(), r, writeInstall(t), makeArchive(t, m, "DB_PASSWORD=x\n"), RestoreOptions{Wait: time.Second, Poll: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	j := strings.Join(r.cmds(), "\n")
	idx := func(s string) int { return strings.Index(j, s) }
	h1, l1 := idx("docker inspect -f {{.State.Health.Status}} zenith-db-11111111"), idx("docker exec -i zenith-db-11111111")
	h2, st, rm, sa := idx("docker inspect -f {{.State.Health.Status}} zenith-db-33333333"), idx("docker stop zenith-db-33333333"), idx("rm -rf /data/appendonlydir"), idx("docker start zenith-db-33333333")
	if !(h1 >= 0 && h1 < l1) || !(h2 >= 0 && h2 < st && st < rm && rm < sa) {
		t.Fatalf("ordering wrong:\n%s", j)
	}

	r = &fakeRunner{answers: answers, fail: map[string]bool{"docker exec -i zenith-db-11111111": true}}
	_, err := Restore(context.Background(), r, writeInstall(t), makeArchive(t, m, "DB_PASSWORD=x\n"), RestoreOptions{Wait: time.Second, Poll: time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "pg1") || strings.Contains(err.Error(), "rd1") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(strings.Join(r.cmds(), "\n"), "docker start zenith-db-33333333") {
		t.Fatal("other database not restored")
	}
}

func TestRestore_RedisCopyFailureStillStarts(t *testing.T) {
	m := Manifest{Format: 1, Version: "0.1.0-alpha.2", Databases: []DBRef{{ID: "33333333-cccc", Engine: "redis", Name: "rd1"}}}
	r := &fakeRunner{
		answers: map[string]string{"docker compose ps -q storage": "cid\n", "docker inspect -f {{range": "vol\n", "docker inspect -f {{.State.Health.Status}}": "healthy\n"},
		fail:    map[string]bool{"~rm -rf /data/appendonlydir": true},
	}
	_, err := Restore(context.Background(), r, writeInstall(t), makeArchive(t, m, "DB_PASSWORD=x\n"), RestoreOptions{Wait: time.Second, Poll: time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "rd1") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(strings.Join(r.cmds(), "\n"), "docker start zenith-db-33333333") {
		t.Fatal("container not restarted after copy failure")
	}
}

func TestRestore_MidFailureReportsState(t *testing.T) {
	r := &fakeRunner{fail: map[string]bool{"docker compose exec -T postgres psql -U zenith -d zenith": true},
		answers: map[string]string{"docker compose ps -q storage": "cid\n"}}
	_, err := Restore(context.Background(), r, writeInstall(t), makeArchive(t, Manifest{Format: 1, Version: "0.1.0-alpha.2"}, ""), RestoreOptions{Wait: time.Second, Poll: time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "restore stopped at loading the platform database") || !strings.Contains(err.Error(), "docker compose up -d") {
		t.Fatalf("err = %v", err)
	}
}
