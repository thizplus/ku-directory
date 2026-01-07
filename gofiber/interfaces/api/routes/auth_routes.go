package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupAuthRoutes(api fiber.Router, h *handlers.Handlers) {
	auth := api.Group("/auth")

	// Traditional auth (optional - keep for admin/testing)
	auth.Post("/register", h.UserHandler.Register)
	auth.Post("/login", h.UserHandler.Login)

	// Google OAuth
	auth.Get("/google", h.AuthHandler.GoogleLogin)
	auth.Get("/google/callback", h.AuthHandler.GoogleCallback)

	// Protected routes
	auth.Get("/me", middleware.Protected(), h.AuthHandler.GetCurrentUser)
	auth.Post("/logout", h.AuthHandler.Logout)
}