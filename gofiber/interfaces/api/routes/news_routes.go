package routes

import (
	"github.com/gofiber/fiber/v2"

	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupNewsRoutes(router fiber.Router, h *handlers.Handlers) {
	news := router.Group("/news")

	// All news routes require authentication
	news.Use(middleware.Protected())

	// Generate news from photos
	news.Post("/generate", h.News.GenerateNews)
}
