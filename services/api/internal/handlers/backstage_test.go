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

func TestBackstageGetCatalogWithStorageBuckets(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	// Create project
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Storage Project",
		"owner":       "user@test.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name: "storageproj",
			Labels: map[string]string{
				"zenith.dev/owner": "user@test.com",
			},
		},
		Spec: projSpec,
	})

	// Create storage bucket in project namespace
	bucketSpec, _ := json.Marshal(map[string]interface{}{
		"access":     "public-read",
		"versioning": true,
		"region":     "fsn1",
		"name":       "cdn-assets",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "StorageBucket",
		Metadata: k8s.ObjectMeta{
			Name:      "sb-12345678",
			Namespace: "zenith-storageproj",
			Labels: map[string]string{
				"zenith.dev/project":     "storageproj",
				"zenith.dev/bucket-name": "cdn-assets",
			},
		},
		Spec: bucketSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Should have: 1 System (project) + 1 Resource (storage) = 2
	if result.Total < 2 {
		t.Errorf("Expected at least 2 entities, got %d", result.Total)
	}

	// Find the storage Resource entity
	foundStorage := false
	for _, e := range result.Items {
		if e.Kind == "Resource" && e.Metadata.Name == "cdn-assets" {
			foundStorage = true
			if e.Metadata.Description != "S3-compatible object storage bucket" {
				t.Errorf("Expected storage description, got '%s'", e.Metadata.Description)
			}
			if e.Metadata.Annotations["zenith.dev/access"] != "public-read" {
				t.Errorf("Expected access annotation 'public-read', got '%s'", e.Metadata.Annotations["zenith.dev/access"])
			}
			if e.Metadata.Annotations["zenith.dev/region"] != "fsn1" {
				t.Errorf("Expected region annotation 'fsn1', got '%s'", e.Metadata.Annotations["zenith.dev/region"])
			}
			// Check tags
			foundStorageTag := false
			for _, tag := range e.Metadata.Tags {
				if tag == "storage" {
					foundStorageTag = true
				}
			}
			if !foundStorageTag {
				t.Error("Expected 'storage' tag on storage entity")
			}
		}
	}
	if !foundStorage {
		t.Error("Expected a Resource entity for the storage bucket")
	}
}

func TestBackstageGetCatalogWithDomains(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	// Create project
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Domain Project",
		"owner":       "user@test.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name: "domainproj",
			Labels: map[string]string{
				"zenith.dev/owner": "user@test.com",
			},
		},
		Spec: projSpec,
	})

	// Create domain in project namespace
	domainSpec, _ := json.Marshal(map[string]interface{}{
		"domain": "api.myapp.com",
		"appRef": "web-service",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Domain",
		Metadata: k8s.ObjectMeta{
			Name:      "api-domain",
			Namespace: "zenith-domainproj",
			Labels:    map[string]string{"zenith.dev/project": "domainproj"},
		},
		Spec: domainSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Find the API entity
	foundAPI := false
	for _, e := range result.Items {
		if e.Kind == "API" && e.Metadata.Name == "api-domain" {
			foundAPI = true
			if e.Spec["type"] != "openapi" {
				t.Errorf("Expected API spec type 'openapi', got '%v'", e.Spec["type"])
			}
			if e.Spec["owner"] != "user@test.com" {
				t.Errorf("Expected API spec owner 'user@test.com', got '%v'", e.Spec["owner"])
			}
			if e.Spec["system"] != "domainproj" {
				t.Errorf("Expected API spec system 'domainproj', got '%v'", e.Spec["system"])
			}
			// Check tags
			foundDomainTag := false
			for _, tag := range e.Metadata.Tags {
				if tag == "domain" {
					foundDomainTag = true
				}
			}
			if !foundDomainTag {
				t.Error("Expected 'domain' tag on API entity")
			}
		}
	}
	if !foundAPI {
		t.Error("Expected an API entity for the domain")
	}
}

func TestBackstageGetCatalogByKindSystem(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog/:kind", handler.GetCatalogByKind)

	// Create project (System) and app (Component)
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

	// Get only System entities (should exclude Component)
	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog/System", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 1 {
		t.Errorf("Expected 1 System entity, got %d", result.Total)
	}
	for _, e := range result.Items {
		if e.Kind != "System" {
			t.Errorf("Expected only System entities, got %s", e.Kind)
		}
	}
}

