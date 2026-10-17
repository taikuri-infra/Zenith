package e2e

import (
	"reflect"
	"strings"
	"testing"
)

var fullEnv = map[string]string{
	"ENDPOINT":              "https://10.0.0.5.sslip.io",
	"DOMAIN":                "10.0.0.5.sslip.io",
	"ADMIN_EMAIL":           "e2e@zenith.local",
	"ADMIN_PASSWORD":        "s3cret",
	"CA_FILE":               "/tmp/a-ca.crt",
	"ZEN":                   "/usr/local/bin/zen",
	"INSTALL_SECONDS":       "321",
	"SERVER_EXEC":           "infra/e2e/server.sh exec a --",
	"REBOOT_CMD":            "infra/e2e/server.sh reboot a",
	"COPY_FROM_SERVER":      "infra/e2e/server.sh cp a",
	"CANDIDATE_VERSION":     "0.1.0-alpha.1-e2e.abc1234",
	"PROBE_IMAGE":           "zenith-e2e-registry:5000/e2e-probe:0.1.0-alpha.1-e2e.abc1234",
	"ARTIFACTS":             "/tmp/art",
	"BACKUP_FILE":           "/opt/zenith/backups/b.tar.gz",
	"EXPECT_ADMIN_EMAIL":    "old@zenith.local",
	"EXPECT_ADMIN_PASSWORD": "old-pw",
}

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k := range fullEnv {
		t.Setenv("ZENITH_E2E_"+k, "")
	}
	for k, v := range env {
		t.Setenv("ZENITH_E2E_"+k, v)
	}
}

func TestLoadFillsEveryField(t *testing.T) {
	setEnv(t, fullEnv)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		Endpoint: "https://10.0.0.5.sslip.io", Domain: "10.0.0.5.sslip.io", AdminEmail: "e2e@zenith.local", AdminPassword: "s3cret",
		CAFile: "/tmp/a-ca.crt", Zen: "/usr/local/bin/zen", InstallSeconds: 321,
		ServerExec:       []string{"infra/e2e/server.sh", "exec", "a", "--"},
		RebootCmd:        []string{"infra/e2e/server.sh", "reboot", "a"},
		CopyFromServer:   []string{"infra/e2e/server.sh", "cp", "a"},
		CandidateVersion: "0.1.0-alpha.1-e2e.abc1234", ProbeImage: "zenith-e2e-registry:5000/e2e-probe:0.1.0-alpha.1-e2e.abc1234",
		Artifacts: "/tmp/art", BackupFile: "/opt/zenith/backups/b.tar.gz", ExpectAdminEmail: "old@zenith.local", ExpectAdminPassword: "old-pw",
	}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("Load() =\n%+v\nwant\n%+v", c, want)
	}
}

func TestLoadOptionalUnset(t *testing.T) {
	setEnv(t, map[string]string{"ENDPOINT": "https://d", "DOMAIN": "d", "ADMIN_EMAIL": "e", "ADMIN_PASSWORD": "p", "ZEN": "zen"})
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.InstallSeconds != 0 || len(c.ServerExec) != 0 || len(c.RebootCmd) != 0 || len(c.CopyFromServer) != 0 || c.BackupFile != "" {
		t.Fatalf("optional fields should be empty: %+v", c)
	}
}

func TestLoadNamesMissingVariable(t *testing.T) {
	for _, k := range []string{"ENDPOINT", "DOMAIN", "ADMIN_EMAIL", "ADMIN_PASSWORD", "ZEN"} {
		env := map[string]string{}
		for kk, v := range fullEnv {
			env[kk] = v
		}
		env[k] = "  "
		setEnv(t, env)
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "ZENITH_E2E_"+k) {
			t.Errorf("missing %s: err = %v", k, err)
		}
	}
}

func TestLoadBadInstallSeconds(t *testing.T) {
	env := map[string]string{}
	for k, v := range fullEnv {
		env[k] = v
	}
	env["INSTALL_SECONDS"] = "ten"
	setEnv(t, env)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ZENITH_E2E_INSTALL_SECONDS") {
		t.Fatalf("err = %v", err)
	}
}

func TestSplitCmd(t *testing.T) {
	if got := SplitCmd("docker exec -w /opt/zenith zenith-e2e-a --"); len(got) != 6 || got[3] != "/opt/zenith" {
		t.Fatalf("SplitCmd = %q", got)
	}
	if got := SplitCmd("  \t "); len(got) != 0 {
		t.Fatalf("empty hook = %q", got)
	}
}
