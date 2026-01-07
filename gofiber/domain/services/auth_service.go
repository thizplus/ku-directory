package services

import (
	"context"

	"gofiber-template/domain/models"
)

type GoogleUserInfo struct {
	ID         string
	Email      string
	Name       string
	GivenName  string
	FamilyName string
	Picture    string
}

type AuthService interface {
	// GetGoogleAuthURL returns the Google OAuth authorization URL
	GetGoogleAuthURL(state string) string

	// HandleGoogleCallback processes the Google OAuth callback
	HandleGoogleCallback(ctx context.Context, code string) (token string, user *models.User, err error)

	// GetCurrentUser returns the current authenticated user from token
	GetCurrentUser(ctx context.Context, token string) (*models.User, error)
}
