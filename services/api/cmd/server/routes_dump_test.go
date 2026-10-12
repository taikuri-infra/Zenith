package main

import (
	"bytes"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestDumpRoutes_SortedDedupedNoHeadNoUse(t *testing.T) {
	app := fiber.New()
	noop := func(c *fiber.Ctx) error { return nil }
	app.Use(noop)
	app.Post("/b", noop)
	app.Get("/b", noop) // Fiber also registers HEAD for GET
	app.Get("/a", noop)
	g := app.Group("/api/v1")
	g.Delete("/apps/:id", noop)

	var buf bytes.Buffer
	if err := dumpRoutes(app, &buf); err != nil {
		t.Fatalf("dumpRoutes: %v", err)
	}

	want := "GET /a\nDELETE /api/v1/apps/:id\nGET /b\nPOST /b\n"
	if got := buf.String(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
