package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

func setupMeteringApp() (*fiber.App, *handlers.MeteringHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	meteringStore := store.NewMemoryMeteringRepository()
	handler := handlers.NewMeteringHandler(meteringStore, customerStore)
	return app, handler
}

// ---------- RecordUsage ----------

func TestRecordUsage(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Post("/api/v1/internal/metering", handler.RecordUsage)

	body := `{"customerId":"cust-001","cpuCores":10.5,"ramGb":20.0,"s3Tb":0.5,"dbStorageGb":45.0,"volumeGb":200.0,"lbCount":2}`
	req := httptest.NewRequest("POST", "/api/v1/internal/metering", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(b))
	}

	var usage models.ResourceUsage
	json.NewDecoder(resp.Body).Decode(&usage)

	if usage.CustomerID != "cust-001" {
		t.Errorf("Expected customerId 'cust-001', got '%s'", usage.CustomerID)
	}
	if usage.CPUCores != 10.5 {
		t.Errorf("Expected cpuCores 10.5, got %f", usage.CPUCores)
	}
	if usage.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestRecordUsageInvalidCustomer(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Post("/api/v1/internal/metering", handler.RecordUsage)

	body := `{"customerId":"nonexistent","cpuCores":1.0,"ramGb":2.0}`
	req := httptest.NewRequest("POST", "/api/v1/internal/metering", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid customer, got %d", resp.StatusCode)
	}
}

func TestRecordUsageMissingCustomerID(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Post("/api/v1/internal/metering", handler.RecordUsage)

	body := `{"cpuCores":1.0,"ramGb":2.0}`
	req := httptest.NewRequest("POST", "/api/v1/internal/metering", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing customerId, got %d", resp.StatusCode)
	}
}

// ---------- GetCustomerUsage ----------

func TestGetCustomerUsage(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/usage", handler.GetCustomerUsage)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/cust-001/usage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var usage models.CustomerUsage
	json.NewDecoder(resp.Body).Decode(&usage)

	// Memory repo is seeded with data for cust-001 (Embermind, Pro plan)
	if usage.CPUCeiling != 16 {
		t.Errorf("Expected CPU ceiling 16 (Pro plan), got %d", usage.CPUCeiling)
	}
	if usage.RAMCeiling != 32 {
		t.Errorf("Expected RAM ceiling 32 (Pro plan), got %d", usage.RAMCeiling)
	}
	if usage.CPUCores <= 0 {
		t.Errorf("Expected non-zero CPU usage, got %f", usage.CPUCores)
	}
	if usage.CPUPercent <= 0 {
		t.Errorf("Expected non-zero CPU percent, got %f", usage.CPUPercent)
	}
}

func TestGetCustomerUsageNoData(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	// Create metering repo WITHOUT seed data
	meteringStore := &emptyMeteringRepo{}
	handler := handlers.NewMeteringHandler(meteringStore, customerStore)

	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/usage", handler.GetCustomerUsage)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/cust-001/usage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var usage models.CustomerUsage
	json.NewDecoder(resp.Body).Decode(&usage)

	// Should return zero usage with ceilings
	if usage.CPUCores != 0 {
		t.Errorf("Expected zero CPU usage, got %f", usage.CPUCores)
	}
	if usage.CPUCeiling != 16 {
		t.Errorf("Expected CPU ceiling 16, got %d", usage.CPUCeiling)
	}
}