func TestBackstageGetCatalogByKindResource(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog/:kind", handler.GetCatalogByKind)

	// Create project + database (Resource)
	projSpec, _ := json.Marshal(map[string]interface{}{"displayName": "Test"})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8s.ObjectMeta{Name: "proj1"},
		Spec:       projSpec,
	})

	dbSpec, _ := json.Marshal(map[string]interface{}{"engine": "postgresql", "version": "16"})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata: k8s.ObjectMeta{
			Name:      "db1",
			Namespace: "zenith-proj1",
		},
		Spec: dbSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog/Resource", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total < 1 {
		t.Errorf("Expected at least 1 Resource, got %d", result.Total)
	}
	for _, e := range result.Items {
		if e.Kind != "Resource" {
			t.Errorf("Expected only Resource entities, got %s", e.Kind)
		}
	}
}

func TestBackstageGetCatalogByKindAPI(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog/:kind", handler.GetCatalogByKind)

	projSpec, _ := json.Marshal(map[string]interface{}{"displayName": "Test"})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8s.ObjectMeta{Name: "proj1"},
		Spec:       projSpec,
	})

	domainSpec, _ := json.Marshal(map[string]interface{}{"domain": "api.test.com", "appRef": "web"})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Domain",
		Metadata: k8s.ObjectMeta{
			Name:      "test-domain",
			Namespace: "zenith-proj1",
		},
		Spec: domainSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog/API", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total < 1 {
		t.Errorf("Expected at least 1 API entity, got %d", result.Total)
	}
	for _, e := range result.Items {
		if e.Kind != "API" {
			t.Errorf("Expected only API entities, got %s", e.Kind)
		}
	}
}

func TestBackstageGetCatalogByKindEmptyResult(t *testing.T) {
	app, handler, _ := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog/:kind", handler.GetCatalogByKind)

	// No data - Component filter should return empty
	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog/Component", nil)
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

func TestBackstageComponentWithDomain(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Test",
		"owner":       "user@test.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name:   "proj1",
			Labels: map[string]string{"zenith.dev/owner": "user@test.com"},
		},
		Spec: projSpec,
	})

	// Create an app with a domain - should have providesApis
	appSpec, _ := json.Marshal(map[string]interface{}{
		"image":  "nginx:latest",
		"domain": "web.example.com",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: k8s.ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-proj1",
			Labels:    map[string]string{"zenith.dev/app-name": "web"},
		},
		Spec: appSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Find Component entity and check providesApis
	for _, e := range result.Items {
		if e.Kind == "Component" && e.Metadata.Name == "web" {
			apis, ok := e.Spec["providesApis"].([]interface{})
			if !ok {
				t.Error("Expected providesApis in Component spec when domain is set")
			} else if len(apis) == 0 {
				t.Error("Expected at least one API in providesApis")
			}
		}
	}
}

func TestBackstageProjectOwnerFallback(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	// Create project with no owner set
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "No Owner",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8s.ObjectMeta{Name: "noowner"},
		Spec:       projSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	for _, e := range result.Items {
		if e.Kind == "System" && e.Metadata.Name == "noowner" {
			owner, _ := e.Spec["owner"].(string)
			if owner != "zenith" {
				t.Errorf("Expected owner fallback 'zenith', got '%s'", owner)
			}
		}
	}
}

func TestBackstageDatabaseResourceTags(t *testing.T) {
	app, handler, memClient := setupBackstageTest()
	app.Get("/api/v1/backstage/catalog", handler.GetCatalog)

	projSpec, _ := json.Marshal(map[string]interface{}{"displayName": "Test"})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8s.ObjectMeta{Name: "proj1"},
		Spec:       projSpec,
	})

	dbSpec, _ := json.Marshal(map[string]interface{}{
		"engine":  "postgresql",
		"version": "16",
	})
	memClient.CreateCRD(nil, &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata: k8s.ObjectMeta{
			Name:      "main-db",
			Namespace: "zenith-proj1",
		},
		Spec: dbSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/backstage/catalog", nil)
	resp, _ := app.Test(req)

	var result handlers.BackstageCatalogResponse
	json.NewDecoder(resp.Body).Decode(&result)

	for _, e := range result.Items {
		if e.Kind == "Resource" && e.Metadata.Name == "main-db" {
			// Should have tags: zenith, database, postgresql
			expectedTags := map[string]bool{"zenith": false, "database": false, "postgresql": false}
			for _, tag := range e.Metadata.Tags {
				if _, ok := expectedTags[tag]; ok {
					expectedTags[tag] = true
				}
			}
			for tag, found := range expectedTags {
				if !found {
					t.Errorf("Expected tag '%s' on database Resource entity", tag)
				}
			}
			// Check description
			if e.Metadata.Description != "postgresql 16 database" {
				t.Errorf("Expected description 'postgresql 16 database', got '%s'", e.Metadata.Description)
			}
		}
	}
}
