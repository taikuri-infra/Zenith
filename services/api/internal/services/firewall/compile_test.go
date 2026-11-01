package firewall

import (
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

var testApp = &entities.App{ID: "ab12cd34-0000-0000-0000-000000000000", Name: "api"}

func rule(dir entities.NetworkPolicyDirection, act entities.NetworkPolicyAction, cfg entities.NetworkPolicyConfig) entities.NetworkPolicyRule {
	return entities.NetworkPolicyRule{Direction: dir, Action: act, Enabled: true, Config: cfg}
}

func spec(t *testing.T, rules []entities.NetworkPolicyRule, cilium bool) map[string]any {
	t.Helper()
	obj, err := Compile(testApp, "zp-ab12cd34-site", rules, cilium)
	if err != nil {
		t.Fatal(err)
	}
	if obj == nil {
		t.Fatal("expected a policy")
	}
	if obj.Metadata.Name != "zenith-fw-ab12cd34" || obj.Metadata.Namespace != "zp-ab12cd34-site" {
		t.Fatalf("metadata = %+v", obj.Metadata)
	}
	var m map[string]any
	if err := json.Unmarshal(obj.Spec, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func dig(t *testing.T, v any, path ...any) any {
	t.Helper()
	for _, p := range path {
		switch k := p.(type) {
		case string:
			v = v.(map[string]any)[k]
		case int:
			v = v.([]any)[k]
		}
	}
	return v
}

func TestPolicyName(t *testing.T) {
	if got := PolicyName(testApp.ID); got != "zenith-fw-ab12cd34" {
		t.Fatal(got)
	}
}

func TestCompileNoEnabledRulesReturnsNil(t *testing.T) {
	r := rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{CIDRs: []string{"10.0.0.0/8"}})
	r.Enabled = false
	obj, err := Compile(testApp, "ns", []entities.NetworkPolicyRule{r}, false)
	if err != nil || obj != nil {
		t.Fatalf("got %v, %v", obj, err)
	}
}

func TestCompileIngressCIDRAndPort(t *testing.T) {
	m := spec(t, []entities.NetworkPolicyRule{rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{
		CIDRs: []string{"10.0.0.0/8"}, Ports: []entities.NetworkPolicyPort{{Protocol: "TCP", Port: 443}},
	})}, false)
	if dig(t, m, "podSelector", "matchLabels", "zenith.dev/app-id") != testApp.ID {
		t.Fatal("pod selector")
	}
	if dig(t, m, "ingress", 0, "from", 0, "ipBlock", "cidr") != "10.0.0.0/8" {
		t.Fatal("cidr")
	}
	if dig(t, m, "ingress", 0, "ports", 0, "port") != float64(443) {
		t.Fatal("port")
	}
	if pt := m["policyTypes"].([]any); len(pt) != 1 || pt[0] != "Ingress" {
		t.Fatalf("policyTypes = %v", pt)
	}
}

func TestCompileEgressAddsDNS(t *testing.T) {
	m := spec(t, []entities.NetworkPolicyRule{rule(entities.NetworkPolicyEgress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{CIDRs: []string{"1.2.3.4/32"}})}, false)
	eg := m["egress"].([]any)
	if len(eg) != 2 {
		t.Fatalf("egress = %v", eg)
	}
	dns := eg[1]
	if dig(t, dns, "to", 0, "namespaceSelector", "matchLabels", "kubernetes.io/metadata.name") != "kube-system" {
		t.Fatal("dns target")
	}
	if dig(t, dns, "ports", 0, "port") != float64(53) {
		t.Fatal("dns port")
	}
}

func TestCompileNamespaceAllow(t *testing.T) {
	m := spec(t, []entities.NetworkPolicyRule{rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{Namespaces: []string{"zp-11112222-other"}})}, false)
	if dig(t, m, "ingress", 0, "from", 0, "namespaceSelector", "matchLabels", "kubernetes.io/metadata.name") != "zp-11112222-other" {
		t.Fatal("namespace")
	}
}

