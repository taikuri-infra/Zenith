package k8sops

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func inst() Install { return Install{Namespace: "zenith-system"} }

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// stack answers the kubectl calls a healthy install gives.
func stack() *fakeRunner {
	const platformPod = "zenith-postgres-1"
	return &fakeRunner{answers: map[string]string{
		"kubectl -n zenith-system get deploy zenith-api -o jsonpath={.spec.template.spec.containers[0].image}": "ghcr.io/taikuri-infra/zenith-api:0.2.0",
		"kubectl -n zenith-system get pods -l cnpg.io/cluster=zenith-postgres,cnpg.io/instanceRole=primary":    platformPod,
		"kubectl -n zenith-system exec " + platformPod + " -c postgres -- psql":                                "11111111-aaaa|postgresql|orders|db_orders|u_orders|zenith-apps\n",
		"kubectl -n zenith-system exec " + platformPod + " -c postgres -- pg_dump":                             "-- platform dump\n",
		"kubectl -n zenith-system exec zenith-storage-0 -- tar":                                                "TARDATA",
		"kubectl -n zenith-system get secret zenith-secrets -o json":                                           `{"metadata":{"name":"zenith-secrets","resourceVersion":"9","uid":"x"},"type":"Opaque","data":{"jwt-secret":"` + b64("j") + `"}}`,
		"kubectl -n zenith-apps get pods -l cnpg.io/cluster=db-11111111,cnpg.io/instanceRole=primary":          "db-11111111-1",
		"kubectl -n zenith-apps exec db-11111111-1 -c postgres -- pg_dump":                                     "-- orders dump\n",
	}}
}

func TestEmbeddedInstallerMatchesRepoScript(t *testing.T) {
	repo, err := os.ReadFile("../../../infra/scripts/install-k3s.sh")
	if err != nil {
		t.Skip("repository script not available:", err)
	}
	if !bytes.Equal(repo, installScript) {
		t.Fatal("cli/internal/k8sops/install-k3s.sh differs from infra/scripts/install-k3s.sh; copy it over")
	}
}

func TestBackupWritesEverythingTheRestoreNeeds(t *testing.T) {
	r := stack()
	out := t.TempDir()
	path, err := Backup(context.Background(), r, inst(), out, time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.Base(path), "zenith-backup-0.2.0-20261025T120000Z") {
		t.Fatalf("archive name %q", path)
	}
	dst := t.TempDir()
	if err := extractArchive(path, dst); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"manifest.json", "platform.sql", "storage.tar", "secrets.json", "databases/11111111-aaaa.sql"} {
		if _, err := os.Stat(filepath.Join(dst, f)); err != nil {
			t.Errorf("archive lacks %s", f)
		}
	}
	m, err := readManifest(dst)
	if err != nil || m.Version != "0.2.0" || len(m.Databases) != 1 || m.Databases[0].DBName != "db_orders" {
		t.Fatalf("manifest = %+v, %v", m, err)
	}
	if entries, _ := os.ReadDir(out); len(entries) != 1 {
		t.Errorf("temp files left in the output dir: %v", entries)
	}
}

