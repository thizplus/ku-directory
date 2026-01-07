package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupSharedFolderRoutes(api fiber.Router, h *handlers.Handlers) {
	// Skip if handler not initialized
	if h.SharedFolder == nil {
		return
	}

	// Protected routes for shared folders
	folders := api.Group("/folders", middleware.Protected())

	// Folder management
	folders.Get("/", h.SharedFolder.ListFolders)
	folders.Get("/:id", h.SharedFolder.GetFolder)
	folders.Post("/", h.SharedFolder.AddFolder)
	folders.Delete("/:id", h.SharedFolder.RemoveFolder)

	// Folder operations
	folders.Post("/:id/sync", h.SharedFolder.TriggerSync)
	folders.Post("/:id/webhook", h.SharedFolder.RegisterWebhook)
	folders.Get("/:id/photos", h.SharedFolder.GetPhotos)
	folders.Get("/:id/subfolders", h.SharedFolder.GetSubFolders)
}
