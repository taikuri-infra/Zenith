package firewall

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type stubApps map[string]*entities.App

func (s stubApps) GetApp(_ context.Context, id string) (*entities.App, error) {
	if a, ok := s[id]; ok {
		return a, nil
	}
	return nil, errors.New("app not found")
}

const testNS = "zp-ab12cd34-site"

func setup(t *testing.T) (*Applier, *memory.FirewallRuleRepository, *k8sclient.MemoryClient) {
	t.Helper()
	apps := stubApps{testApp.ID: {ID: testApp.ID, Name: "api", ProjectID: "p1", UserID: "u1"}}
	repo := memory.NewFirewallRuleRepository()
	k8s := k8sclient.NewMemoryClient()
	a := NewApplier(repo, apps, k8s, func(context.Context, string) string { return testNS })
	return a, repo, k8s
}

func addRule(t *testing.T, repo *memory.FirewallRuleRepository, id string, port int) {
	t.Helper()
	_ = repo.Create(context.Background(), &entities.NetworkPolicyRule{
		ID: id, AppID: testApp.ID, Name: id, Direction: entities.NetworkPolicyIngress, Action: entities.NetworkPolicyAllow,
		Enabled: true, CreatedAt: time.Now(),
		Config: entities.NetworkPolicyConfig{Namespaces: []string{"zp-11112222-other"}, Ports: []entities.NetworkPolicyPort{{Port: port}}},
	})
}

func TestApplyCreatesPolicyInProjectNamespace(t *testing.T) {
	a, repo, k8s := setup(t)
	addRule(t, repo, "r1", 80)
	if err := a.Apply(context.Background(), testApp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := k8s.GetCRD(context.Background(), "NetworkPolicy", testNS, "zenith-fw-ab12cd34"); err != nil {
		t.Fatalf("policy not created: %v", err)
	}
}

func TestApplyDeletesWhenAllDisabled(t *testing.T) {
	a, repo, k8s := setup(t)
	ctx := context.Background()
	addRule(t, repo, "r1", 80)
	_ = a.Apply(ctx, testApp.ID)
	r, _ := repo.Get(ctx, "r1")
	r.Enabled = false
	_ = repo.Update(ctx, r)
	if err := a.Apply(ctx, testApp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := k8s.GetCRD(ctx, "NetworkPolicy", testNS, "zenith-fw-ab12cd34"); err == nil {
		t.Fatal("policy should be deleted")
	}
	// Applying again with nothing to delete is not an error.
	if err := a.Apply(ctx, testApp.ID); err != nil {
		t.Fatal(err)
	}
}

func TestApplyUpdatesExisting(t *testing.T) {
	a, repo, k8s := setup(t)
	ctx := context.Background()
	addRule(t, repo, "r1", 80)
	_ = a.Apply(ctx, testApp.ID)
	r, _ := repo.Get(ctx, "r1")
	r.Config.Ports[0].Port = 8080
	_ = repo.Update(ctx, r)
	if err := a.Apply(ctx, testApp.ID); err != nil {
		t.Fatal(err)
	}
	obj, _ := k8s.GetCRD(ctx, "NetworkPolicy", testNS, "zenith-fw-ab12cd34")
	var m map[string]any
	_ = json.Unmarshal(obj.Spec, &m)
	if dig(t, m, "ingress", 0, "ports", 0, "port") != float64(8080) {
		t.Fatalf("port not updated: %s", obj.Spec)
	}
	list, _ := k8s.ListCRDs(ctx, "NetworkPolicy", testNS)
	if len(list) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(list))
	}
}

// flakyProbe is a real-looking client whose Cilium probe fails transiently.
type flakyProbe struct{ *k8sclient.MemoryClient }

func (flakyProbe) ListCRDsWithVersion(context.Context, string, string, string) ([]*k8sclient.CRDObject, error) {
	return nil, errors.New("dial tcp: connection refused")
}

func TestApplyChangesNothingWhenCiliumProbeFails(t *testing.T) {
	_, repo, mem := setup(t)
	a := NewApplier(repo, stubApps{testApp.ID: {ID: testApp.ID, ProjectID: "p1"}}, flakyProbe{mem}, func(context.Context, string) string { return testNS })
	addRule(t, repo, "r1", 80)
	if err := a.Apply(context.Background(), testApp.ID); err == nil {
		t.Fatal("expected an error when cilium detection is inconclusive")
	}
	if list, _ := mem.ListCRDs(context.Background(), "NetworkPolicy", testNS); len(list) != 0 {
		t.Fatalf("no policy should be created, got %d", len(list))
	}
}
