package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupActivityLogRoutes(api fiber.Router, h *handlers.Handlers) {
	// Skip if handler not initialized
	if h.ActivityLog == nil {
		return
	}

	// Protected routes for activity logs
	activity := api.Group("/activity-logs", middleware.Protected())

	// Get activity types (no auth needed for this)
	activity.Get("/types", h.ActivityLog.GetActivityTypes)

	// Get recent logs
	activity.Get("/recent", h.ActivityLog.GetRecentActivityLogs)

	// Get logs by folder
	activity.Get("/", h.ActivityLog.GetActivityLogs)
}
