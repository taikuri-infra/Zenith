package handlers_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupCITemplateTest() (*fiber.App, *handlers.CITemplateHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	handler := handlers.NewCITemplateHandler()
	return app, handler
}

func TestCITemplateListTemplates(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates", handler.ListTemplates)

	req := httptest.NewRequest("GET", "/ci-templates", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Frameworks []string `json:"frameworks"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Frameworks) == 0 {
		t.Error("Expected non-empty frameworks list")
	}
}

func TestCITemplateGetTemplateGo(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/go", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/yaml") {
		t.Errorf("Expected Content-Type text/yaml, got '%s'", contentType)
	}
}

func TestCITemplateGetTemplateNextjs(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/nextjs", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCITemplateGetTemplatePython(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/python", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCITemplateGetTemplateRust(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/rust", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCITemplateGetTemplateNodejs(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/nodejs", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCITemplateGetTemplateUnsupported(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/php", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCITemplateGetTemplateWithPlaceholders(t *testing.T) {
	app, handler := setupCITemplateTest()
	app.Get("/ci-templates/:framework", handler.GetTemplate)

	req := httptest.NewRequest("GET", "/ci-templates/go?project=myproj&service=mysvc", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if strings.Contains(body, "<your-project>") {
		t.Error("Expected <your-project> to be replaced")
	}
	if strings.Contains(body, "<your-service>") {
		t.Error("Expected <your-service> to be replaced")
	}
	if !strings.Contains(body, "myproj") {
		t.Error("Expected 'myproj' in output")
	}
	if !strings.Contains(body, "mysvc") {
		t.Error("Expected 'mysvc' in output")
	}
}
