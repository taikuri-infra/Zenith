package composeops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeInstall(t *testing.T) Install {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  api:\n    image: ghcr.io/taikuri-infra/zenith-api:${ZENITH_VERSION:-latest}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("ZENITH_VERSION=0.1.0-alpha.2\nDB_PASSWORD=old\n"), 0o600)
	return Install{Dir: dir}
}

func TestDetect(t *testing.T) {
	inst := writeInstall(t)
	if got, ok := Detect(inst.Dir); !ok || got.Dir != inst.Dir {
		t.Fatalf("Detect(%q) = %v %v", inst.Dir, got, ok)
	}
	if _, ok := Detect(t.TempDir()); ok {
		t.Fatal("empty dir detected as install")
	}
}

func TestEnvGetSet(t *testing.T) {
	inst := writeInstall(t)
	if inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" {
		t.Fatal("EnvGet")
	}
	if err := inst.EnvSet("DB_PASSWORD", "n|e&w"); err != nil {
		t.Fatal(err)
	}
	if err := inst.EnvSet("NEW_KEY", "v"); err != nil {
		t.Fatal(err)
	}
	if inst.EnvGet("DB_PASSWORD") != "n|e&w" || inst.EnvGet("NEW_KEY") != "v" || inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" {
		t.Fatalf("env after set: %q %q", inst.EnvGet("DB_PASSWORD"), inst.EnvGet("NEW_KEY"))
	}
	if st, _ := os.Stat(filepath.Join(inst.Dir, ".env")); st.Mode().Perm() != 0o600 {
		t.Fatalf(".env mode %v", st.Mode().Perm())
	}
}

func TestDetect_RequiresEnvFile(t *testing.T) {
	inst := writeInstall(t)
	os.Remove(filepath.Join(inst.Dir, ".env"))
	if _, ok := Detect(inst.Dir); ok {
		t.Fatal("a checkout without .env is not an install")
	}
}

func TestEnvUnset(t *testing.T) {
	inst := writeInstall(t)
	if err := inst.EnvUnset("ZENITH_VERSION"); err != nil {
		t.Fatal(err)
	}
	if inst.EnvGet("ZENITH_VERSION") != "" || inst.EnvGet("DB_PASSWORD") != "old" {
		t.Fatal("EnvUnset wrong")
	}
}

func TestDatabases_NameMayContainPipe(t *testing.T) {
	inst := writeInstall(t)
	r := &fakeRunner{answers: map[string]string{"docker compose exec -T postgres psql": "11111111-aaaa|postgresql|my|db\n"}}
	dbs, err := inst.Databases(context.Background(), r)
	if err != nil || len(dbs) != 1 || dbs[0].Name != "my|db" || dbs[0].Engine != "postgresql" {
		t.Fatalf("dbs=%v err=%v", dbs, err)
	}
}
