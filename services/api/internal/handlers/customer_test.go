package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/capi"
	"github.com/dotechhq/zenith/services/api/internal/cluster"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

func setupCustomerApp() (*fiber.App, *handlers.CustomerHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	adminStore := store.NewMemoryAdminRepository()
	handler := handlers.NewCustomerHandler(customerStore, adminStore, nil)
	return app, handler
}

// ---------- Plans ----------

func TestListPlans(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/plans", handler.ListPlans)

	req := httptest.NewRequest("GET", "/api/v1/admin/plans", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var plans []entities.Plan
	json.NewDecoder(resp.Body).Decode(&plans)

	if len(plans) != 3 {
		t.Errorf("Expected 3 seeded plans, got %d", len(plans))
	}
}

func TestCreatePlan(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/plans", handler.CreatePlan)

	body := `{"name":"Micro","cpuCores":2,"ramGb":4,"priceCents":4900}`
	req := httptest.NewRequest("POST", "/api/v1/admin/plans", bytes.NewBufferString(body))
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

	var plan entities.Plan
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan.Name != "Micro" {
		t.Errorf("Expected name 'Micro', got '%s'", plan.Name)
	}
	if plan.CPUCores != 2 {
		t.Errorf("Expected 2 CPU cores, got %d", plan.CPUCores)
	}
	if plan.Currency != "EUR" {
		t.Errorf("Expected default currency EUR, got '%s'", plan.Currency)
	}
	if !plan.Active {
		t.Error("Expected plan to be active by default")
	}
}

func TestCreatePlanMissingName(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/plans", handler.CreatePlan)

	body := `{"cpuCores":2,"ramGb":4,"priceCents":4900}`
	req := httptest.NewRequest("POST", "/api/v1/admin/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePlanDuplicateName(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/plans", handler.CreatePlan)

	// "Starter" already exists in seed data
	body := `{"name":"Starter","cpuCores":2,"ramGb":4,"priceCents":4900}`
	req := httptest.NewRequest("POST", "/api/v1/admin/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409 for duplicate name, got %d", resp.StatusCode)
	}
}

