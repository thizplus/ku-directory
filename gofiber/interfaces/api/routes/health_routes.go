package routes

import (
	"github.com/gofiber/fiber/v2"
)

func SetupHealthRoutes(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Server is running",
			"service": "GoFiber Template API",
		})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to GoFiber Template API",
			"version": "1.0.0",
			"docs":    "/api/v1",
			"health":  "/health",
		})
	})
}
