package scalar

import (
	"html/template"

	"github.com/gofiber/fiber/v2"
	"github.com/swaggo/swag"
)

// Config for Scalar API Reference
type Config struct {
	Title string
	Theme string // default, moon, purple, solarized, bluePlanet, deepSpace, saturn, kepler, mars, none
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Title: "API Reference",
		Theme: "deepSpace",
	}
}

const scalarTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <script id="api-reference" data-url="/docs/openapi.json"></script>
    <script>
        document.getElementById('api-reference').dataset.configuration = JSON.stringify({
            theme: '{{.Theme}}'
        });
    </script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

// New creates Scalar API Reference middleware
func New(config ...Config) fiber.Handler {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.Title == "" {
		cfg.Title = "API Reference"
	}
	if cfg.Theme == "" {
		cfg.Theme = "deepSpace"
	}

	tmpl := template.Must(template.New("scalar").Parse(scalarTemplate))

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Serve OpenAPI JSON
		if path == "/docs/openapi.json" || c.Params("*") == "openapi.json" {
			doc, err := swag.ReadDoc()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			c.Set("Content-Type", "application/json")
			return c.SendString(doc)
		}

		// Serve Scalar HTML
		c.Set("Content-Type", "text/html")
		return tmpl.Execute(c.Response().BodyWriter(), cfg)
	}
}

// SetupRoutes adds both the docs UI and OpenAPI JSON routes
func SetupRoutes(app *fiber.App, config ...Config) {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	tmpl := template.Must(template.New("scalar").Parse(scalarTemplate))

	// Serve OpenAPI JSON
	app.Get("/docs/openapi.json", func(c *fiber.Ctx) error {
		doc, err := swag.ReadDoc()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", "application/json")
		return c.SendString(doc)
	})

	// Serve Scalar HTML
	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return tmpl.Execute(c.Response().BodyWriter(), cfg)
	})
	app.Get("/docs/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return tmpl.Execute(c.Response().BodyWriter(), cfg)
	})
}
