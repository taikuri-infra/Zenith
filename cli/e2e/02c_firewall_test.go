//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestFirewallRuleOpensAndClosesTraffic: a pod outside the project cannot reach the
// app (project isolation); an ingress allow rule for its namespace opens the path,
// disabling the rule closes it again, and deny rules are refused without Cilium.
func TestFirewallRuleOpensAndClosesTraffic(t *testing.T) {
	st := requireDeployState(t)
	c := api(t)
	const probeNS = "fw-e2e-probe"

	nsOut, err := serverExec(t, 60*time.Second, "kubectl", "get", "deploy", "-A", "-l", "zenith.dev/app-id="+st.AppID,
		"-o", "jsonpath={.items[0].metadata.namespace} {.items[0].metadata.labels.app}")
	parts := strings.Fields(nsOut)
	if err != nil || len(parts) != 2 {
		t.Fatalf("find app deployment: %v %q", err, nsOut)
	}
	appNS, svc := parts[0], parts[1]
	target := "http://" + svc + "." + appNS + ".svc.cluster.local/"

	if out, err := serverExec(t, 60*time.Second, "kubectl", "create", "namespace", probeNS); err != nil && !strings.Contains(out, "AlreadyExists") {
		t.Fatalf("create probe namespace: %v %s", err, out)
	}
	t.Cleanup(func() { _, _ = serverExec(t, 120*time.Second, "kubectl", "delete", "namespace", probeNS, "--wait=false") })

	// reachable runs a one-shot curl from the probe namespace.
	reachable := func() (bool, string) {
		name := "probe-" + time.Now().Format("150405")
		out, err := serverExec(t, 90*time.Second, "kubectl", "-n", probeNS, "run", name, "--rm", "-i", "--restart=Never",
			"--image=curlimages/curl:8.10.1", "--", "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "5", target)
		return err == nil && strings.Contains(out, "200"), out
	}

	if ok, out := reachable(); ok {
		t.Fatalf("app reachable from another namespace before any rule: %s", out)
	}

	var rule struct {
		ID string `json:"id"`
	}
	path := "/api/v1/apps/" + st.AppID + "/network-policies"
	if raw, err := call(t, c, "POST", path, map[string]any{
		"name": "allow-probe", "direction": "ingress", "action": "allow",
		"config": map[string]any{"namespaces": []string{probeNS}},
	}, &rule); err != nil {
		t.Fatalf("create rule: %v %s", err, raw)
	}
	policy := "zenith-fw-" + st.AppID[:8]
	if out, err := serverExec(t, 60*time.Second, "kubectl", "-n", appNS, "get", "networkpolicy", policy); err != nil {
		t.Fatalf("policy %s missing in %s: %v %s", policy, appNS, err, out)
	}
	waitFor(t, "probe reaches app after allow rule", 2*time.Minute, reachable)

	if raw, err := call(t, c, "PUT", path+"/"+rule.ID, map[string]any{"enabled": false}, nil); err != nil {
		t.Fatalf("disable rule: %v %s", err, raw)
	}
	waitFor(t, "policy removed after disabling", time.Minute, func() (bool, string) {
		out, err := serverExec(t, 30*time.Second, "kubectl", "-n", appNS, "get", "networkpolicy", policy)
		return err != nil, out
	})
	waitFor(t, "probe blocked again", 2*time.Minute, func() (bool, string) {
		ok, out := reachable()
		return !ok, out
	})

	if _, err := call(t, c, "POST", path, map[string]any{
		"name": "deny", "direction": "ingress", "action": "deny",
		"config": map[string]any{"cidrs": []string{"1.1.1.1/32"}},
	}, nil); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("deny rule without Cilium: expected 400, got %v", err)
	}
}
