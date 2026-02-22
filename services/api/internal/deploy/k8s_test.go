package deploy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/store"
)

func TestGenerateK8sResources(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Name:      "web",
		Subdomain: "web",
		Port:      3000,
	}
	envVars := []entities.EnvVar{
		{Key: "DATABASE_URL", Value: "postgres://..."},
		{Key: "API_KEY", Value: "secret"},
	}

	resources := GenerateK8sResources(app, "registry/web:latest", "freezenith.com", envVars, nil)

	// Verify Deployment
	if resources.Deployment["kind"] != "Deployment" {
		t.Errorf("Expected kind 'Deployment', got '%v'", resources.Deployment["kind"])
	}

	// Verify Service
	if resources.Service["kind"] != "Service" {
		t.Errorf("Expected kind 'Service', got '%v'", resources.Service["kind"])
	}

	// Verify IngressRoute
	if resources.IngressRoute["kind"] != "IngressRoute" {
		t.Errorf("Expected kind 'IngressRoute', got '%v'", resources.IngressRoute["kind"])
	}
}

func TestGenerateK8sResourcesDefaultPort(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Name:      "api",
		Subdomain: "api",
		Port:      0,
	}

	resources := GenerateK8sResources(app, "registry/api:latest", "freezenith.com", nil, nil)

	// Verify port defaults to 8080
	data, _ := json.Marshal(resources.Deployment)
	if !containsStr(string(data), "8080") {
		t.Error("Expected default port 8080 in deployment")
	}
}

func TestGenerateK8sResourcesLabels(t *testing.T) {
	app := &entities.App{
		ID:        "app-456",
		Name:      "worker",
		Subdomain: "worker",
		Port:      8080,
	}

	resources := GenerateK8sResources(app, "registry/worker:v1", "example.com", nil, nil)

	metadata := resources.Deployment["metadata"].(map[string]interface{})
	labels := metadata["labels"].(map[string]string)

	if labels["zenith.dev/app-id"] != "app-456" {
		t.Errorf("Expected app-id label, got '%s'", labels["zenith.dev/app-id"])
	}
	if labels["zenith.dev/managed-by"] != "zenith" {
		t.Error("Expected managed-by label")
	}
}

func TestGenerateIngressRouteHost(t *testing.T) {
	app := &entities.App{
		ID:        "app-789",
		Name:      "frontend",
		Subdomain: "frontend",
		Port:      3000,
	}

	resources := GenerateK8sResources(app, "reg/fe:v1", "mypaas.dev", nil, nil)

	data, _ := json.Marshal(resources.IngressRoute)
	content := string(data)

	if !containsStr(content, "frontend.mypaas.dev") {
		t.Error("Expected host 'frontend.mypaas.dev' in IngressRoute")
	}
	if !containsStr(content, "letsencrypt") {
		t.Error("Expected TLS certResolver 'letsencrypt'")
	}
}

func TestResourcesSerializeToJSON(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Name:      "web",
		Subdomain: "web",
		Port:      3000,
	}

	resources := GenerateK8sResources(app, "reg/web:v1", "test.com", nil, nil)

	for _, r := range []map[string]interface{}{resources.Deployment, resources.Service, resources.IngressRoute} {
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("Failed to serialize resource: %v", err)
		}
		if len(data) < 10 {
			t.Error("Serialized resource is suspiciously small")
		}
	}
}

// --- Deployer tests ---

func TestDeployerDeployApp(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	repo := store.NewMemoryAppRepository()
	deployer := NewDeployer(k8sClient, repo, nil, "freezenith.com")

	// Create an app
	app, err := repo.CreateApp(context.Background(), &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "web",
		RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Deploy
	err = deployer.DeployApp(context.Background(), app, "registry/web:latest")
	if err != nil {
		t.Fatalf("DeployApp failed: %v", err)
	}

	// Verify app status updated to running
	updated, _ := repo.GetApp(context.Background(), app.ID)
	if updated.Status != entities.AppStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", updated.Status)
	}
}

func TestDeployerDeleteApp(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	repo := store.NewMemoryAppRepository()
	deployer := NewDeployer(k8sClient, repo, nil, "freezenith.com")

	app := &entities.App{
		ID:        "app-123",
		Name:      "web",
		Subdomain: "web",
	}

	// Should not error even if resources don't exist
	err := deployer.DeleteApp(context.Background(), app)
	if err != nil {
		t.Fatalf("DeleteApp should not error for nonexistent resources: %v", err)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
