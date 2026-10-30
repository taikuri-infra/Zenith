package firewall

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// AppGetter is the part of the app repository the applier needs.
type AppGetter interface {
	GetApp(ctx context.Context, id string) (*entities.App, error)
}

// Applier keeps each app's firewall policy in the cluster in sync with its rules.
type Applier struct {
	repo  ports.FirewallRuleRepository
	apps  AppGetter
	k8s   k8sclient.Client
	nsFor func(ctx context.Context, projectID string) string

	mu          sync.Mutex
	ciliumKnown bool
	cilium      bool
}

// NewApplier creates an Applier.
func NewApplier(repo ports.FirewallRuleRepository, apps AppGetter, k8s k8sclient.Client, nsFor func(context.Context, string) string) *Applier {
	return &Applier{repo: repo, apps: apps, k8s: k8s, nsFor: nsFor}
}

// Cilium reports whether the cluster has the CiliumNetworkPolicy CRD. When the
// probe is inconclusive it answers false, the stricter choice for validation.
func (a *Applier) Cilium(ctx context.Context) bool {
	ok, _ := a.ciliumState()
	return ok
}

// ciliumState probes for Cilium once it gets a definite answer; a transient
// API error is returned (and retried on the next call), never cached.
func (a *Applier) ciliumState() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ciliumKnown {
		return a.cilium, nil
	}
	if _, mem := a.k8s.(*k8sclient.MemoryClient); mem {
		a.ciliumKnown, a.cilium = true, false
		return false, nil
	}
	cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := a.k8s.ListCRDsWithVersion(cctx, "cilium.io/v2", "CiliumNetworkPolicy", "kube-system")
	switch {
	case err == nil:
		a.ciliumKnown, a.cilium = true, true
	case isNotFound(err):
		a.ciliumKnown, a.cilium = true, false
	default:
		return false, fmt.Errorf("detect cilium: %w", err)
	}
	return a.cilium, nil
}

func isNotFound(err error) bool {
	if apierrors.IsNotFound(err) {
		return true
	}
	// The memory client and wrapped dynamic-client errors only carry text.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "could not find the requested resource")
}

// policyKeys lists every rule section a policy may have; absent ones are sent
// as null so a merge patch removes sections that no longer have rules.
var policyKeys = map[string][]string{
	"NetworkPolicy":       {"ingress", "egress"},
	"CiliumNetworkPolicy": {"ingress", "egress", "ingressDeny", "egressDeny"},
}

// Apply compiles the app's rules and creates, updates or deletes its policy.
func (a *Applier) Apply(ctx context.Context, appID string) error {
	app, err := a.apps.GetApp(ctx, appID)
	if err != nil {
		return fmt.Errorf("app not found: %w", err)
	}
	ns := a.nsFor(ctx, app.ProjectID)
	rules, err := a.repo.ListByApp(ctx, appID)
	if err != nil {
		return err
	}
	cilium, err := a.ciliumState()
	if err != nil {
		return err // unsure which policy kind applies: change nothing
	}
	obj, err := Compile(app, ns, rules, cilium)
	if err != nil {
		return err
	}
	name := PolicyName(app.ID)
	if obj == nil {
		a.remove(ctx, ns, name, "networking.k8s.io/v1", "NetworkPolicy")
		a.remove(ctx, ns, name, "cilium.io/v2", "CiliumNetworkPolicy")
		return nil
	}
	if err := a.upsert(ctx, obj); err != nil {
		return err
	}
	// Only one policy kind may exist for the app; drop the other one once the
	// new policy is in place.
	if obj.Kind == "NetworkPolicy" {
		a.remove(ctx, ns, name, "cilium.io/v2", "CiliumNetworkPolicy")
	} else {
		a.remove(ctx, ns, name, "networking.k8s.io/v1", "NetworkPolicy")
	}
	return nil
}

// upsert creates the policy, or merge-patches it when it already exists.
func (a *Applier) upsert(ctx context.Context, obj *k8sclient.CRDObject) error {
	if _, err := a.k8s.GetCRDWithVersion(ctx, obj.APIVersion, obj.Kind, obj.Metadata.Namespace, obj.Metadata.Name); err != nil {
		if !isNotFound(err) {
			return err
		}
		return a.k8s.CreateCRD(ctx, obj)
	}
	var spec map[string]any
	if err := json.Unmarshal(obj.Spec, &spec); err != nil {
		return err
	}
	for _, k := range policyKeys[obj.Kind] {
		if _, ok := spec[k]; !ok {
			spec[k] = nil
		}
	}
	patch := *obj
	raw, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	patch.Spec = raw
	if _, mem := a.k8s.(*k8sclient.MemoryClient); mem {
		patch.Spec = obj.Spec // the memory client stores the object as-is
	}
	return a.k8s.PatchCRD(ctx, &patch)
}

func (a *Applier) remove(ctx context.Context, ns, name, apiVersion, kind string) {
	err := a.k8s.DeleteCRDWithVersion(ctx, apiVersion, kind, ns, name)
	if err != nil && !isNotFound(err) {
		slog.Warn("firewall policy not deleted", "namespace", ns, "name", name, "kind", kind, "error", err)
	}
}

// ReconcileAll re-applies the policy of every app that has firewall rules.
func (a *Applier) ReconcileAll(ctx context.Context) {
	rules, err := a.repo.ListAll(ctx)
	if err != nil {
		slog.Warn("firewall reconcile: list rules", "error", err)
		return
	}
	seen := map[string]bool{}
	for _, r := range rules {
		if seen[r.AppID] {
			continue
		}
		seen[r.AppID] = true
		if err := a.Apply(ctx, r.AppID); err != nil {
			slog.Warn("firewall reconcile: apply", "app_id", r.AppID, "error", err)
		}
	}
}
