package handlers_test

import (
	"io"
	"net/http"
	"strings"

	"bytes"
	"encoding/json"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
)

func TestDomainAddRejectsInvalidAndReserved(t *testing.T) {
	for _, domain := range []string{"x`) || PathPrefix(`/", "zenith.example.com", "app.zenith.example.com", "no-dot", "a b.com"} {
		fiberApp, handler, _, appID := setupDomainTest()
		handler.SetReservedDomain("zenith.example.com")
		fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

		body, _ := json.Marshal(map[string]string{"domain": domain})
		req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := fiberApp.Test(req, -1)
		if resp.StatusCode != 400 {
			t.Errorf("domain %q: status %d, want 400", domain, resp.StatusCode)
		}
	}
}

func TestDomainAddNormalizes(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	handler.SetReservedDomain("zenith.example.com")
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(`{"domain":" UPPER.Example.com "}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("status %d, want 201", resp.StatusCode)
	}
	var result dto.DomainInfo
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Domain != "upper.example.com" {
		t.Errorf("stored domain %q, want upper.example.com", result.Domain)
	}
}

func postDomain(t *testing.T, appID, domain string, fiberApp interface {
	Test(*http.Request, ...int) (*http.Response, error)
}) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"domain": domain})
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req, -1)
	return resp
}

func TestDomainAddSubdomainsAllowed(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	handler.SetReservedDomain("zenith.example.com")
	handler.SetSubdomainsAllowed(true)
	handler.SetChecker(services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com"))
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	resp := postDomain(t, appID, "myapp.zenith.example.com", fiberApp)
	if resp.StatusCode != 201 {
		t.Fatalf("subdomain: status %d, want 201", resp.StatusCode)
	}
	var out dto.DomainInfo
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusPending {
		t.Errorf("status %q, want pending", out.Status)
	}
	if resp := postDomain(t, appID, "zenith.example.com", fiberApp); resp.StatusCode != 400 {
		t.Errorf("base domain: status %d, want 400", resp.StatusCode)
	}
}

func TestDomainAddRefusesAppGeneratedHost(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest() // app subdomain: test-app
	handler.SetReservedDomain("zenith.example.com")
	handler.SetSubdomainsAllowed(true)
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	resp := postDomain(t, appID, "Test-App.zenith.example.com", fiberApp)
	if resp.StatusCode != 409 {
		t.Fatalf("status %d, want 409", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "that host belongs to an app") {
		t.Errorf("body %s", b)
	}
	// a deeper name under an app host is not that app's host
	if resp := postDomain(t, appID, "x.test-app.zenith.example.com", fiberApp); resp.StatusCode != 201 {
		t.Errorf("nested: status %d, want 201", resp.StatusCode)
	}
}

func TestDomainAddSubdomainRefusedInSaaS(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	handler.SetReservedDomain("zenith.example.com")
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)
	if resp := postDomain(t, appID, "myapp.zenith.example.com", fiberApp); resp.StatusCode != 400 {
		t.Errorf("status %d, want 400", resp.StatusCode)
	}
}
