//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestProjectNamespaceIsolation: the deployed probe app and its database run in the
// project's own namespace (not the shared legacy one), which carries the isolation policy.
func TestProjectNamespaceIsolation(t *testing.T) {
	st := requireDeployState(t)
	_ = st
	out, err := serverExec(t, 60*time.Second, "kubectl", "get", "namespace", "-l", "zenith.dev/managed=true", "-o", "name")
	if err != nil {
		t.Fatalf("list project namespaces: %v %s", err, out)
	}
	nss := strings.Fields(out)
	if len(nss) == 0 {
		t.Fatalf("no project namespace was created; got %q", out)
	}
	ns := strings.TrimPrefix(nss[0], "namespace/")
	if !strings.HasPrefix(ns, "zp-") {
		t.Fatalf("unexpected namespace %q", ns)
	}
	if out, err := serverExec(t, 60*time.Second, "kubectl", "-n", ns, "get", "networkpolicy", "project-isolation"); err != nil {
		t.Fatalf("isolation policy missing in %s: %v %s", ns, err, out)
	}
	deps, err := serverExec(t, 60*time.Second, "kubectl", "-n", ns, "get", "deployments", "-o", "name")
	if err != nil || strings.TrimSpace(deps) == "" {
		t.Fatalf("no app deployment in %s: %v %s", ns, err, deps)
	}
	legacy, _ := serverExec(t, 60*time.Second, "kubectl", "-n", "zenith-apps", "get", "deployments", "-o", "name")
	if strings.TrimSpace(legacy) != "" {
		t.Fatalf("workloads leaked into the shared namespace: %s", legacy)
	}
}
