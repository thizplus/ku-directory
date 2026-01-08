package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
	"gofiber-template/pkg/config"
)

func SetupAuthRoutes(api fiber.Router, h *handlers.Handlers, rateLimitCfg *config.RateLimitConfig) {
	auth := api.Group("/auth")

	// Apply stricter rate limiting to all auth endpoints
	authLimiter := middleware.AuthRateLimiter(rateLimitCfg)

	// Traditional auth (optional - keep for admin/testing)
	auth.Post("/register", authLimiter, h.UserHandler.Register)
	auth.Post("/login", authLimiter, h.UserHandler.Login)

	// Google OAuth
	auth.Get("/google", authLimiter, h.AuthHandler.GoogleLogin)
	auth.Get("/google/callback", h.AuthHandler.GoogleCallback)

	// Protected routes
	auth.Get("/me", middleware.Protected(), h.AuthHandler.GetCurrentUser)
	auth.Post("/logout", h.AuthHandler.Logout)
}