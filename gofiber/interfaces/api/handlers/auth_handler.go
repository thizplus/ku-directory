package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/logger"
	"gofiber-template/pkg/utils"
)

type AuthHandler struct {
	authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// GoogleLogin redirects to Google OAuth
func (h *AuthHandler) GoogleLogin(c *fiber.Ctx) error {
	logger.Auth("LOGIN_START", "User initiating Google OAuth login", map[string]interface{}{
		"ip":         c.IP(),
		"user_agent": c.Get("User-Agent"),
	})

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		logger.AuthError("LOGIN_ERROR", "Failed to generate state", err, nil)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate state", err)
	}

	// Store state in cookie for validation
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
	})

	// Get the redirect URL from query param (for frontend redirect after login)
	redirectURL := c.Query("redirect", "/")
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_redirect",
		Value:    redirectURL,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
	})

	// Redirect to Google
	authURL := h.authService.GetGoogleAuthURL(state)

	logger.Auth("LOGIN_REDIRECT", "Redirecting to Google OAuth", map[string]interface{}{
		"redirect_url": redirectURL,
		"state":        state[:10] + "...",
	})

	return c.Redirect(authURL)
}

// GoogleCallback handles the OAuth callback from Google
func (h *AuthHandler) GoogleCallback(c *fiber.Ctx) error {
	logger.Auth("CALLBACK_START", "Received OAuth callback from Google", map[string]interface{}{
		"ip":    c.IP(),
		"query": c.OriginalURL(),
	})

	// Verify state
	state := c.Query("state")
	storedState := c.Cookies("oauth_state")

	if state == "" || state != storedState {
		logger.AuthError("CALLBACK_ERROR", "Invalid state parameter", nil, map[string]interface{}{
			"state_match": state == storedState,
		})
		return c.Redirect("/?error=invalid_state")
	}

	// Clear state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	})

	// Check for error from Google
	if errMsg := c.Query("error"); errMsg != "" {
		logger.AuthError("CALLBACK_ERROR", "Google returned error", nil, map[string]interface{}{
			"google_error": errMsg,
		})
		return c.Redirect(fmt.Sprintf("/?error=%s", errMsg))
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		logger.AuthError("CALLBACK_ERROR", "Missing authorization code", nil, nil)
		return c.Redirect("/?error=missing_code")
	}

	logger.Auth("CALLBACK_EXCHANGE", "Exchanging code for token", map[string]interface{}{
		"code_length": len(code),
	})

	// Exchange code for token and get user
	token, user, err := h.authService.HandleGoogleCallback(c.Context(), code)
	if err != nil {
		logger.AuthError("CALLBACK_ERROR", "Failed to exchange code", err, nil)
		return c.Redirect(fmt.Sprintf("/?error=auth_failed&message=%s", err.Error()))
	}

	logger.Auth("CALLBACK_SUCCESS", "User authenticated successfully", map[string]interface{}{
		"user_id":    user.ID.String(),
		"user_email": user.Email,
		"user_name":  user.FirstName + " " + user.LastName,
	})

	// Get redirect URL
	redirectURL := c.Cookies("oauth_redirect", "/")
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_redirect",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	})

	// Set auth token in cookie
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(7 * 24 * time.Hour), // 7 days
		HTTPOnly: true,
		Secure:   false, // Set to true in production
		SameSite: "Lax",
	})

	// Redirect to frontend with token (for SPA)
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	frontendURL := fmt.Sprintf("%s/auth/callback?token=%s&redirect=%s", baseURL, token, redirectURL)

	logger.Auth("CALLBACK_REDIRECT", "Redirecting to frontend", map[string]interface{}{
		"redirect_url": redirectURL,
	})

	return c.Redirect(frontendURL)
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	// Get full user from auth service
	user, err := h.authService.GetCurrentUser(c.Context(), c.Get("Authorization")[7:]) // Remove "Bearer "
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get user", err)
	}

	userResponse := dto.UserToUserResponse(user)
	_ = userCtx // userCtx available if needed for additional checks
	return utils.SuccessResponse(c, "User retrieved successfully", userResponse)
}

// Logout clears the auth cookie
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	})

	return utils.SuccessResponse(c, "Logged out successfully", nil)
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