func TestCompileCiliumDenyAndFQDN(t *testing.T) {
	obj, err := Compile(testApp, "ns", []entities.NetworkPolicyRule{
		rule(entities.NetworkPolicyIngress, entities.NetworkPolicyDeny, entities.NetworkPolicyConfig{CIDRs: []string{"6.6.6.0/24"}}),
		rule(entities.NetworkPolicyEgress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{FQDNs: []string{"api.stripe.com"}}),
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if obj.APIVersion != "cilium.io/v2" || obj.Kind != "CiliumNetworkPolicy" {
		t.Fatalf("kind = %s %s", obj.APIVersion, obj.Kind)
	}
	var m map[string]any
	_ = json.Unmarshal(obj.Spec, &m)
	if dig(t, m, "ingressDeny", 0, "fromCIDR", 0) != "6.6.6.0/24" {
		t.Fatal("ingressDeny")
	}
	if dig(t, m, "egress", 0, "toFQDNs", 0, "matchPattern") != "api.stripe.com" {
		t.Fatal("fqdn")
	}
	if dig(t, m, "endpointSelector", "matchLabels", "zenith.dev/app-id") != testApp.ID {
		t.Fatal("endpoint selector")
	}
}

func TestValidate(t *testing.T) {
	ok := rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{CIDRs: []string{"10.0.0.0/8"}, Ports: []entities.NetworkPolicyPort{{Port: 80}}})
	if err := Validate(ok, false); err != nil {
		t.Fatalf("good rule: %v", err)
	}
	cases := map[string]entities.NetworkPolicyRule{
		"bad cidr":       rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{CIDRs: []string{"10.0.0.0/33"}}),
		"not cidr":       rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{CIDRs: []string{"abc"}}),
		"port 0":         rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{Ports: []entities.NetworkPolicyPort{{Port: 0}}}),
		"port 70000":     rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{Ports: []entities.NetworkPolicyPort{{Port: 70000}}}),
		"icmp":           rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{Ports: []entities.NetworkPolicyPort{{Protocol: "ICMP", Port: 1}}}),
		"deny no cilium": rule(entities.NetworkPolicyIngress, entities.NetworkPolicyDeny, entities.NetworkPolicyConfig{CIDRs: []string{"1.1.1.1/32"}}),
		"fqdn no cilium": rule(entities.NetworkPolicyEgress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{FQDNs: []string{"a.com"}}),
		"bad namespace":  rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{Namespaces: []string{"Bad_NS"}}),
	}
	for name, r := range cases {
		if err := Validate(r, false); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if err := Validate(rule(entities.NetworkPolicyEgress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{FQDNs: []string{"bad fqdn"}}), true); err == nil {
		t.Error("bad fqdn: expected error")
	}
	if err := Validate(rule(entities.NetworkPolicyEgress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{FQDNs: []string{"*.googleapis.com"}}), true); err != nil {
		t.Errorf("wildcard fqdn: %v", err)
	}
	if err := Validate(cases["deny no cilium"], false); err == nil || err.Error() != ErrNeedsCilium.Error() {
		t.Errorf("deny message = %v", err)
	}
}

func TestCompileCiliumSplitsSelectorTypes(t *testing.T) {
	obj, err := Compile(testApp, "ns", []entities.NetworkPolicyRule{
		rule(entities.NetworkPolicyIngress, entities.NetworkPolicyAllow, entities.NetworkPolicyConfig{
			CIDRs: []string{"10.0.0.0/8"}, Namespaces: []string{"other"},
		}),
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(obj.Spec, &m)
	in := m["ingress"].([]any)
	if len(in) != 2 {
		t.Fatalf("expected one entry per selector type, got %v", in)
	}
	for _, e := range in {
		em := e.(map[string]any)
		if em["fromCIDR"] != nil && em["fromEndpoints"] != nil {
			t.Fatalf("CIDR and endpoints combined in one entry: %v", em)
		}
	}
}
