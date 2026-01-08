package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
)

// SetupLogRoutes sets up log-related routes
func SetupLogRoutes(router fiber.Router, h *handlers.Handlers) {
	admin := router.Group("/admin")

	// Log endpoints (protected by admin token in header or query param)
	admin.Get("/logs", h.Log.GetLogs)
	admin.Get("/logs/files", h.Log.GetLogFiles)
	admin.Get("/logs/stats", h.Log.GetLogStats)
	admin.Get("/logs/folder/:id", h.Log.GetFolderLogs) // Get logs by folder ID
}
