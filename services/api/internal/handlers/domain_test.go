package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

func setupDomainTest() (*fiber.App, *handlers.DomainHandler, *memory.MemoryDomainRepository, string) {
	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	domainRepo := memory.NewMemoryDomainRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Pro plan for custom domains
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	handler := handlers.NewDomainHandler(domainRepo, appRepo, planRepo)

	app, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})

	return fiberApp, handler, domainRepo, app.ID
}

func TestDomainAdd(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", result.Domain)
	}
	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestDomainAddNoDomain(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDomainAddAppNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainAddForbidden(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-2"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestDomainAddDuplicate(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	fiberApp.Test(req, -1)

	// Duplicate
	req2 := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req2, -1)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409, got %d", resp.StatusCode)
	}
}

func TestDomainList(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domainRepo.AddDomain(nil, appID, "user-1", "a.example.com")
	domainRepo.AddDomain(nil, appID, "user-1", "b.example.com")

	fiberApp.Get("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/domains", nil)
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(result))
	}
}

func TestDomainListEmpty(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Get("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/domains", nil)
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestDomainDelete(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domain, _ := domainRepo.AddDomain(nil, appID, "user-1", "example.com")

	fiberApp.Delete("/api/v1/apps/:appId/domains/:domainId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/domains/"+domain.ID, nil)
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestDomainDeleteNotFound(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Delete("/api/v1/apps/:appId/domains/:domainId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/domains/nonexistent", nil)
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainDeleteForbidden(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domain, _ := domainRepo.AddDomain(nil, appID, "user-1", "example.com")

	fiberApp.Delete("/api/v1/apps/:appId/domains/:domainId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/domains/"+domain.ID, nil)
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestDomainListByUser(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domainRepo.AddDomain(nil, appID, "user-1", "a.example.com")
	domainRepo.AddDomain(nil, appID, "user-1", "b.example.com")

	fiberApp.Get("/api/v1/domains", injectUserID("user-1"), handler.ListByUser)

	req := httptest.NewRequest("GET", "/api/v1/domains", nil)
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(result))
	}
}

type recordingRedeployer struct{ image string }

func (r *recordingRedeployer) DeployApp(_ context.Context, _ *entities.App, image string) error {
	r.image = image
	return nil
}

func TestDomainAdd_RedeploysWithCurrentImage(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	domainRepo := memory.NewMemoryDomainRepository()
	planRepo := memory.NewUnlimitedPlanRepository(memory.NewMemoryUserPlanRepository())
	a, _ := appRepo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	d, _ := appRepo.CreateDeployment(context.Background(), a.ID, "nginx:1")
	_ = appRepo.UpdateDeploymentStatus(context.Background(), d.ID, entities.DeployStatusActive, "", "")

	h := handlers.NewDomainHandler(domainRepo, appRepo, planRepo)
	rec := &recordingRedeployer{}
	h.SetDeployer(rec)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), h.Add)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+a.ID+"/domains", bytes.NewBufferString(`{"domain":"myapp.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if rec.image != "nginx:1" {
		t.Errorf("redeployed image = %q, want nginx:1", rec.image)
	}
}

type stubResolver struct{ ips map[string][]string }

func (s stubResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := s.ips[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

type stubProber struct{ valid bool }

func (s stubProber) Probe(context.Context, string, string) (bool, error) { return s.valid, nil }

func TestDomainAddIsPendingWithChecker(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	checker := services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com")
	handler.SetChecker(checker)
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(`{"domain":"app.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusPending || out.StatusMessage != "waiting for DNS check" || out.LastCheckedAt != nil {
		t.Fatalf("got %+v", out)
	}
}

func TestDomainAddStaysActiveWithoutChecker(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(`{"domain":"app.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req, -1)
	var out dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusActive {
		t.Fatalf("saas behaviour changed: %+v", out)
	}
}

func TestDomainVerifyRunsOneCheck(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	checker := services.NewDomainChecker(repo, stubResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, stubProber{valid: true}, "zenith.example.com")
	handler.SetChecker(checker)
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	_ = checker.MarkPending(context.Background(), d.ID)

	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusActive || !out.TLSReady || out.LastCheckedAt == nil {
		t.Fatalf("got %+v", out)
	}
}

func TestDomainVerifyForbiddenForOtherUser(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	handler.SetChecker(services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com"))
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-2"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 403 {
		t.Fatalf("status %d, want 403", resp.StatusCode)
	}
}

func TestDomainVerifyWithoutCheckerIs404(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

func TestDomainVerifyUnknownDomainIs404(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	handler.SetChecker(services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com"))
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/nope/verify", nil), -1)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

func TestDomainVerifyCheckerErrorIs502(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	handler.SetChecker(services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com")) // dashboard does not resolve
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 502 {
		t.Fatalf("status %d, want 502", resp.StatusCode)
	}
}

type chanRefresher chan string

func (r chanRefresher) RefreshRoutes(_ context.Context, appID string) error {
	r <- appID
	return nil
}

func TestDomainVerifyResetsRefreshSchedule(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	checker := services.NewDomainChecker(repo, stubResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, stubProber{valid: false}, "zenith.example.com")
	refreshes := make(chanRefresher, 4)
	checker.SetRefresher(refreshes)
	handler.SetChecker(checker)
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	_ = repo.UpdateDomainCheck(context.Background(), d.ID, entities.DomainStatusVerified, false, "DNS ok; waiting for the certificate", time.Now())
	d, _ = repo.GetDomain(context.Background(), d.ID)
	if _, err := checker.Check(context.Background(), d); err != nil { // background round: seeded, no refresh
		t.Fatal(err)
	}
	select {
	case <-refreshes:
		t.Fatal("first sight of a verified domain must not refresh")
	case <-time.After(50 * time.Millisecond):
	}

	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	select {
	case got := <-refreshes:
		if got != appID {
			t.Fatalf("refreshed %s, want %s", got, appID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Check now did not refresh routes")
	}
}