func TestUpdatePlan(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/plans/:id", handler.UpdatePlan)

	body := `{"priceCents":12900}`
	req := httptest.NewRequest("PUT", "/api/v1/admin/plans/plan-starter", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var plan entities.Plan
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan.PriceCents != 12900 {
		t.Errorf("Expected price 12900, got %d", plan.PriceCents)
	}
	if plan.Name != "Starter" {
		t.Errorf("Expected name 'Starter' unchanged, got '%s'", plan.Name)
	}
}

func TestUpdatePlanNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/plans/:id", handler.UpdatePlan)

	body := `{"priceCents":12900}`
	req := httptest.NewRequest("PUT", "/api/v1/admin/plans/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// ---------- Customers ----------

func TestListCustomers(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers", handler.ListCustomers)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var customers []entities.Customer
	json.NewDecoder(resp.Body).Decode(&customers)

	if len(customers) != 3 {
		t.Errorf("Expected 3 seeded customers, got %d", len(customers))
	}

	// Verify plan is populated
	for _, c := range customers {
		if c.Plan == nil {
			t.Errorf("Expected plan to be populated for customer %s", c.Name)
		}
	}
}

func TestCreateCustomer(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	body := `{"name":"NewCo","domain":"newco.dev","planId":"plan-starter","contactEmail":"hi@newco.dev","contactName":"Jane Doe"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(body))
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

	var customer entities.Customer
	json.NewDecoder(resp.Body).Decode(&customer)

	if customer.Name != "NewCo" {
		t.Errorf("Expected name 'NewCo', got '%s'", customer.Name)
	}
	if customer.Domain != "newco.dev" {
		t.Errorf("Expected domain 'newco.dev', got '%s'", customer.Domain)
	}
	if customer.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.Status)
	}
	if customer.ClusterStatus != "pending" {
		t.Errorf("Expected cluster status 'pending', got '%s'", customer.ClusterStatus)
	}
}

func TestCreateCustomerMissingName(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	body := `{"domain":"x.com","planId":"plan-starter","contactEmail":"a@x.com"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateCustomerMissingDomain(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	body := `{"name":"X","planId":"plan-starter","contactEmail":"a@x.com"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateCustomerDuplicateDomain(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	// "embermind.app" already exists in seed data
	body := `{"name":"Dup","domain":"embermind.app","planId":"plan-starter","contactEmail":"a@dup.com"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409 for duplicate domain, got %d", resp.StatusCode)
	}
}

func TestCreateCustomerInvalidPlan(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	body := `{"name":"X","domain":"x.com","planId":"nonexistent","contactEmail":"a@x.com"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid plan, got %d", resp.StatusCode)
	}
}

func TestGetCustomer(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id", handler.GetCustomer)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/cust-001", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var customer entities.Customer
	json.NewDecoder(resp.Body).Decode(&customer)

	if customer.Name != "Embermind" {
		t.Errorf("Expected name 'Embermind', got '%s'", customer.Name)
	}
	if customer.Plan == nil {
		t.Error("Expected plan to be populated")
	} else if customer.Plan.Name != "Pro" {
		t.Errorf("Expected plan 'Pro', got '%s'", customer.Plan.Name)
	}
}

func TestGetCustomerNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id", handler.GetCustomer)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateCustomer(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/customers/:id", handler.UpdateCustomer)

	body := `{"name":"Embermind LLC","notes":"VIP customer"}`
	req := httptest.NewRequest("PUT", "/api/v1/admin/customers/cust-001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var customer entities.Customer
	json.NewDecoder(resp.Body).Decode(&customer)

	if customer.Name != "Embermind LLC" {
		t.Errorf("Expected name 'Embermind LLC', got '%s'", customer.Name)
	}
	if customer.Notes != "VIP customer" {
		t.Errorf("Expected notes 'VIP customer', got '%s'", customer.Notes)
	}
	// Domain should remain unchanged
	if customer.Domain != "embermind.app" {
		t.Errorf("Expected domain unchanged, got '%s'", customer.Domain)
	}
}

func TestUpdateCustomerNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/customers/:id", handler.UpdateCustomer)

	body := `{"name":"X"}`
	req := httptest.NewRequest("PUT", "/api/v1/admin/customers/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteCustomer(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Delete("/api/v1/admin/customers/:id", handler.DeleteCustomer)
	app.Get("/api/v1/admin/customers/:id", handler.GetCustomer)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/admin/customers/cust-003", nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		b, _ := io.ReadAll(deleteResp.Body)
		t.Fatalf("Expected 200, got %d: %s", deleteResp.StatusCode, string(b))
	}

	// Verify deleted
	getReq := httptest.NewRequest("GET", "/api/v1/admin/customers/cust-003", nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

func TestDeleteCustomerNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Delete("/api/v1/admin/customers/:id", handler.DeleteCustomer)

	req := httptest.NewRequest("DELETE", "/api/v1/admin/customers/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// ---------- Suspend / Activate ----------

func TestSuspendCustomer(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/suspend", handler.SuspendCustomer)

	req := httptest.NewRequest("POST", "/api/v1/admin/customers/cust-001/suspend", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var customer entities.Customer
	json.NewDecoder(resp.Body).Decode(&customer)

	if customer.Status != "suspended" {
		t.Errorf("Expected status 'suspended', got '%s'", customer.Status)
	}
}

func TestSuspendCustomerNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/suspend", handler.SuspendCustomer)

	req := httptest.NewRequest("POST", "/api/v1/admin/customers/nonexistent/suspend", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestActivateCustomer(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/suspend", handler.SuspendCustomer)
	app.Post("/api/v1/admin/customers/:id/activate", handler.ActivateCustomer)

	// Suspend first
	suspendReq := httptest.NewRequest("POST", "/api/v1/admin/customers/cust-001/suspend", nil)
	app.Test(suspendReq)

	// Activate
	activateReq := httptest.NewRequest("POST", "/api/v1/admin/customers/cust-001/activate", nil)
	activateResp, _ := app.Test(activateReq)
	defer activateResp.Body.Close()

	if activateResp.StatusCode != 200 {
		b, _ := io.ReadAll(activateResp.Body)
		t.Fatalf("Expected 200, got %d: %s", activateResp.StatusCode, string(b))
	}

	var customer entities.Customer
	json.NewDecoder(activateResp.Body).Decode(&customer)

	if customer.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.Status)
	}
}

func TestActivateCustomerNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/activate", handler.ActivateCustomer)

	req := httptest.NewRequest("POST", "/api/v1/admin/customers/nonexistent/activate", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// ---------- Stats ----------

func TestGetCustomerStats(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/stats", handler.GetCustomerStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var stats entities.CustomerStats
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats.TotalCustomers != 3 {
		t.Errorf("Expected 3 total customers, got %d", stats.TotalCustomers)
	}
	if stats.ActiveCustomers != 3 {
		t.Errorf("Expected 3 active customers, got %d", stats.ActiveCustomers)
	}
	if stats.MRR == "" {
		t.Error("Expected non-empty MRR")
	}
}

func TestGetCustomerStatsAfterSuspend(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/suspend", handler.SuspendCustomer)
	app.Get("/api/v1/admin/customers/stats", handler.GetCustomerStats)

	// Suspend one customer
	suspendReq := httptest.NewRequest("POST", "/api/v1/admin/customers/cust-001/suspend", nil)
	app.Test(suspendReq)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/stats", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	var stats entities.CustomerStats
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats.TotalCustomers != 3 {
		t.Errorf("Expected 3 total, got %d", stats.TotalCustomers)
	}
	if stats.ActiveCustomers != 2 {
		t.Errorf("Expected 2 active after suspend, got %d", stats.ActiveCustomers)
	}
}

// ---------- Full CRUD flow ----------

func TestCustomerCRUDFlow(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)
	app.Get("/api/v1/admin/customers/:id", handler.GetCustomer)
	app.Put("/api/v1/admin/customers/:id", handler.UpdateCustomer)
	app.Post("/api/v1/admin/customers/:id/suspend", handler.SuspendCustomer)
	app.Post("/api/v1/admin/customers/:id/activate", handler.ActivateCustomer)
	app.Delete("/api/v1/admin/customers/:id", handler.DeleteCustomer)

	// Create
	createBody := `{"name":"FlowCo","domain":"flow.co","planId":"plan-pro","contactEmail":"admin@flow.co","contactName":"Flow Admin"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	if createResp.StatusCode != 201 {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("Create: Expected 201, got %d: %s", createResp.StatusCode, string(b))
	}

	var created entities.Customer
	json.NewDecoder(createResp.Body).Decode(&created)
	id := created.ID

	// Get
	getReq := httptest.NewRequest("GET", "/api/v1/admin/customers/"+id, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Get: Expected 200, got %d", getResp.StatusCode)
	}

	var fetched entities.Customer
	json.NewDecoder(getResp.Body).Decode(&fetched)

	if fetched.Name != "FlowCo" {
		t.Errorf("Get: Expected name 'FlowCo', got '%s'", fetched.Name)
	}

	// Update
	updateBody := `{"name":"FlowCo Inc."}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/admin/customers/"+id, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 200 {
		t.Fatalf("Update: Expected 200, got %d", updateResp.StatusCode)
	}

	var updated entities.Customer
	json.NewDecoder(updateResp.Body).Decode(&updated)

	if updated.Name != "FlowCo Inc." {
		t.Errorf("Update: Expected name 'FlowCo Inc.', got '%s'", updated.Name)
	}

	// Suspend
	suspendReq := httptest.NewRequest("POST", "/api/v1/admin/customers/"+id+"/suspend", nil)
	suspendResp, _ := app.Test(suspendReq)

	if suspendResp.StatusCode != 200 {
		t.Fatalf("Suspend: Expected 200, got %d", suspendResp.StatusCode)
	}

	var suspended entities.Customer
	json.NewDecoder(suspendResp.Body).Decode(&suspended)

	if suspended.Status != "suspended" {
		t.Errorf("Suspend: Expected 'suspended', got '%s'", suspended.Status)
	}

	// Activate
	activateReq := httptest.NewRequest("POST", "/api/v1/admin/customers/"+id+"/activate", nil)
	activateResp, _ := app.Test(activateReq)

	if activateResp.StatusCode != 200 {
		t.Fatalf("Activate: Expected 200, got %d", activateResp.StatusCode)
	}

	var activated entities.Customer
	json.NewDecoder(activateResp.Body).Decode(&activated)

	if activated.Status != "active" {
		t.Errorf("Activate: Expected 'active', got '%s'", activated.Status)
	}

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/admin/customers/"+id, nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Delete: Expected 200, got %d", deleteResp.StatusCode)
	}

	// Verify deleted
	verifyReq := httptest.NewRequest("GET", "/api/v1/admin/customers/"+id, nil)
	verifyResp, _ := app.Test(verifyReq)

	if verifyResp.StatusCode != 404 {
		t.Errorf("Verify: Expected 404, got %d", verifyResp.StatusCode)
	}
}

// ---------- Cluster endpoints ----------

func setupCustomerAppWithProvisioner() (*fiber.App, *handlers.CustomerHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	customerStore := store.NewMemoryCustomerRepository()
	adminStore := store.NewMemoryAdminRepository()
	// Create provisioner with in-memory CAPI
	k8sClient := k8s.NewMemoryClient()
	capiClient := capi.NewClient(k8sClient)
	provisioner := cluster.NewProvisioner(capiClient, customerStore, adminStore)
	handler := handlers.NewCustomerHandler(customerStore, adminStore, provisioner)
	return app, handler
}

func TestGetCustomerCluster(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/cluster", handler.GetCustomerCluster)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/cust-001/cluster", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestGetCustomerClusterNotFound(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/customers/:id/cluster", handler.GetCustomerCluster)

	req := httptest.NewRequest("GET", "/api/v1/admin/customers/nonexistent/cluster", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestScaleClusterEndpoint(t *testing.T) {
	app, handler := setupCustomerAppWithProvisioner()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)
	app.Post("/api/v1/admin/customers/:id/cluster/scale", handler.ScaleCluster)

	// Create a customer first (triggers provisioning)
	createBody := `{"name":"ScaleTest","domain":"scaletest.dev","planId":"plan-starter","contactEmail":"a@s.dev","contactName":"Admin"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created entities.Customer
	json.NewDecoder(createResp.Body).Decode(&created)

	// Give goroutine time to provision
	// Scale
	scaleBody := `{"nodes":5}`
	scaleReq := httptest.NewRequest("POST", "/api/v1/admin/customers/"+created.ID+"/cluster/scale", bytes.NewBufferString(scaleBody))
	scaleReq.Header.Set("Content-Type", "application/json")
	scaleResp, err := app.Test(scaleReq)
	if err != nil {
		t.Fatal(err)
	}
	defer scaleResp.Body.Close()

	if scaleResp.StatusCode != 200 {
		b, _ := io.ReadAll(scaleResp.Body)
		t.Fatalf("Expected 200, got %d: %s", scaleResp.StatusCode, string(b))
	}
}

func TestScaleClusterBadNodes(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/cluster/scale", handler.ScaleCluster)

	body := `{"nodes":0}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers/cust-001/cluster/scale", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for 0 nodes, got %d", resp.StatusCode)
	}
}

func TestUpgradeClusterEndpoint(t *testing.T) {
	app, handler := setupCustomerAppWithProvisioner()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)
	app.Post("/api/v1/admin/customers/:id/cluster/upgrade", handler.UpgradeCluster)

	// Create a customer first
	createBody := `{"name":"UpgradeTest","domain":"upgradetest.dev","planId":"plan-starter","contactEmail":"a@u.dev","contactName":"Admin"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created entities.Customer
	json.NewDecoder(createResp.Body).Decode(&created)

	// Allow async provisioning goroutine to create the cluster CRD
	time.Sleep(100 * time.Millisecond)

	// Upgrade
	upgradeBody := `{"version":"v1.32.0"}`
	upgradeReq := httptest.NewRequest("POST", "/api/v1/admin/customers/"+created.ID+"/cluster/upgrade", bytes.NewBufferString(upgradeBody))
	upgradeReq.Header.Set("Content-Type", "application/json")
	upgradeResp, err := app.Test(upgradeReq)
	if err != nil {
		t.Fatal(err)
	}
	defer upgradeResp.Body.Close()

	if upgradeResp.StatusCode != 200 {
		b, _ := io.ReadAll(upgradeResp.Body)
		t.Fatalf("Expected 200, got %d: %s", upgradeResp.StatusCode, string(b))
	}
}

func TestCustomerUpgradeClusterMissingVersion(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers/:id/cluster/upgrade", handler.UpgradeCluster)

	body := `{"version":""}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers/cust-001/cluster/upgrade", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for empty version, got %d", resp.StatusCode)
	}
}

func TestCreateCustomerSetsClusterFields(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	body := `{"name":"ClusterCo","domain":"cluster.co","planId":"plan-starter","contactEmail":"a@c.co","contactName":"Admin"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(b))
	}

	var customer entities.Customer
	json.NewDecoder(resp.Body).Decode(&customer)

	if customer.CAPIClusterName != "cluster-co" {
		t.Errorf("Expected capiClusterName 'cluster-co', got '%s'", customer.CAPIClusterName)
	}
	if customer.ClusterRegion != "fsn1" {
		t.Errorf("Expected clusterRegion 'fsn1', got '%s'", customer.ClusterRegion)
	}
	if customer.ClusterNodes != 3 {
		t.Errorf("Expected clusterNodes 3, got %d", customer.ClusterNodes)
	}
	if customer.ClusterK8sVersion != "v1.31.2" {
		t.Errorf("Expected clusterK8sVersion 'v1.31.2', got '%s'", customer.ClusterK8sVersion)
	}
}

// ---------- Invalid body tests ----------

func TestCreateCustomerInvalidBody(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/customers", handler.CreateCustomer)

	req := httptest.NewRequest("POST", "/api/v1/admin/customers", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePlanInvalidBody(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/plans", handler.CreatePlan)

	req := httptest.NewRequest("POST", "/api/v1/admin/plans", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateCustomerInvalidBody(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/customers/:id", handler.UpdateCustomer)

	req := httptest.NewRequest("PUT", "/api/v1/admin/customers/cust-001", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdatePlanInvalidBody(t *testing.T) {
	app, handler := setupCustomerApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/plans/:id", handler.UpdatePlan)

	req := httptest.NewRequest("PUT", "/api/v1/admin/plans/plan-starter", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
