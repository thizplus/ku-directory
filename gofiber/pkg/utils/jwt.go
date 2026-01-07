package utils

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	ErrMissingToken = errors.New("missing token")
)

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

type UserContext struct {
	ID       uuid.UUID
	Username string
	Email    string
	Role     string
}

func ValidateTokenStringToUUID(tokenString, jwtSecret string) (*UserContext, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	// Remove "Bearer " prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &UserContext{
		ID:       userID,
		Username: claims.Username,
		Email:    claims.Email,
		Role:     claims.Role,
	}, nil
}

func ExtractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

func GetUserFromContext(c *fiber.Ctx) (*UserContext, error) {
	log.Println("🔍 Checking user context...")

	user := c.Locals("user")

	if user == nil {
		log.Println("❌ User not found in context")
		return nil, errors.New("user not found in context")
	}

	log.Printf("✅ Found user in context: %+v (type: %T)\n", user, user)

	userCtx, ok := user.(*UserContext)
	if !ok {
		log.Printf("❌ Invalid user context type: %T\n", user)
		return nil, errors.New("invalid user context type")
	}

	log.Printf("✅ User context valid: ID=%s, Email=%s\n", userCtx.ID, userCtx.Email)
	return userCtx, nil
}