func TestGetCustomerUsageNotFound(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/usage", handler.GetCustomerUsage)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/nonexistent/usage", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// ---------- GetCustomerUsageHistory ----------

func TestGetCustomerUsageHistory(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/usage/history", handler.GetCustomerUsageHistory)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/cust-001/usage/history?days=30", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var history []models.UsageHistoryEntry
	json.NewDecoder(resp.Body).Decode(&history)

	if len(history) == 0 {
		t.Error("Expected non-empty usage history")
	}

	// Verify entries have dates and non-zero values
	for _, e := range history {
		if e.Date == "" {
			t.Error("Expected non-empty date")
		}
		if e.CPUAvg <= 0 {
			t.Errorf("Expected positive CPU avg, got %f", e.CPUAvg)
		}
	}
}

func TestGetCustomerUsageHistoryNotFound(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/usage/history", handler.GetCustomerUsageHistory)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/nonexistent/usage/history", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// ---------- GetPlatformUsageSummary ----------

func TestGetPlatformUsageSummary(t *testing.T) {
	app, handler := setupMeteringApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/dashboard/usage", handler.GetPlatformUsageSummary)

	req := httptest.NewRequest("GET", "/api/v1/admin/dashboard/usage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var summary models.PlatformUsageSummary
	json.NewDecoder(resp.Body).Decode(&summary)

	if summary.CustomersReporting != 3 {
		t.Errorf("Expected 3 customers reporting, got %d", summary.CustomersReporting)
	}
	if summary.TotalCPU <= 0 {
		t.Errorf("Expected positive total CPU, got %f", summary.TotalCPU)
	}
	if summary.TotalRAM <= 0 {
		t.Errorf("Expected positive total RAM, got %f", summary.TotalRAM)
	}
}

// ---------- Internal Secret Auth ----------

func TestInternalSecretAuthMissing(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	meteringStore := store.NewMemoryMeteringRepository()
	handler := handlers.NewMeteringHandler(meteringStore, customerStore)

	internal := app.Group("/api/v1/internal", middleware.RequireInternalSecret("test-secret"))
	internal.Post("/metering", handler.RecordUsage)

	body := `{"customerId":"cust-001","cpuCores":1.0}`
	req := httptest.NewRequest("POST", "/api/v1/internal/metering", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 for missing secret, got %d", resp.StatusCode)
	}
}

func TestInternalSecretAuthInvalid(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	meteringStore := store.NewMemoryMeteringRepository()
	handler := handlers.NewMeteringHandler(meteringStore, customerStore)

	internal := app.Group("/api/v1/internal", middleware.RequireInternalSecret("test-secret"))
	internal.Post("/metering", handler.RecordUsage)

	body := `{"customerId":"cust-001","cpuCores":1.0}`
	req := httptest.NewRequest("POST", "/api/v1/internal/metering", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "wrong-secret")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for invalid secret, got %d", resp.StatusCode)
	}
}

func TestInternalSecretAuthValid(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	meteringStore := store.NewMemoryMeteringRepository()
	handler := handlers.NewMeteringHandler(meteringStore, customerStore)

	internal := app.Group("/api/v1/internal", middleware.RequireInternalSecret("test-secret"))
	internal.Post("/metering", handler.RecordUsage)

	body := `{"customerId":"cust-001","cpuCores":10.0,"ramGb":20.0}`
	req := httptest.NewRequest("POST", "/api/v1/internal/metering", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "test-secret")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 201 for valid secret, got %d: %s", resp.StatusCode, string(b))
	}
}

// ---------- Helpers ----------

// emptyMeteringRepo is a MeteringRepository that always returns "no data".
type emptyMeteringRepo struct{}

func (r *emptyMeteringRepo) RecordUsage(_ context.Context, _ *models.MeteringInput) (*models.ResourceUsage, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *emptyMeteringRepo) GetLatestUsage(_ context.Context, _ string) (*models.ResourceUsage, error) {
	return nil, fmt.Errorf("no usage data found")
}
func (r *emptyMeteringRepo) GetUsageHistory(_ context.Context, _ string, _ int) ([]models.UsageHistoryEntry, error) {
	return []models.UsageHistoryEntry{}, nil
}
func (r *emptyMeteringRepo) GetPlatformUsageSummary(_ context.Context) (*models.PlatformUsageSummary, error) {
	return &models.PlatformUsageSummary{}, nil
}
