package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	Username  string `json:"username" validate:"required,min=3,max=20,alphanum"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
	FirstName string `json:"firstName" validate:"required,min=1,max=50"`
	LastName  string `json:"lastName" validate:"required,min=1,max=50"`
}

type UpdateUserRequest struct {
	FirstName string `json:"firstName" validate:"omitempty,min=1,max=50"`
	LastName  string `json:"lastName" validate:"omitempty,min=1,max=50"`
	Avatar    string `json:"avatar" validate:"omitempty,url,max=500"`
}

type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Avatar    string     `json:"avatar"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"isActive"`
	Provider  string     `json:"provider"`
	LastLogin *time.Time `json:"lastLogin,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`

	// Gemini AI settings
	GeminiAPIKey string `json:"geminiApiKey,omitempty"`
	GeminiModel  string `json:"geminiModel,omitempty"`
}

// GeminiSettingsRequest for updating Gemini API settings
type GeminiSettingsRequest struct {
	APIKey string `json:"apiKey"`
	Model  string `json:"model"`
}

type UserListResponse struct {
	Users []UserResponse `json:"users"`
	Meta  PaginationMeta `json:"meta"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=NewPassword"`
}
