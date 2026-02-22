package docs

import "github.com/gofiber/fiber/v2"

const scalarHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Zenith API Reference</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
</head>
<body>
  <script id="api-reference" data-url="/docs/openapi.yaml"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

// RegisterRoutes mounts the API documentation endpoints.
func RegisterRoutes(app *fiber.App) {
	spec := SpecYAML()

	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(scalarHTML)
	})

	app.Get("/docs/openapi.yaml", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml")
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Send(spec)
	})
}
