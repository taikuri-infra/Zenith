package deploy

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestShouldScaleToZero(t *testing.T) {
	tests := []struct {
		name   string
		tier   entities.PlanTier
		expect bool
	}{
		{"free tier scales to zero", entities.PlanFree, true},
		{"pro tier always on", entities.PlanPro, false},
		{"team tier always on", entities.PlanTeam, false},
		{"enterprise tier always on", entities.PlanEnterprise, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limits := entities.DefaultPlanLimits(tc.tier)
			got := ShouldScaleToZero(&limits)
			if got != tc.expect {
				t.Errorf("ShouldScaleToZero(%s) = %v, want %v", tc.tier, got, tc.expect)
			}
		})
	}
}

func TestGenerateHTTPScaledObject(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Subdomain: "myapp",
		Port:      8080,
	}

	obj := GenerateHTTPScaledObject(app, "freezenith.com", 15)

	// Verify top-level fields
	if obj["apiVersion"] != "http.keda.sh/v1alpha1" {
		t.Errorf("apiVersion = %v, want http.keda.sh/v1alpha1", obj["apiVersion"])
	}
	if obj["kind"] != "HTTPScaledObject" {
		t.Errorf("kind = %v, want HTTPScaledObject", obj["kind"])
	}

	// Verify metadata
	meta := obj["metadata"].(map[string]interface{})
	if meta["name"] != "myapp" {
		t.Errorf("metadata.name = %v, want myapp", meta["name"])
	}
	if meta["namespace"] != "zenith-apps" {
		t.Errorf("metadata.namespace = %v, want zenith-apps", meta["namespace"])
	}

	// Verify spec
	spec := obj["spec"].(map[string]interface{})

	hosts := spec["hosts"].([]string)
	if len(hosts) != 1 || hosts[0] != "myapp.freezenith.com" {
		t.Errorf("spec.hosts = %v, want [myapp.freezenith.com]", hosts)
	}

	targetRef := spec["scaleTargetRef"].(map[string]interface{})
	if targetRef["name"] != "myapp" {
		t.Errorf("scaleTargetRef.name = %v, want myapp", targetRef["name"])
	}
	if targetRef["kind"] != "Deployment" {
		t.Errorf("scaleTargetRef.kind = %v, want Deployment", targetRef["kind"])
	}

	replicas := spec["replicas"].(map[string]interface{})
	if replicas["min"] != 0 {
		t.Errorf("replicas.min = %v, want 0", replicas["min"])
	}
	if replicas["max"] != 1 {
		t.Errorf("replicas.max = %v, want 1", replicas["max"])
	}

	if spec["scaledownPeriod"] != 900 {
		t.Errorf("scaledownPeriod = %v, want 900", spec["scaledownPeriod"])
	}
	if spec["targetPendingRequests"] != 1 {
		t.Errorf("targetPendingRequests = %v, want 1", spec["targetPendingRequests"])
	}
}

func TestGenerateHTTPScaledObjectDefaultScaledown(t *testing.T) {
	app := &entities.App{ID: "a", Subdomain: "test"}
	obj := GenerateHTTPScaledObject(app, "example.com", 0)
	spec := obj["spec"].(map[string]interface{})
	if spec["scaledownPeriod"] != 900 {
		t.Errorf("expected default scaledownPeriod 900, got %v", spec["scaledownPeriod"])
	}
}

func TestK8sResourcesWithPlan_FreeTier(t *testing.T) {
	app := &entities.App{
		ID:        "app-1",
		Subdomain: "freeapp",
		Port:      3000,
	}
	limits := entities.DefaultPlanLimits(entities.PlanFree)

	resources := GenerateK8sResources(app, "freeapp:v1", "freezenith.com", nil, &limits, entities.PlanFree, nil)

	// Free tier: replicas should be 0 (KEDA manages scaling)
	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(0) {
		t.Errorf("free tier replicas = %v, want 0", spec["replicas"])
	}

	// HTTPScaledObject should be set
	if resources.HTTPScaledObject == nil {
		t.Error("free tier HTTPScaledObject should not be nil")
	}

	hso := resources.HTTPScaledObject
	if hso["kind"] != "HTTPScaledObject" {
		t.Errorf("HTTPScaledObject kind = %v, want HTTPScaledObject", hso["kind"])
	}
}

func TestK8sResourcesAlwaysOn(t *testing.T) {
	app := &entities.App{
		ID:        "app-2",
		Subdomain: "proapp",
		Port:      8080,
	}
	limits := entities.DefaultPlanLimits(entities.PlanPro)

	resources := GenerateK8sResources(app, "proapp:v1", "freezenith.com", nil, &limits, entities.PlanPro, nil)

	// Paid tier: replicas should be 1
	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(1) {
		t.Errorf("paid tier replicas = %v, want 1", spec["replicas"])
	}

	// No HTTPScaledObject for paid tier
	if resources.HTTPScaledObject != nil {
		t.Error("paid tier HTTPScaledObject should be nil")
	}
}

func TestK8sResourcesNilPlanLimits(t *testing.T) {
	app := &entities.App{
		ID:        "app-3",
		Subdomain: "noplan",
		Port:      8080,
	}

	resources := GenerateK8sResources(app, "noplan:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	// nil limits = always-on (backwards compatible)
	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(1) {
		t.Errorf("nil plan replicas = %v, want 1", spec["replicas"])
	}

	if resources.HTTPScaledObject != nil {
		t.Error("nil plan HTTPScaledObject should be nil")
	}
}
