package deploy

import (
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestStagingPerAppResources(t *testing.T) {
	cpuL, memL, cpuR, memR := StagingPerAppResources()
	if cpuL != "250m" {
		t.Errorf("Expected cpuLimit '250m', got '%s'", cpuL)
	}
	if memL != "256Mi" {
		t.Errorf("Expected memLimit '256Mi', got '%s'", memL)
	}
	if cpuR != "50m" {
		t.Errorf("Expected cpuReq '50m', got '%s'", cpuR)
	}
	if memR != "64Mi" {
		t.Errorf("Expected memReq '64Mi', got '%s'", memR)
	}
}

func TestAppHostname_Production(t *testing.T) {
	hostname := AppHostname("myapp", "apps.freezenith.com", false)
	if hostname != "myapp.apps.freezenith.com" {
		t.Errorf("Expected 'myapp.apps.freezenith.com', got '%s'", hostname)
	}
}

func TestAppHostname_Staging(t *testing.T) {
	hostname := AppHostname("myapp", "apps.freezenith.com", true)
	if hostname != "myapp.dev.apps.freezenith.com" {
		t.Errorf("Expected 'myapp.dev.apps.freezenith.com', got '%s'", hostname)
	}
}

func TestPerAppResources_AllTiers(t *testing.T) {
	tests := []struct {
		tier                             entities.PlanTier
		expectedCPULimit, expectedMemLim string
	}{
		{entities.PlanFree, "100m", "128Mi"},
		{entities.PlanPro, "500m", "512Mi"},
		{entities.PlanTeam, "1000m", "1Gi"},
		{entities.PlanBusiness, "2000m", "2Gi"},
		{entities.PlanEnterprise, "4000m", "4Gi"},
	}

	for _, tc := range tests {
		t.Run(string(tc.tier), func(t *testing.T) {
			cpuL, memL, _, _ := PerAppResources(tc.tier)
			if cpuL != tc.expectedCPULimit {
				t.Errorf("CPU limit for %s: expected '%s', got '%s'", tc.tier, tc.expectedCPULimit, cpuL)
			}
			if memL != tc.expectedMemLim {
				t.Errorf("Mem limit for %s: expected '%s', got '%s'", tc.tier, tc.expectedMemLim, memL)
			}
		})
	}
}

func TestGenerateK8sResources_WithCustomDomains(t *testing.T) {
	app := &entities.App{
		ID:        "app-custom",
		Name:      "custom-domain-app",
		Subdomain: "myapp",
		Port:      8080,
	}
	customDomains := []string{"example.com", "www.example.com"}

	resources := GenerateK8sResources(app, "myapp:v1", "freezenith.com", nil, nil, entities.PlanFree, customDomains)

	// IngressRoute should contain the custom domains
	data, _ := json.Marshal(resources.IngressRoute)
	content := string(data)

	if !containsStr(content, "example.com") {
		t.Error("Expected 'example.com' in IngressRoute")
	}
	if !containsStr(content, "www.example.com") {
		t.Error("Expected 'www.example.com' in IngressRoute")
	}

	// Certificate should be generated for custom domains
	if resources.Certificate == nil {
		t.Error("Expected Certificate resource for custom domains")
	}
}

func TestGenerateK8sResources_WithEnvVars(t *testing.T) {
	app := &entities.App{
		ID:        "app-env",
		Name:      "env-app",
		Subdomain: "envapp",
		Port:      3000,
	}
	envVars := []entities.EnvVar{
		{Key: "DATABASE_URL", Value: "postgres://..."},
		{Key: "SECRET_KEY", Value: "mysecret"},
	}

	resources := GenerateK8sResources(app, "envapp:v1", "freezenith.com", envVars, nil, entities.PlanFree, nil)

	data, _ := json.Marshal(resources.Deployment)
	content := string(data)

	if !containsStr(content, "DATABASE_URL") {
		t.Error("Expected DATABASE_URL in deployment env vars")
	}
	if !containsStr(content, "SECRET_KEY") {
		t.Error("Expected SECRET_KEY in deployment env vars")
	}
}

func TestGenerateK8sResources_StagingFlag(t *testing.T) {
	app := &entities.App{
		ID:        "app-staging",
		Name:      "staging-app",
		Subdomain: "staging",
		Port:      8080,
	}

	resources := GenerateK8sResources(app, "staging:v1", "apps.freezenith.com", nil, nil, entities.PlanFree, nil, true)

	// Staging apps should use dev subdomain
	data, _ := json.Marshal(resources.IngressRoute)
	content := string(data)

	if !containsStr(content, "staging.dev.apps.freezenith.com") {
		t.Error("Expected staging hostname 'staging.dev.apps.freezenith.com' in IngressRoute")
	}
}

func TestGenerateK8sResources_NetworkPolicy(t *testing.T) {
	app := &entities.App{
		ID:        "app-np",
		Name:      "netpol-app",
		Subdomain: "netpol",
		Port:      8080,
	}

	resources := GenerateK8sResources(app, "netpol:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	if resources.NetworkPolicy == nil {
		t.Error("Expected NetworkPolicy resource")
	}
	if resources.NetworkPolicy["kind"] != "NetworkPolicy" {
		t.Errorf("Expected kind 'NetworkPolicy', got '%v'", resources.NetworkPolicy["kind"])
	}
}

func TestGenerateK8sResources_ScaleToZero(t *testing.T) {
	app := &entities.App{
		ID:        "app-s2z",
		Name:      "sleep-app",
		Subdomain: "sleepapp",
		Port:      8080,
	}
	limits := entities.PlanLimits{
		AlwaysOn:       false, // scale-to-zero enabled
		SleepAfterMins: 15,
	}

	resources := GenerateK8sResources(app, "sleepapp:v1", "freezenith.com", nil, &limits, entities.PlanFree, nil)

	// Scale-to-zero: replicas should be 0
	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(0) {
		t.Errorf("scale-to-zero replicas = %v, want 0", spec["replicas"])
	}

	// Should have HTTPScaledObject
	if resources.HTTPScaledObject == nil {
		t.Error("Expected HTTPScaledObject for scale-to-zero")
	}

	// IngressRoute should have cold-start middleware
	data, _ := json.Marshal(resources.IngressRoute)
	if !containsStr(string(data), "cold-start-errors") {
		t.Error("Expected cold-start-errors middleware in IngressRoute")
	}
}

func TestGenerateK8sResources_ScaleToZero_Staging_NoHTTPScaledObject(t *testing.T) {
	app := &entities.App{
		ID:        "app-s2z-staging",
		Name:      "sleep-staging-app",
		Subdomain: "sleepstg",
		Port:      8080,
	}
	limits := entities.PlanLimits{
		AlwaysOn:       false,
		SleepAfterMins: 15,
	}

	// Staging should NOT scale-to-zero even with scale-to-zero limits
	resources := GenerateK8sResources(app, "sleepstg:v1", "apps.freezenith.com", nil, &limits, entities.PlanFree, nil, true)

	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(1) {
		t.Errorf("staging replicas = %v, want 1 (no scale-to-zero)", spec["replicas"])
	}

	if resources.HTTPScaledObject != nil {
		t.Error("Staging should not have HTTPScaledObject")
	}
}

func TestGenerateK8sResources_WorkerApp(t *testing.T) {
	app := &entities.App{
		ID:        "app-worker",
		Name:      "worker-app",
		Subdomain: "worker",
		Port:      8080,
		AppType:   entities.AppTypeWorker,
	}

	resources := GenerateK8sResources(app, "worker:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	data, _ := json.Marshal(resources.Deployment)
	content := string(data)

	// Worker should have TCP probes, not HTTP
	if !containsStr(content, "tcpSocket") {
		t.Error("Expected TCP socket probe for worker app")
	}
	if containsStr(content, "httpGet") {
		t.Error("Worker app should not have HTTP probes")
	}
}

func TestGenerateK8sResources_CronApp(t *testing.T) {
	app := &entities.App{
		ID:        "app-cron",
		Name:      "cron-app",
		Subdomain: "cron",
		Port:      0,
		AppType:   entities.AppTypeCron,
	}

	resources := GenerateK8sResources(app, "cron:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	data, _ := json.Marshal(resources.Deployment)
	content := string(data)

	// Cron jobs should have no probes
	if containsStr(content, "readinessProbe") {
		t.Error("Cron app should not have readiness probe")
	}
}

func TestGenerateK8sResources_CustomHealthCheck(t *testing.T) {
	app := &entities.App{
		ID:              "app-health",
		Name:            "health-app",
		Subdomain:       "healthapp",
		Port:            3000,
		HealthCheckPath: "/healthz",
	}

	resources := GenerateK8sResources(app, "healthapp:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	data, _ := json.Marshal(resources.Deployment)
	content := string(data)

	if !containsStr(content, "/healthz") {
		t.Error("Expected custom health check path '/healthz' in deployment")
	}
}

func TestGenerateK8sResources_MultipleReplicas(t *testing.T) {
	app := &entities.App{
		ID:        "app-multi",
		Name:      "multi-app",
		Subdomain: "multi",
		Port:      8080,
		Replicas:  3,
	}

	resources := GenerateK8sResources(app, "multi:v1", "freezenith.com", nil, nil, entities.PlanPro, nil)

	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(3) {
		t.Errorf("Expected 3 replicas, got %v", spec["replicas"])
	}
}

func TestGenerateK8sResources_WithCommand(t *testing.T) {
	app := &entities.App{
		ID:        "app-cmd",
		Name:      "cmd-app",
		Subdomain: "cmdapp",
		Port:      8080,
		Command:   "node server.js --port 8080",
	}

	resources := GenerateK8sResources(app, "cmdapp:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	data, _ := json.Marshal(resources.Deployment)
	content := string(data)

	if !containsStr(content, "node") {
		t.Error("Expected command args in deployment")
	}
}

func TestGenerateK8sResources_WithDependsOn(t *testing.T) {
	app := &entities.App{
		ID:        "app-deps",
		Name:      "deps-app",
		Subdomain: "depsapp",
		Port:      8080,
		DependsOn: []string{"postgres", "redis"},
	}

	resources := GenerateK8sResources(app, "depsapp:v1", "freezenith.com", nil, nil, entities.PlanFree, nil)

	data, _ := json.Marshal(resources.Deployment)
	content := string(data)

	if !containsStr(content, "wait-for-postgres") {
		t.Error("Expected init container for postgres dependency")
	}
	if !containsStr(content, "wait-for-redis") {
		t.Error("Expected init container for redis dependency")
	}
}

func TestGenerateNetworkPolicy_WithApisix(t *testing.T) {
	app := &entities.App{
		ID:        "app-apisix",
		Subdomain: "apisixapp",
	}
	labels := map[string]string{"app": "apisixapp"}

	np := generateNetworkPolicy(app, "zenith-apps", labels, true)

	data, _ := json.Marshal(np)
	content := string(data)

	if !containsStr(content, "apisix") {
		t.Error("Expected APISIX namespace selector in network policy with allowApisix=true")
	}
}

func TestGenerateK8sResources_WithPlanLimits_ProTier(t *testing.T) {
	app := &entities.App{
		ID:        "app-pro",
		Name:      "pro-app",
		Subdomain: "proapp",
		Port:      8080,
	}
	limits := entities.DefaultPlanLimits(entities.PlanPro)

	resources := GenerateK8sResources(app, "proapp:v1", "freezenith.com", nil, &limits, entities.PlanPro, nil)

	// Pro tier should have 1 replica (always-on)
	deploy := resources.Deployment
	spec := deploy["spec"].(map[string]interface{})
	if spec["replicas"] != int32(1) {
		t.Errorf("pro tier replicas = %v, want 1", spec["replicas"])
	}

	// Verify resources are set according to tier
	data, _ := json.Marshal(resources.Deployment)
	content := string(data)
	if !containsStr(content, "500m") {
		t.Error("Expected Pro tier CPU limit '500m' in deployment")
	}
}
