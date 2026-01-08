package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"gofiber-template/pkg/config"
)

// RateLimiter returns a general rate limiting middleware
func RateLimiter(cfg *config.RateLimitConfig) fiber.Handler {
	if !cfg.Enabled {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	return limiter.New(limiter.Config{
		Max:        cfg.MaxRequests,
		Expiration: time.Duration(cfg.WindowSeconds) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	})
}

// AuthRateLimiter returns a stricter rate limiting middleware for auth endpoints
func AuthRateLimiter(cfg *config.RateLimitConfig) fiber.Handler {
	if !cfg.Enabled {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	return limiter.New(limiter.Config{
		Max:        cfg.AuthMaxRequests,
		Expiration: time.Duration(cfg.AuthWindowSeconds) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_RATE_LIMIT_EXCEEDED",
					"message": "Too many authentication attempts. Please try again later.",
				},
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	})
}