func TestBackupFailureLeavesNoArchive(t *testing.T) {
	r := stack()
	r.fail = map[string]bool{"kubectl -n zenith-system exec zenith-storage-0": true}
	out := t.TempDir()
	if _, err := Backup(context.Background(), r, inst(), out, time.Now()); err == nil {
		t.Fatal("expected an error")
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestRestoreReplacesDataInOrder(t *testing.T) {
	r := stack()
	path, err := Backup(context.Background(), r, inst(), t.TempDir(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.calls = nil })
	r2 := stack()
	r2.answers["kubectl -n zenith-system get deploy zenith-api -o jsonpath={.status.replicas}"] = "0"
	r2.answers[`kubectl -n zenith-apps get pod db-11111111-1 -o jsonpath={.status.conditions[?(@.type=="Ready")].status}`] = "True"

	m, err := Restore(context.Background(), r2, inst(), path, RestoreOptions{Wait: time.Second, Poll: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Databases) != 1 {
		t.Fatalf("manifest %+v", m)
	}
	cmds := strings.Join(r2.cmds(), "\n")
	want := []string{
		"scale deploy/zenith-api --replicas=0",
		"apply -f -",
		"psql -U postgres -d zenith -v ON_ERROR_STOP=1",
		"tar -C /data -xf -",
		"rollout restart statefulset/zenith-storage",
		"scale deploy/zenith-api --replicas=1",
		"rollout status deploy/zenith-api",
		"psql -U postgres -d db_orders",
	}
	last := -1
	for _, w := range want {
		idx := strings.Index(cmds, w)
		if idx < 0 || idx < last {
			t.Fatalf("%q missing or out of order in:\n%s", w, cmds)
		}
		last = idx
	}
	var applied string
	for k, v := range r2.stdins {
		if strings.HasSuffix(k, "apply -f -") {
			applied = v
		}
	}
	if !strings.Contains(applied, `"name":"zenith-secrets"`) || strings.Contains(applied, "resourceVersion") {
		t.Fatalf("applied secret = %s", applied)
	}
	for k, v := range r2.stdins {
		if strings.Contains(k, "-d zenith ") && !strings.HasPrefix(v, "SET ROLE zenith;\n") {
			t.Fatalf("platform restore must run as the zenith role: %q", v[:20])
		}
	}
}

func TestRestoreRefusesNewerArchive(t *testing.T) {
	r := stack()
	path, err := Backup(context.Background(), r, inst(), t.TempDir(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r2 := stack()
	r2.answers["kubectl -n zenith-system get deploy zenith-api -o jsonpath={.spec.template.spec.containers[0].image}"] = "x/zenith-api:0.1.0"
	if _, err := Restore(context.Background(), r2, inst(), path, RestoreOptions{}); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("err = %v", err)
	}
	for _, c := range r2.cmds() {
		if strings.Contains(c, "scale") {
			t.Fatal("nothing may be touched before the version check")
		}
	}
}

func TestUpgradeUsesExistingReleaseSettings(t *testing.T) {
	r := stack()
	r.answers["helm get values zenith -n zenith-system -o json"] = `{"domain":"z.example.com","images":{"api":"reg.local/zenith-api:0.2.0","pullPolicy":"Never"},"tls":{"issuerName":"ca","caSecret":"my-ca-secret","issuer":{"create":false,"email":"a@b.c"}}}`
	var got map[string]string
	old := runScript
	runScript = func(_ context.Context, env map[string]string, _ string, _ io.Writer) error { got = env; return nil }
	t.Cleanup(func() { runScript = old })

	res, err := Upgrade(context.Background(), r, inst(), UpgradeOptions{Target: "0.3.0", SkipBackup: true}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if res.From != "0.2.0" || res.To != "0.3.0" {
		t.Fatalf("res = %+v", res)
	}
	for k, v := range map[string]string{
		"ZENITH_DOMAIN": "z.example.com", "ZENITH_VERSION": "0.3.0", "ZENITH_BRANCH": "v0.3.0",
		"ZENITH_IMAGE_REPO": "reg.local", "ZENITH_PULL_POLICY": "Never", "ZENITH_TLS_ISSUER": "ca", "ZENITH_TLS_CA_SECRET": "my-ca-secret", "ZENITH_EMAIL": "a@b.c",
	} {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestUpgradeRollsBackOnFailure(t *testing.T) {
	r := stack()
	r.answers["helm get values zenith -n zenith-system -o json"] = `{"domain":"z.example.com"}`
	old := runScript
	runScript = func(context.Context, map[string]string, string, io.Writer) error { return errors.New("helm timed out") }
	t.Cleanup(func() { runScript = old })

	res, err := Upgrade(context.Background(), r, inst(), UpgradeOptions{Target: "0.3.0", SkipBackup: true}, io.Discard)
	if err == nil || !res.RolledBack {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !strings.Contains(strings.Join(r.cmds(), "\n"), "helm rollback zenith -n zenith-system") {
		t.Fatal("no helm rollback")
	}
}

func TestUpgradeRefusesDowngradeWithoutForce(t *testing.T) {
	r := stack()
	if _, err := Upgrade(context.Background(), r, inst(), UpgradeOptions{Target: "0.1.0", SkipBackup: true}, io.Discard); err == nil {
		t.Fatal("expected refusal")
	}
	if _, err := Upgrade(context.Background(), r, inst(), UpgradeOptions{Target: "not-a-version"}, io.Discard); err == nil {
		t.Fatal("expected invalid version error")
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"0.1.0", "0.1.0", 0}, {"0.2.0", "0.1.9", 1}, {"v0.1.0-alpha.1", "0.1.0", -1}, {"0.1.0-alpha.2", "0.1.0-alpha.10", -1}} {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%s,%s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestDatabaseRowPatternsRejectInjection(t *testing.T) {
	for _, ns := range []string{"zenith-system", "kube-system", "zp-a|zenith-apps", "zp-", "ZP-x"} {
		if dbNSPattern.MatchString(ns) {
			t.Errorf("namespace %q must be rejected", ns)
		}
	}
	for _, ns := range []string{"zenith-apps", "zp-1a2b3c4d-site"} {
		if !dbNSPattern.MatchString(ns) {
			t.Errorf("namespace %q must be accepted", ns)
		}
	}
	if dbIDPattern.MatchString("abc; rm -rf") {
		t.Error("bad id accepted")
	}
}
