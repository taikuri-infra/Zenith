package handlers_test

import (
	"bytes"
	"encoding/json"
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
