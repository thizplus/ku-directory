package routes

import (
	"github.com/gofiber/fiber/v2"

	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupFaceRoutes(router fiber.Router, h *handlers.Handlers) {
	faces := router.Group("/faces")

	// Detect faces (no auth required - just detection, no DB access)
	faces.Post("/detect", h.Face.DetectFaces)

	// All routes below require authentication
	faces.Use(middleware.Protected())

	// Face search endpoints
	faces.Post("/search/image", h.Face.SearchByImage)   // Search by uploading image
	faces.Post("/search/face", h.Face.SearchByFaceID)   // Search by existing face ID

	// Get faces
	faces.Get("/", h.Face.GetFaces)                     // Get all faces (paginated)
	faces.Get("/photo/:photo_id", h.Face.GetFacesByPhoto) // Get faces in a photo

	// Stats
	faces.Get("/stats", h.Face.GetProcessingStats)      // Get processing stats
}
