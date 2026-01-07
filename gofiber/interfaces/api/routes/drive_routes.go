package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupDriveRoutes(router fiber.Router, h *handlers.Handlers) {
	drive := router.Group("/drive")

	// Public webhook endpoint (no auth required)
	drive.Post("/webhook", h.Drive.Webhook)

	// Thumbnail endpoint with query token support (for browser img src)
	// Must be registered BEFORE protected group to avoid middleware conflicts
	drive.Get("/thumbnail/:driveFileId", middleware.ProtectedWithQueryToken(), h.Drive.GetThumbnail)

	// OAuth endpoints (partial auth - user must be logged in)
	drive.Get("/connect", middleware.Protected(), h.Drive.Connect)
	drive.Get("/callback", h.Drive.Callback) // No auth - handles OAuth redirect

	// Protected endpoints (require auth header)
	drive.Get("/status", middleware.Protected(), h.Drive.Status)
	drive.Post("/disconnect", middleware.Protected(), h.Drive.Disconnect)
	drive.Get("/folders", middleware.Protected(), h.Drive.ListFolders)
	drive.Post("/root-folder", middleware.Protected(), h.Drive.SetRootFolder)
	drive.Post("/sync", middleware.Protected(), h.Drive.StartSync)
	drive.Get("/sync/status", middleware.Protected(), h.Drive.GetSyncStatus)
	drive.Get("/photos", middleware.Protected(), h.Drive.GetPhotos)
	drive.Post("/download", middleware.Protected(), h.Drive.DownloadPhotos)
}
