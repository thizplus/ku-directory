package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupFileRoutes(api fiber.Router, h *handlers.Handlers) {
	files := api.Group("/files")
	files.Use(middleware.Protected())
	files.Post("/upload", h.FileHandler.UploadFile)
	files.Get("/", middleware.AdminOnly(), h.FileHandler.ListFiles)
	files.Get("/my", h.FileHandler.GetUserFiles)
	files.Get("/:id", h.FileHandler.GetFile)
	files.Delete("/:id", middleware.OwnerOnly(), h.FileHandler.DeleteFile)
}