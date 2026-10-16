package main

import (
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/config"
	"github.com/gofiber/fiber/v2"
)

// saasOnlyPrefixes are route families that must not exist on a self-hosted install.
var saasOnlyPrefixes = []string{
	"/api/v1/billing", "/api/v1/plan", "/api/v1/referral",
	"/api/v1/admin/plans", "/api/v1/admin/billing", "/api/v1/admin/exit-surveys",
	"/api/v1/admin/referrals", "/api/v1/admin/crm", "/api/v1/admin/users/:userId/plan",
}

func registeredPaths(t *testing.T, mode string) map[string]bool {
	t.Helper()
	cfg := &config.Config{Mode: mode, JWTSecret: "test-secret-test-secret-test-secret", K8sMode: "memory", BaseDomain: "example.com", CORSOrigins: "http://localhost:3000"}
	app := fiber.New()
	setupRoutes(app, cfg,
		memory.NewMemoryUserRepository(), memory.NewMemoryAdminRepository(),
		memory.NewMemoryCustomerRepository(), memory.NewMemoryMeteringRepository(),
		memory.NewMemoryAppRepository(), nil)
	paths := map[string]bool{}
	for _, r := range app.GetRoutes(true) {
		paths[r.Path] = true
	}
	return paths
}

func TestStandaloneRegistersNoSaaSRoutes(t *testing.T) {
	paths := registeredPaths(t, "standalone")
	for p := range paths {
		for _, prefix := range saasOnlyPrefixes {
			if strings.HasPrefix(p, prefix) {
				t.Errorf("standalone registered SaaS route %s", p)
			}
		}
	}
	if !paths["/api/v1/auth/register"] || !paths["/api/v1/apps/check-name"] {
		t.Fatalf("core routes missing: %v", paths)
	}
}

func TestSaaSRegistersSaaSRoutes(t *testing.T) {
	paths := registeredPaths(t, "saas")
	for _, want := range []string{"/api/v1/billing", "/api/v1/plan", "/api/v1/admin/crm/pipeline"} {
		found := false
		for p := range paths {
			if p == want || strings.HasPrefix(p, want+"/") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("saas mode did not register %s", want)
		}
	}
	if !paths["/api/v1/auth/register"] {
		t.Fatal("core routes missing in saas mode")
	}
}
