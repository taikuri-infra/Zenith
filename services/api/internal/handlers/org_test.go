package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/gofiber/fiber/v2"
)

func orgTestApp(t *testing.T, userID string) (*fiber.App, string) {
	t.Helper()
	projects := memory.NewMemoryProjectRepository()
	p, err := projects.CreateProject(context.Background(), "u1", "P", "p", "")
	if err != nil {
		t.Fatal(err)
	}
	h := NewOrgHandler(memory.NewMemoryOrgRepository(), projects)
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", userID); return c.Next() })
	app.Get("/projects/:projectId/orgs", h.List)
	app.Post("/projects/:projectId/orgs", h.Create)
	app.Delete("/orgs/:orgId", h.Delete)
	return app, p.ID
}

func TestOrgListCreatesDefault(t *testing.T) {
	app, pid := orgTestApp(t, "u1")
	resp, _ := app.Test(httptest.NewRequest("GET", "/projects/"+pid+"/orgs", nil))
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestOrgForeignProjectForbidden(t *testing.T) {
	app, pid := orgTestApp(t, "someone-else")
	resp, _ := app.Test(httptest.NewRequest("GET", "/projects/"+pid+"/orgs", nil))
	if resp.StatusCode != 403 {
		t.Fatalf("status %d, want 403", resp.StatusCode)
	}
}
