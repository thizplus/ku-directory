package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupUserRoutes(api fiber.Router, h *handlers.Handlers) {
	users := api.Group("/users")
	users.Use(middleware.Protected())
	users.Get("/profile", h.UserHandler.GetProfile)
	users.Put("/profile", h.UserHandler.UpdateProfile)
	users.Put("/gemini-settings", h.UserHandler.UpdateGeminiSettings)
	users.Delete("/profile", h.UserHandler.DeleteUser)
	users.Get("/", middleware.AdminOnly(), h.UserHandler.ListUsers)
}