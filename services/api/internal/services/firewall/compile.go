// Package firewall turns per-app firewall rules into Kubernetes network policies.
package firewall

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// ErrNeedsCilium is returned for rules standard NetworkPolicy cannot express.
var ErrNeedsCilium = errors.New("deny rules and FQDN egress need Cilium")

const appIDLabel = "zenith.dev/app-id"

var namespacePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`)

// PolicyName is the name of the app's firewall policy.
func PolicyName(appID string) string {
	if len(appID) > 8 {
		appID = appID[:8]
	}
	return "zenith-fw-" + appID
}

// Validate checks a rule before it is stored.
func Validate(r entities.NetworkPolicyRule, cilium bool) error {
	if !cilium && (r.Action == entities.NetworkPolicyDeny || len(r.Config.FQDNs) > 0) {
		return ErrNeedsCilium
	}
	for _, c := range r.Config.CIDRs {
		if _, _, err := net.ParseCIDR(c); err != nil {
			return fmt.Errorf("invalid CIDR %q", c)
		}
	}
	for _, p := range r.Config.Ports {
		if p.Port < 1 || p.Port > 65535 {
			return fmt.Errorf("port %d out of range 1-65535", p.Port)
		}
		if p.Protocol != "" && p.Protocol != entities.NetworkPolicyTCP && p.Protocol != entities.NetworkPolicyUDP {
			return fmt.Errorf("protocol must be TCP or UDP")
		}
	}
	for _, ns := range r.Config.Namespaces {
		if !namespacePattern.MatchString(ns) {
			return fmt.Errorf("invalid namespace %q", ns)
		}
	}
	return nil
}

// Compile builds the app's policy from its enabled rules. It returns nil when
// no rule is enabled, meaning the policy should not exist.
func Compile(app *entities.App, ns string, rules []entities.NetworkPolicyRule, cilium bool) (*k8sclient.CRDObject, error) {
	enabled := make([]entities.NetworkPolicyRule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled {
			if err := Validate(r, cilium); err != nil {
				return nil, fmt.Errorf("rule %q: %w", r.Name, err)
			}
			enabled = append(enabled, r)
		}
	}
	if len(enabled) == 0 {
		return nil, nil
	}
	sort.SliceStable(enabled, func(i, j int) bool { return enabled[i].Priority < enabled[j].Priority })

	var spec map[string]any
	obj := &k8sclient.CRDObject{Metadata: k8sclient.ObjectMeta{
		Name: PolicyName(app.ID), Namespace: ns,
		Labels: map[string]string{appIDLabel: app.ID, "zenith.dev/managed-by": "zenith"},
	}}
	if cilium {
		obj.APIVersion, obj.Kind = "cilium.io/v2", "CiliumNetworkPolicy"
		spec = compileCilium(app, enabled)
	} else {
		obj.APIVersion, obj.Kind = "networking.k8s.io/v1", "NetworkPolicy"
		spec = compileStandard(app, enabled)
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	obj.Spec = raw
	return obj, nil
}

func protocol(p entities.NetworkPolicyPort) string {
	if p.Protocol == "" {
		return string(entities.NetworkPolicyTCP)
	}
	return string(p.Protocol)
}

func nsSelector(ns string) map[string]any {
	return map[string]any{"namespaceSelector": map[string]any{
		"matchLabels": map[string]string{"kubernetes.io/metadata.name": ns}}}
}

func dnsPorts() []map[string]any {
	return []map[string]any{{"protocol": "UDP", "port": 53}, {"protocol": "TCP", "port": 53}}
}

func compileStandard(app *entities.App, rules []entities.NetworkPolicyRule) map[string]any {
	var ingress, egress []map[string]any
	for _, r := range rules {
		var peers []map[string]any
		for _, c := range r.Config.CIDRs {
			peers = append(peers, map[string]any{"ipBlock": map[string]any{"cidr": c}})
		}
		for _, ns := range r.Config.Namespaces {
			peers = append(peers, nsSelector(ns))
		}
		if len(r.Config.PodLabels) > 0 {
			peers = append(peers, map[string]any{"podSelector": map[string]any{"matchLabels": r.Config.PodLabels}})
		}
		entry := map[string]any{}
		if len(r.Config.Ports) > 0 {
			ports := make([]map[string]any, 0, len(r.Config.Ports))
			for _, p := range r.Config.Ports {
				ports = append(ports, map[string]any{"protocol": protocol(p), "port": p.Port})
			}
			entry["ports"] = ports
		}
		if r.Direction == entities.NetworkPolicyEgress {
			if len(peers) > 0 {
				entry["to"] = peers
			}
			egress = append(egress, entry)
		} else {
			if len(peers) > 0 {
				entry["from"] = peers
			}
			ingress = append(ingress, entry)
		}
	}
	spec := map[string]any{"podSelector": map[string]any{"matchLabels": map[string]string{appIDLabel: app.ID}}}
	var types []string
	if len(ingress) > 0 {
		spec["ingress"] = ingress
		types = append(types, "Ingress")
	}
	if len(egress) > 0 {
		egress = append(egress, map[string]any{"to": []map[string]any{nsSelector("kube-system")}, "ports": dnsPorts()})
		spec["egress"] = egress
		types = append(types, "Egress")
	}
	spec["policyTypes"] = types
	return spec
}

func compileCilium(app *entities.App, rules []entities.NetworkPolicyRule) map[string]any {
	buckets := map[string][]map[string]any{}
	needDNSProxy := false
	for _, r := range rules {
		egress := r.Direction == entities.NetworkPolicyEgress
		cidrKey, epKey := "fromCIDR", "fromEndpoints"
		if egress {
			cidrKey, epKey = "toCIDR", "toEndpoints"
		}
		entry := map[string]any{}
		if len(r.Config.CIDRs) > 0 {
			entry[cidrKey] = r.Config.CIDRs
		}
		var eps []map[string]any
		for _, ns := range r.Config.Namespaces {
			eps = append(eps, map[string]any{"matchLabels": map[string]string{"k8s:io.kubernetes.pod.namespace": ns}})
		}
		if len(r.Config.PodLabels) > 0 {
			eps = append(eps, map[string]any{"matchLabels": r.Config.PodLabels})
		}
		if len(eps) > 0 {
			entry[epKey] = eps
		}
		if egress && len(r.Config.FQDNs) > 0 {
			fq := make([]map[string]any, 0, len(r.Config.FQDNs))
			for _, f := range r.Config.FQDNs {
				fq = append(fq, map[string]any{"matchPattern": f})
			}
			entry["toFQDNs"] = fq
			needDNSProxy = true
		}
		if len(r.Config.Ports) > 0 {
			ports := make([]map[string]any, 0, len(r.Config.Ports))
			for _, p := range r.Config.Ports {
				ports = append(ports, map[string]any{"protocol": protocol(p), "port": fmt.Sprint(p.Port)})
			}
			entry["toPorts"] = []map[string]any{{"ports": ports}}
		}
		key := "ingress"
		if egress {
			key = "egress"
		}
		if r.Action == entities.NetworkPolicyDeny {
			key += "Deny"
		}
		buckets[key] = append(buckets[key], entry)
	}
	spec := map[string]any{"endpointSelector": map[string]any{"matchLabels": map[string]string{appIDLabel: app.ID}}}
	if len(buckets["egress"]) > 0 || len(buckets["egressDeny"]) > 0 {
		dns := map[string]any{
			"toEndpoints": []map[string]any{{"matchLabels": map[string]string{"k8s:io.kubernetes.pod.namespace": "kube-system"}}},
			"toPorts":     []map[string]any{{"ports": []map[string]any{{"protocol": "UDP", "port": "53"}, {"protocol": "TCP", "port": "53"}}}},
		}
		if needDNSProxy {
			dns["toPorts"].([]map[string]any)[0]["rules"] = map[string]any{"dns": []map[string]any{{"matchPattern": "*"}}}
		}
		buckets["egress"] = append(buckets["egress"], dns)
	}
	for k, v := range buckets {
		spec[k] = v
	}
	return spec
}
