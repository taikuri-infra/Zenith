package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/gofiber/fiber/v2"
)

func setupBackstageTest() (*fiber.App, *handlers.BackstageHandler, *k8s.MemoryClient) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	client := k8s.NewMemoryClient()
	handler := handlers.NewBackstageHandler(client)
	return app, handler, client
}

func TestBackstageGetCatalogEmpty(t *testing.T) {
	app, handler, _ := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 0 {
		t.Errorf("Expected 0 entities, got %d", result.Total)
	}
	if len(result.Items) != 0 {
		t.Errorf("Expected empty items, got %d", len(result.Items))
	}
}

func TestBackstageGetCatalogWithProject(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	// Create a project CRD
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "My Project",
		"owner":       "user@test.com",
		"plan":        "pro",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name: "myproj",
			Labels: map[string]string{
				"zenith.dev/owner": "user@test.com",
			},
		},
		Spec: projSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total < 1 {
		t.Errorf("Expected at least 1 entity (System), got %d", result.Total)
	}

	// Check that the System entity is present
	foundSystem := false
	for _, e := range result.Items {
		if e.Kind == "System" && e.Metadata.Name == "myproj" {
			foundSystem = true
			if e.APIVersion != "backstage.io/v1alpha1" {
				t.Errorf("Expected backstage.io/v1alpha1, got %s", e.APIVersion)
			}
		}
	}
	if !foundSystem {
		t.Error("Expected a System entity for the project")
	}
}

func TestBackstageGetCatalogWithAppsAndDatabases(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	// Create project
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Test Project",
		"owner":       "user@test.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name: "testproj",
			Labels: map[string]string{
				"zenith.dev/owner": "user@test.com",
			},
		},
		Spec: projSpec,
	})

	// Create app in project namespace
	appSpec, _ := json.Marshal(map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
		"domain":   "app.example.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: k8s.ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-testproj",
			Labels: map[string]string{
				"zenith.dev/app-name": "web",
				"zenith.dev/project":  "testproj",
			},
		},
		Spec: appSpec,
	})

	// Create database in project namespace
	dbSpec, _ := json.Marshal(map[string]interface{}{
		"engine":  "postgresql",
		"version": "16",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata: k8s.ObjectMeta{
			Name:      "main-db",
			Namespace: "zenith-testproj",
			Labels: map[string]string{
				"zenith.dev/project": "testproj",
			},
		},
		Spec: dbSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Should have: 1 System (project) + 1 Component (app) + 1 Resource (database) = 3
	if result.Total < 3 {
		t.Errorf("Expected at least 3 entities, got %d", result.Total)
	}

	kindCounts := make(map[string]int)
	for _, e := range result.Items {
		kindCounts[e.Kind]++
	}

	if kindCounts["System"] < 1 {
		t.Error("Expected at least 1 System entity")
	}
	if kindCounts["Component"] < 1 {
		t.Error("Expected at least 1 Component entity")
	}
	if kindCounts["Resource"] < 1 {
		t.Error("Expected at least 1 Resource entity")
	}
}

func TestBackstageGetCatalogByKind(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog/:kind", handler.GetCatalogByKind)

	// Create project and app
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Test",
		"owner":       "user@test.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8s.ObjectMeta{Name: "proj1"},
		Spec:       projSpec,
	})

	appSpec, _ := json.Marshal(map[string]interface{}{"image": "nginx:latest"})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: k8s.ObjectMeta{
			Name:      "my-app",
			Namespace: "zenith-proj1",
			Labels:    map[string]string{"zenith.dev/app-name": "my-app"},
		},
		Spec: appSpec,
	})

	// Get only Component entities
	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog/Component", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	for _, e := range result.Items {
		if e.Kind != "Component" {
			t.Errorf("Expected only Component entities, got %s", e.Kind)
		}
	}
}

func TestBackstageGetCatalogByInvalidKind(t *testing.T) {
	app, handler, _ := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog/:kind", handler.GetCatalogByKind)

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog/InvalidKind", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestBackstageEntityAnnotations(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Annotated Project",
		"owner":       "owner@test.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name: "annotproj",
			Labels: map[string]string{
				"zenith.dev/owner": "owner@test.com",
			},
		},
		Spec: projSpec,
	})

	// Create a domain
	domainSpec, _ := json.Marshal(map[string]interface{}{
		"domain": "api.example.com",
		"appRef": "web-app",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Domain",
		Metadata: k8s.ObjectMeta{
			Name:      "api-domain",
			Namespace: "zenith-annotproj",
			Labels:    map[string]string{"zenith.dev/project": "annotproj"},
		},
		Spec: domainSpec,
	})

	// Ignore unused variable
	_ = bytes.NewBuffer(nil)

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Find the API entity
	for _, e := range result.Items {
		if e.Kind == "API" && e.Metadata.Name == "api-domain" {
			if e.Metadata.Annotations["zenith.dev/domain"] != "api.example.com" {
				t.Errorf("Expected domain annotation, got '%s'", e.Metadata.Annotations["zenith.dev/domain"])
			}
			if e.Metadata.Annotations["zenith.dev/app-ref"] != "web-app" {
				t.Errorf("Expected app-ref annotation, got '%s'", e.Metadata.Annotations["zenith.dev/app-ref"])
			}
			if e.Metadata.Annotations["zenith.dev/project"] != "annotproj" {
				t.Errorf("Expected project annotation, got '%s'", e.Metadata.Annotations["zenith.dev/project"])
			}
		}
	}
}
