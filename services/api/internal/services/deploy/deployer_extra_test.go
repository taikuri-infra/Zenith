package deploy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDeployer_SetDomainRepo(t *testing.T) {
	deployer := NewDeployer(nil, nil, nil, "freezenith.com")
	if deployer.domainRepo != nil {
		t.Error("Expected nil domainRepo initially")
	}
	deployer.SetDomainRepo(nil)
}

func TestDeployer_SetEnvVarRepo(t *testing.T) {
	deployer := NewDeployer(nil, nil, nil, "freezenith.com")
	if deployer.envVarRepo != nil {
		t.Error("Expected nil envVarRepo initially")
	}
	deployer.SetEnvVarRepo(nil)
}

func TestDeployer_SetEnvCrypto(t *testing.T) {
	deployer := NewDeployer(nil, nil, nil, "freezenith.com")
	if deployer.envCrypto != nil {
		t.Error("Expected nil envCrypto initially")
	}
	deployer.SetEnvCrypto(nil)
}

func TestDeployer_SetEnvRepo(t *testing.T) {
	deployer := NewDeployer(nil, nil, nil, "freezenith.com")
	if deployer.envRepo != nil {
		t.Error("Expected nil envRepo initially")
	}
	deployer.SetEnvRepo(nil)
}

func TestDeployer_DeployApp_WithDomains(t *testing.T) {
	k8sClient := k8sclient.NewMemoryClient()
	appRepo := memory.NewMemoryAppRepository()
	domainRepo := memory.NewMemoryDomainRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	ctx := context.Background()
	planRepo.SetUserPlan(ctx, "user-1", entities.PlanPro)

	app := createTestApp(t, ctx, appRepo, "user-1", "domain-app")

	// Add an active custom domain via the domain repo
	dom, err := domainRepo.AddDomain(ctx, app.ID, "user-1", "example.com")
	if err != nil {
		t.Fatalf("AddDomain failed: %v", err)
	}
	// Mark it as active
	domainRepo.UpdateDomainStatus(ctx, dom.ID, entities.DomainStatusActive, true)

	deployer := NewDeployer(k8sClient, appRepo, planRepo, "freezenith.com")
	deployer.SetDomainRepo(domainRepo)

	err = deployer.DeployApp(ctx, app, "domain-app:v1")
	if err != nil {
		t.Fatalf("DeployApp with domains failed: %v", err)
	}

	// Verify app is running
	updated, _ := appRepo.GetApp(ctx, app.ID)
	if updated.Status != entities.AppStatusRunning {
		t.Errorf("app status = %v, want running", updated.Status)
	}

	// Verify the IngressRoute was created with the custom domain
	ir, err := k8sClient.GetCRD(ctx, "IngressRoute", "zenith-apps", app.Subdomain)
	if err != nil {
		t.Fatalf("GetCRD IngressRoute failed: %v", err)
	}
	data, _ := json.Marshal(ir)
	if !containsStr(string(data), "example.com") {
		t.Error("Expected custom domain in IngressRoute")
	}
}

func TestDeployer_DeployApp_WithEnvVars(t *testing.T) {
	k8sClient := k8sclient.NewMemoryClient()
	appRepo := memory.NewMemoryAppRepository()

	ctx := context.Background()
	app := createTestApp(t, ctx, appRepo, "user-1", "env-deploy-app")

	// Add env vars to the app via legacy method
	appRepo.SetEnvVars(ctx, app.ID, map[string]string{
		"NODE_ENV": "production",
	})

	deployer := NewDeployer(k8sClient, appRepo, nil, "freezenith.com")
	err := deployer.DeployApp(ctx, app, "env-deploy-app:v1")
	if err != nil {
		t.Fatalf("DeployApp with env vars failed: %v", err)
	}

	// Verify deployment was created with env vars
	dep, err := k8sClient.GetCRD(ctx, "Deployment", "zenith-apps", app.Subdomain)
	if err != nil {
		t.Fatalf("GetCRD Deployment failed: %v", err)
	}
	data, _ := json.Marshal(dep)
	if !containsStr(string(data), "NODE_ENV") {
		t.Error("Expected NODE_ENV in deployment env vars")
	}
}
