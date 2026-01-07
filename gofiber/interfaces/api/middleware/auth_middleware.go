package middleware

import (
	"gofiber-template/pkg/utils"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

// Protected middleware validates JWT tokens and sets user context
func Protected() fiber.Handler {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.UnauthorizedResponse(c, "Missing authorization header")
		}

		// Extract token from header
		token := utils.ExtractTokenFromHeader(authHeader)
		if token == "" {
			return utils.UnauthorizedResponse(c, "Invalid authorization header format")
		}

		// Validate token and get user context
		userCtx, err := utils.ValidateTokenStringToUUID(token, jwtSecret)
		if err != nil {
			log.Printf("❌ Token validation failed: %v", err)
			switch err {
			case utils.ErrExpiredToken:
				return utils.UnauthorizedResponse(c, "Token has expired")
			case utils.ErrInvalidToken:
				return utils.UnauthorizedResponse(c, "Invalid token")
			case utils.ErrMissingToken:
				return utils.UnauthorizedResponse(c, "Missing token")
			default:
				return utils.UnauthorizedResponse(c, "Token validation failed")
			}
		}

		log.Printf("✅ Token validated for user: %s (%s)", userCtx.Email, userCtx.ID)

		// Set user context in fiber locals
		c.Locals("user", userCtx)

		return c.Next()
	}
}

// RequireRole middleware checks if user has specific role
func RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := utils.GetUserFromContext(c)
		if err != nil {
			return utils.UnauthorizedResponse(c, "User not authenticated")
		}

		if user.Role != role {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Insufficient permissions",
				"error":   "Access denied",
			})
		}

		return c.Next()
	}
}

// AdminOnly middleware ensures only admin users can access
func AdminOnly() fiber.Handler {
	return RequireRole("admin")
}

// OwnerOnly middleware checks if user is the owner of the resource
func OwnerOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := utils.GetUserFromContext(c)
		if err != nil {
			return utils.UnauthorizedResponse(c, "User not authenticated")
		}

		c.Locals("requireOwnership", true)
		c.Locals("ownerUserID", user.ID)

		return c.Next()
	}
}

// Optional middleware that doesn't require authentication but sets user context if token is present
func Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		token := utils.ExtractTokenFromHeader(authHeader)
		if token == "" {
			return c.Next()
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		userCtx, err := utils.ValidateTokenStringToUUID(token, jwtSecret)
		if err != nil {
			return c.Next()
		}

		c.Locals("user", userCtx)
		return c.Next()
	}
}

// OptionalWithQueryToken middleware that checks both header and query parameter for token
// Used for WebSocket connections where Authorization header can't be sent
func OptionalWithQueryToken() fiber.Handler {
	jwtSecret := os.Getenv("JWT_SECRET")

	return func(c *fiber.Ctx) error {
		var token string

		// First try Authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			token = utils.ExtractTokenFromHeader(authHeader)
		}

		// If no header token, try query parameter
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			return c.Next() // No token, continue as anonymous
		}

		// Validate token
		userCtx, err := utils.ValidateTokenStringToUUID(token, jwtSecret)
		if err != nil {
			return c.Next() // Invalid token, continue as anonymous
		}

		c.Locals("user", userCtx)
		return c.Next()
	}
}

// ProtectedWithQueryToken middleware validates JWT tokens from header OR query parameter
// This is useful for image/file endpoints where browser can't send Authorization header
func ProtectedWithQueryToken() fiber.Handler {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return func(c *fiber.Ctx) error {
		var token string

		// First try Authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			token = utils.ExtractTokenFromHeader(authHeader)
			log.Printf("🔑 Token from header: %d chars", len(token))
		}

		// If no header token, try query parameter
		if token == "" {
			token = c.Query("token")
			if token != "" {
				log.Printf("🔑 Token from query: %d chars", len(token))
			} else {
				log.Printf("❌ No token in header or query")
			}
		}

		if token == "" {
			return utils.UnauthorizedResponse(c, "Missing authorization")
		}

		// Validate token and get user context
		userCtx, err := utils.ValidateTokenStringToUUID(token, jwtSecret)
		if err != nil {
			switch err {
			case utils.ErrExpiredToken:
				return utils.UnauthorizedResponse(c, "Token has expired")
			case utils.ErrInvalidToken:
				return utils.UnauthorizedResponse(c, "Invalid token")
			default:
				return utils.UnauthorizedResponse(c, "Token validation failed")
			}
		}

		c.Locals("user", userCtx)
		return c.Next()
	}
}
