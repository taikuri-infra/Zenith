package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupStorageV2Test() (*fiber.App, *handlers.StorageHandlerV2, *memory.MemoryStorageRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	storageRepo := memory.NewMemoryStorageRepository()
	appRepo := memory.NewMemoryAppRepository()
	handler := handlers.NewStorageHandlerV2(storageRepo, appRepo, nil)

	// Create a test app owned by user-1
	appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})

	return app, handler, storageRepo
}

func getTestAppID(t *testing.T) string {
	t.Helper()
	appRepo := memory.NewMemoryAppRepository()
	app, err := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	return app.ID
}

func setupStorageV2TestWithApp() (*fiber.App, *handlers.StorageHandlerV2, *memory.MemoryStorageRepository, string) {
	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	storageRepo := memory.NewMemoryStorageRepository()
	appRepo := memory.NewMemoryAppRepository()
	handler := handlers.NewStorageHandlerV2(storageRepo, appRepo, nil)

	app, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})

	return fiberApp, handler, storageRepo, app.ID
}

func TestStorageV2Create(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/apps/:appId/storage", injectUserID("user-1"), handler.Create)

	body := `{"name":"uploads"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "uploads" {
		t.Errorf("Expected name 'uploads', got '%s'", result.Name)
	}
	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if result.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", result.Status)
	}
}

func TestStorageV2CreateNoName(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/apps/:appId/storage", injectUserID("user-1"), handler.Create)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestStorageV2CreateAppNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/apps/:appId/storage", injectUserID("user-1"), handler.Create)

	body := `{"name":"uploads"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestStorageV2CreateForbidden(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/apps/:appId/storage", injectUserID("user-2"), handler.Create)

	body := `{"name":"uploads"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestStorageV2List(t *testing.T) {
	fiberApp, handler, storageRepo, appID := setupStorageV2TestWithApp()

	storageRepo.CreateBucket(nil, appID, "user-1", &dto.CreateBucketInput{Name: "bucket1"})
	storageRepo.CreateBucket(nil, appID, "user-1", &dto.CreateBucketInput{Name: "bucket2"})

	fiberApp.Get("/api/v1/apps/:appId/storage", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/storage", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 buckets, got %d", len(result))
	}
}

func TestStorageV2ListEmpty(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Get("/api/v1/apps/:appId/storage", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/storage", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 0 {
		t.Errorf("Expected 0 buckets, got %d", len(result))
	}
}

func TestStorageV2Get(t *testing.T) {
	fiberApp, handler, storageRepo, appID := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, appID, "user-1", &dto.CreateBucketInput{Name: "mybucket"})

	fiberApp.Get("/api/v1/apps/:appId/storage/:bucketId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/storage/"+bucket.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "mybucket" {
		t.Errorf("Expected 'mybucket', got '%s'", result.Name)
	}
}

func TestStorageV2GetNotFound(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Get("/api/v1/apps/:appId/storage/:bucketId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/storage/nonexistent", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestStorageV2Delete(t *testing.T) {
	fiberApp, handler, storageRepo, appID := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, appID, "user-1", &dto.CreateBucketInput{Name: "todelete"})

	fiberApp.Delete("/api/v1/apps/:appId/storage/:bucketId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/storage/"+bucket.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestStorageV2DeleteForbidden(t *testing.T) {
	fiberApp, handler, storageRepo, appID := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, appID, "user-1", &dto.CreateBucketInput{Name: "todelete"})

	fiberApp.Delete("/api/v1/apps/:appId/storage/:bucketId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/storage/"+bucket.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestStorageV2DeleteNotFound(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Delete("/api/v1/apps/:appId/storage/:bucketId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/storage/nonexistent", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestStorageV2CreateStandalone(t *testing.T) {
	fiberApp, handler, _, _ := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/storage-buckets", injectUserID("user-1"), handler.CreateStandalone)

	body := `{"name":"standalone-bucket"}`
	req := httptest.NewRequest("POST", "/api/v1/storage-buckets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "standalone-bucket" {
		t.Errorf("Expected 'standalone-bucket', got '%s'", result.Name)
	}
}

func TestStorageV2CreateStandaloneNoName(t *testing.T) {
	fiberApp, handler, _, _ := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/storage-buckets", injectUserID("user-1"), handler.CreateStandalone)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/storage-buckets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestStorageV2ListByUser(t *testing.T) {
	fiberApp, handler, storageRepo, _ := setupStorageV2TestWithApp()

	storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "bucket1"})
	storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "bucket2"})

	fiberApp.Get("/api/v1/storage-buckets", injectUserID("user-1"), handler.ListByUser)

	req := httptest.NewRequest("GET", "/api/v1/storage-buckets", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 buckets, got %d", len(result))
	}
}

func TestStorageV2GetStandalone(t *testing.T) {
	fiberApp, handler, storageRepo, _ := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "mybucket"})

	fiberApp.Get("/api/v1/storage-buckets/:bucketId", injectUserID("user-1"), handler.GetStandalone)

	req := httptest.NewRequest("GET", "/api/v1/storage-buckets/"+bucket.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestStorageV2GetStandaloneForbidden(t *testing.T) {
	fiberApp, handler, storageRepo, _ := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "mybucket"})

	fiberApp.Get("/api/v1/storage-buckets/:bucketId", injectUserID("user-2"), handler.GetStandalone)

	req := httptest.NewRequest("GET", "/api/v1/storage-buckets/"+bucket.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestStorageV2DeleteStandalone(t *testing.T) {
	fiberApp, handler, storageRepo, _ := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "todelete"})

	fiberApp.Delete("/api/v1/storage-buckets/:bucketId", injectUserID("user-1"), handler.DeleteStandalone)

	req := httptest.NewRequest("DELETE", "/api/v1/storage-buckets/"+bucket.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestStorageV2UpdateBucket(t *testing.T) {
	fiberApp, handler, storageRepo, _ := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "mybucket"})

	fiberApp.Put("/api/v1/storage-buckets/:bucketId", injectUserID("user-1"), handler.UpdateBucket)

	body := `{"access":"public"}`
	req := httptest.NewRequest("PUT", "/api/v1/storage-buckets/"+bucket.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result dto.BucketInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Access != "public" {
		t.Errorf("Expected access 'public', got '%s'", result.Access)
	}
}

func TestStorageV2UpdateBucketForbidden(t *testing.T) {
	fiberApp, handler, storageRepo, _ := setupStorageV2TestWithApp()

	bucket, _ := storageRepo.CreateBucket(nil, "", "user-1", &dto.CreateBucketInput{Name: "mybucket"})

	fiberApp.Put("/api/v1/storage-buckets/:bucketId", injectUserID("user-2"), handler.UpdateBucket)

	body := `{"access":"public"}`
	req := httptest.NewRequest("PUT", "/api/v1/storage-buckets/"+bucket.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestStorageV2CreateDuplicate(t *testing.T) {
	fiberApp, handler, _, appID := setupStorageV2TestWithApp()
	fiberApp.Post("/api/v1/apps/:appId/storage", injectUserID("user-1"), handler.Create)

	body := `{"name":"mybucket"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	fiberApp.Test(req)

	// Duplicate
	req2 := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/storage", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req2)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409, got %d", resp.StatusCode)
	}
}
