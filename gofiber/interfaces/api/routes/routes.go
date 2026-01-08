package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/pkg/config"
)

func SetupRoutes(app *fiber.App, h *handlers.Handlers, cfg *config.Config) {
	// Setup health and root routes
	SetupHealthRoutes(app)

	// API version group
	api := app.Group("/api/v1")

	// Setup all route groups
	SetupAuthRoutes(api, h, &cfg.RateLimit)
	SetupUserRoutes(api, h)
	SetupTaskRoutes(api, h)
	SetupFileRoutes(api, h)
	SetupJobRoutes(api, h)
	SetupDriveRoutes(api, h)
	SetupFaceRoutes(api, h)
	SetupNewsRoutes(api, h)
	SetupSharedFolderRoutes(api, h)
	SetupLogRoutes(api, h)

	// Setup WebSocket routes (needs app, not api group)
	SetupWebSocketRoutes(app)
}