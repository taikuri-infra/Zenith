package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func setupManagedServiceTest() (*fiber.App, *handlers.ManagedServiceHandler, *memory.MemoryProjectRepository, *memory.MemoryManagedServiceRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	projectRepo := memory.NewMemoryProjectRepository()
	msRepo := memory.NewMemoryManagedServiceRepository()
	handler := handlers.NewManagedServiceHandler(projectRepo, msRepo)
	return app, handler, projectRepo, msRepo
}

func TestManagedServiceProvision(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-1"), handler.Provision)

	body := `{"service_type":"postgresql","name":"mydb","version":"16","storage_gb":10}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "mydb" {
		t.Errorf("Expected name 'mydb', got '%v'", result["name"])
	}
	if result["service_type"] != "postgresql" {
		t.Errorf("Expected service_type 'postgresql', got '%v'", result["service_type"])
	}
}

func TestManagedServiceProvisionNoName(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-1"), handler.Provision)

	body := `{"service_type":"postgresql"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestManagedServiceProvisionNoServiceType(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-1"), handler.Provision)

	body := `{"name":"mydb"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestManagedServiceProvisionInvalidType(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-1"), handler.Provision)

	body := `{"service_type":"oracle","name":"mydb"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestManagedServiceProvisionProjectNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupManagedServiceTest()

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-1"), handler.Provision)

	body := `{"service_type":"postgresql","name":"mydb"}`
	req := httptest.NewRequest("POST", "/projects/nonexistent/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestManagedServiceProvisionForbidden(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-2"), handler.Provision)

	body := `{"service_type":"postgresql","name":"mydb"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestManagedServiceProvisionNoAuth(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", handler.Provision)

	body := `{"service_type":"postgresql","name":"mydb"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestManagedServiceList(t *testing.T) {
	fiberApp, handler, projectRepo, msRepo := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	// Create managed services directly
	msRepo.CreateManagedService(nil, &entities.ManagedService{
		ID:          uuid.New().String(),
		ProjectID:   project.ID,
		UserID:      "user-1",
		ServiceType: entities.ServiceTypePostgreSQL,
		Name:        "db1",
		Version:     "16",
		Port:        5432,
		Status:      entities.ManagedServiceProvisioning,
		StorageGB:   10,
	})
	msRepo.CreateManagedService(nil, &entities.ManagedService{
		ID:          uuid.New().String(),
		ProjectID:   project.ID,
		UserID:      "user-1",
		ServiceType: entities.ServiceTypeRedis,
		Name:        "cache1",
		Version:     "7",
		Port:        6379,
		Status:      entities.ManagedServiceProvisioning,
		StorageGB:   5,
	})

	fiberApp.Get("/projects/:projectId/managed-services", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/managed-services", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 services, got %d", result.Total)
	}
}

func TestManagedServiceListEmpty(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Get("/projects/:projectId/managed-services", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/managed-services", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0, got %d", result.Total)
	}
}

func TestManagedServiceGet(t *testing.T) {
	fiberApp, handler, projectRepo, msRepo := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	msID := uuid.New().String()
	msRepo.CreateManagedService(nil, &entities.ManagedService{
		ID:          msID,
		ProjectID:   project.ID,
		UserID:      "user-1",
		ServiceType: entities.ServiceTypePostgreSQL,
		Name:        "mydb",
		Version:     "16",
		Port:        5432,
		Status:      entities.ManagedServiceProvisioning,
		StorageGB:   10,
	})

	fiberApp.Get("/projects/:projectId/managed-services/:msId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/managed-services/"+msID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "mydb" {
		t.Errorf("Expected name 'mydb', got '%v'", result["name"])
	}
}

func TestManagedServiceGetNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupManagedServiceTest()

	fiberApp.Get("/projects/:projectId/managed-services/:msId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/projects/proj-1/managed-services/nonexistent", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestManagedServiceDelete(t *testing.T) {
	fiberApp, handler, projectRepo, msRepo := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	msID := uuid.New().String()
	msRepo.CreateManagedService(nil, &entities.ManagedService{
		ID:          msID,
		ProjectID:   project.ID,
		UserID:      "user-1",
		ServiceType: entities.ServiceTypePostgreSQL,
		Name:        "mydb",
		Version:     "16",
		Port:        5432,
		Status:      entities.ManagedServiceProvisioning,
		StorageGB:   10,
	})

	fiberApp.Delete("/projects/:projectId/managed-services/:msId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/projects/"+project.ID+"/managed-services/"+msID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestManagedServiceDeleteNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupManagedServiceTest()

	fiberApp.Delete("/projects/:projectId/managed-services/:msId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/projects/proj-1/managed-services/nonexistent", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestManagedServiceDeleteForbidden(t *testing.T) {
	fiberApp, handler, projectRepo, msRepo := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	msID := uuid.New().String()
	msRepo.CreateManagedService(nil, &entities.ManagedService{
		ID:          msID,
		ProjectID:   project.ID,
		UserID:      "user-1",
		ServiceType: entities.ServiceTypePostgreSQL,
		Name:        "mydb",
		Version:     "16",
		Port:        5432,
		Status:      entities.ManagedServiceProvisioning,
		StorageGB:   10,
	})

	fiberApp.Delete("/projects/:projectId/managed-services/:msId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/projects/"+project.ID+"/managed-services/"+msID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestManagedServiceProvisionRedis(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupManagedServiceTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Post("/projects/:projectId/managed-services", injectUserID("user-1"), handler.Provision)

	body := `{"service_type":"redis","name":"cache"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/managed-services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["service_type"] != "redis" {
		t.Errorf("Expected 'redis', got '%v'", result["service_type"])
	}
}
